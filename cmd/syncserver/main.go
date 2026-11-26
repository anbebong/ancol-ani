// syncserver is the backend for ancol-ani's account-based sync & public
// sharing. It's a standalone service (not something that runs on a
// client's machine) - run it locally for development, or point it at a
// real DSN/driver to deploy it for real; business logic doesn't change
// either way (see store.go).
//
// Each user gets a configurable quota of GIF slots (default 3, editable
// per-user by an admin). There's no server-side "current/active" gif -
// a client syncs by picking one slot from the user's library (GET
// /me/gifs, then GET /me/gifs/{id}/image) and deciding what to do with
// it itself. Any gif slot can independently be marked public, which
// hands out a random slug any logged-in user (not just the owner) can
// browse.
//
// Run for local testing:
//
//	go run ./cmd/syncserver
package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// The web portal (accounts, GIF management, admin, public gallery) is its
// own Vite project under portal/, built separately (cd portal && npm run
// build) and embedded here as static files - same pattern as the desktop
// app's local settings page (see /settings-ui/ at the project root).
//
//go:embed portal/dist
var webFiles embed.FS

type server struct {
	store     *Store
	jwtSecret []byte
	limits    *limiter
	// ALLOW_REGISTER=false closes sign-up (existing accounts keep working).
	allowRegister bool
}

func main() {
	driver := envOr("DB_DRIVER", "sqlite")
	dsn := envOr("DB_DSN", "file:data/ancol-ani.db?_pragma=busy_timeout(5000)")
	addr := envOr("ADDR", "0.0.0.0:9090")
	secret := envOr("JWT_SECRET", "")

	makeAdmin := flag.String("make-admin", "", "username to grant admin rights to, then exit")
	setQuota := flag.String("set-quota", "", "username=N: set that user's GIF quota (0-50), then exit")
	flag.Parse()

	store, err := OpenStore(driver, dsn)
	if err != nil {
		log.Fatalf("cannot open store (%s): %v", driver, err)
	}

	if *makeAdmin != "" {
		if err := store.SetAdmin(*makeAdmin, true); err != nil {
			log.Fatalf("cannot grant admin to %q: %v", *makeAdmin, err)
		}
		fmt.Printf("%q is now an admin\n", *makeAdmin)
		return
	}

	// Same idea as -make-admin, for servers with no admin account yet (the
	// portal's quota field needs one). Safe to run while the server is up.
	if *setQuota != "" {
		username, rawN, ok := strings.Cut(*setQuota, "=")
		n, err := strconv.ParseInt(rawN, 10, 64)
		if !ok || err != nil || n < 0 || n > 50 {
			log.Fatalf("-set-quota expects username=N with N in 0-50, got %q", *setQuota)
		}
		user, err := store.GetUserByUsername(username)
		if err != nil {
			log.Fatalf("cannot find user %q: %v", username, err)
		}
		if err := store.SetGifQuota(user.ID, n); err != nil {
			log.Fatalf("cannot set quota for %q: %v", username, err)
		}
		fmt.Printf("%q quota: %d -> %d\n", username, user.GifQuota, n)
		return
	}

	if secret == "" {
		log.Println("WARNING: JWT_SECRET not set, using an insecure built-in dev secret - do not deploy this for real without setting one")
		secret = "dev-only-insecure-secret-change-me"
	}

	s := &server{
		store:         store,
		jwtSecret:     []byte(secret),
		limits:        newLimiter(),
		allowRegister: envOr("ALLOW_REGISTER", "true") != "false",
	}
	if !s.allowRegister {
		log.Println("registration closed (ALLOW_REGISTER=false)")
	}

	webFS, err := fs.Sub(webFiles, "portal/dist")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(webFS)))

	mux.HandleFunc("/identity", s.withCORS(handleIdentity))

	mux.HandleFunc("/auth/register", s.withCORS(s.handleRegister))
	mux.HandleFunc("/auth/login", s.withCORS(s.handleLogin))
	mux.HandleFunc("/auth/logout", s.withCORS(s.requireAuth(s.handleLogout)))
	mux.HandleFunc("/auth/token", s.withCORS(s.requireAuth(s.handleIssueToken)))
	mux.HandleFunc("/me", s.withCORS(s.requireAuth(s.handleMe)))
	mux.HandleFunc("/me/password", s.withCORS(s.requireAuth(s.handleChangePassword)))


	// GIF library: up to the user's quota, each independently shareable.
	mux.HandleFunc("/me/gifs", s.withCORS(s.dispatchMyGifsRoot))
	mux.HandleFunc("/me/gifs/", s.withCORS(s.requireAuth(s.dispatchMyGifsSub)))

	mux.HandleFunc("/public", s.withCORS(s.requireAuth(s.handlePublicList)))
	mux.HandleFunc("/public/", s.withCORS(s.requireAuth(s.handlePublicItem)))

	mux.HandleFunc("/admin/users", s.withCORS(s.requireAdmin(s.handleAdminListUsers)))
	mux.HandleFunc("/admin/users/", s.withCORS(s.requireAdmin(s.dispatchAdminUsersSub)))
	mux.HandleFunc("/admin/public/", s.withCORS(s.requireAdmin(s.handleAdminUnpublish)))

	// GIF toolkit: public (no requireAuth), each behind the shared
	// concurrency+timeout middleware so a couple of heavy jobs can't tie up
	// the whole process.
	if err := os.MkdirAll(toolTmpDir, 0o755); err != nil {
		log.Fatalf("cannot create %s: %v", toolTmpDir, err)
	}
	if !ffmpegAvailable() {
		log.Println("warning: ffmpeg/ffprobe not found, /api/tools/convert video path will return 503")
	}
	mux.HandleFunc("/api/tools/convert", s.withCORS(withToolLimits(handleToolConvert)))
	mux.HandleFunc("/api/tools/resize", s.withCORS(withToolLimits(handleToolResize)))
	mux.HandleFunc("/api/tools/crop", s.withCORS(withToolLimits(handleToolCrop)))
	mux.HandleFunc("/api/tools/optimize", s.withCORS(withToolLimits(handleToolOptimize)))
	mux.HandleFunc("/api/tools/rotate", s.withCORS(withToolLimits(handleToolRotate)))
	mux.HandleFunc("/api/tools/reverse", s.withCORS(withToolLimits(handleToolReverse)))
	mux.HandleFunc("/api/tools/speed", s.withCORS(withToolLimits(handleToolSpeed)))
	mux.HandleFunc("/api/tools/split", s.withCORS(withToolLimits(handleToolSplit)))
	mux.HandleFunc("/api/tools/caption", s.withCORS(withToolLimits(handleToolCaption)))

	log.Printf("syncserver listening on %s (driver=%s)", addr, driver)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func (s *server) dispatchMyGifsRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		s.requireAuth(s.handleCreateGif)(w, r)
		return
	}
	s.requireAuth(s.handleListGifs)(w, r)
}

func (s *server) withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			return
		}
		next(w, r)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
