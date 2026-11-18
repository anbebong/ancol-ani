package main

import (
	"bytes"
	"encoding/json"
	"image/gif"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const maxGifBytes = 5 << 20 // 5MB

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var creds credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if !usernamePattern.MatchString(creds.Username) {
		http.Error(w, "username phai 3-32 ky tu, chi gom chu/so/_", http.StatusBadRequest)
		return
	}
	if len(creds.Password) < 6 {
		http.Error(w, "password phai it nhat 6 ky tu", http.StatusBadRequest)
		return
	}

	hash, err := hashPassword(creds.Password)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	userID, err := s.store.CreateUser(creds.Username, hash)
	if err == ErrDuplicate {
		http.Error(w, "username da ton tai", http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	s.respondWithToken(w, userID)
}

func (s *server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var creds credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	user, err := s.store.GetUserByUsername(creds.Username)
	if err == ErrNotFound || (err == nil && !checkPassword(user.PasswordHash, creds.Password)) {
		http.Error(w, "sai username hoac password", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	s.respondWithToken(w, user.ID)
}

func (s *server) respondWithToken(w http.ResponseWriter, userID int64) {
	token, err := s.issueToken(userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func (s *server) handleMe(w http.ResponseWriter, r *http.Request, userID int64) {
	user, err := s.store.GetUserByID(userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"username": user.Username,
		"isAdmin":  user.IsAdmin,
		"gifQuota": user.GifQuota,
	})
}

func (s *server) handleLogout(w http.ResponseWriter, r *http.Request, _ int64) {
	// requireAuth already validated the token; re-parse just to get the
	// jti so we know which session row to revoke.
	raw := r.Header.Get("Authorization")[len("Bearer "):]
	_, jti, err := s.verifyToken(raw)
	if err == nil {
		s.store.RevokeSession(jti)
	}
	w.WriteHeader(http.StatusNoContent)
}

const defaultPageSize = 10
const maxPageSize = 50

// parsePagination reads ?page= (1-based, default 1) and ?pageSize=
// (default 10, capped at 50 so a client can't force a huge unpaginated
// dump by passing pageSize=100000).
func parsePagination(r *http.Request) (page, pageSize, offset int) {
	page = 1
	if v, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && v > 0 {
		page = v
	}
	pageSize = defaultPageSize
	if v, err := strconv.Atoi(r.URL.Query().Get("pageSize")); err == nil && v > 0 {
		pageSize = v
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize, (page - 1) * pageSize
}

func (s *server) handleListGifs(w http.ResponseWriter, r *http.Request, userID int64) {
	page, pageSize, offset := parsePagination(r)
	gifs, total, err := s.store.ListGifs(userID, pageSize, offset)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"items": gifs, "total": total, "page": page, "pageSize": pageSize,
	})
}

func (s *server) handleCreateGif(w http.ResponseWriter, r *http.Request, userID int64) {
	data, err := readGifUpload(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id, err := s.store.CreateGif(userID, data)
	if err == ErrQuota {
		http.Error(w, "da dat toi da so GIF duoc phep, xoa 1 cai truoc khi them moi", http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

// dispatchMyGifsSub handles /me/gifs/{id} and /me/gifs/{id}/{action}.
func (s *server) dispatchMyGifsSub(w http.ResponseWriter, r *http.Request, userID int64) {
	rest := strings.TrimPrefix(r.URL.Path, "/me/gifs/")
	parts := strings.SplitN(rest, "/", 2)
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	action := ""
	if len(parts) == 2 {
		action = parts[1]
	}

	switch {
	case action == "" && r.Method == http.MethodDelete:
		s.handleDeleteGif(w, r, userID, id)
	case action == "image" && r.Method == http.MethodGet:
		s.handleGetGifImage(w, r, userID, id)
	case action == "share" && r.Method == http.MethodPost:
		s.handleShareGif(w, r, userID, id)
	case action == "unshare" && r.Method == http.MethodPost:
		s.handleUnshareGif(w, r, userID, id)
	default:
		http.NotFound(w, r)
	}
}

func (s *server) handleDeleteGif(w http.ResponseWriter, r *http.Request, userID, gifID int64) {
	if err := s.store.DeleteGif(userID, gifID); err == ErrNotFound {
		http.Error(w, "not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleGetGifImage(w http.ResponseWriter, r *http.Request, userID, gifID int64) {
	data, err := s.store.GetGifData(userID, gifID)
	if err == ErrNotFound {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/gif")
	w.Write(data)
}

func (s *server) handleShareGif(w http.ResponseWriter, r *http.Request, userID, gifID int64) {
	var body struct {
		Name string `json:"name"`
		Tags string `json:"tags"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" {
		http.Error(w, "phai dat ten cho gif truoc khi chia se", http.StatusBadRequest)
		return
	}

	slug := randomHex(8)
	err := s.store.ShareGif(userID, gifID, slug, body.Name, normalizeTags(body.Tags))
	if err == ErrNotFound {
		http.Error(w, "khong tim thay gif", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"slug": slug})
}

func (s *server) handleUnshareGif(w http.ResponseWriter, r *http.Request, userID, gifID int64) {
	if err := s.store.UnshareGif(userID, gifID); err == ErrNotFound {
		http.Error(w, "not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleMyShareHistory(w http.ResponseWriter, r *http.Request, userID int64) {
	entries, err := s.store.ListShareHistory(userID, 50)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

// handlePublicList supports ?q= (free-text over name+tags) and ?user=
// (exact publisher username), so anyone can search by tag or by who
// shared it, plus ?page=/?pageSize= pagination.
func (s *server) handlePublicList(w http.ResponseWriter, r *http.Request, _ int64) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	user := strings.TrimSpace(r.URL.Query().Get("user"))
	page, pageSize, offset := parsePagination(r)

	entries, total, err := s.store.ListPublic(pageSize, offset, q, user)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"items": entries, "total": total, "page": page, "pageSize": pageSize,
	})
}

// normalizeTags trims whitespace around each comma-separated tag and
// drops empties, so "a, ,b" becomes "a,b" - keeps LIKE search and display
// predictable regardless of how the user typed it.
func normalizeTags(raw string) string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, ",")
}

func (s *server) handlePublicItem(w http.ResponseWriter, r *http.Request, _ int64) {
	slug := r.URL.Path[len("/public/"):]
	data, err := s.store.GetPublicBySlug(slug)
	if err == ErrNotFound {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/gif")
	w.Write(data)
}

func (s *server) handleAdminListUsers(w http.ResponseWriter, r *http.Request, _ int64) {
	users, err := s.store.ListUsers()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// dispatchAdminUsersSub handles /admin/users/{id} (DELETE) and
// /admin/users/{id}/quota (PATCH).
func (s *server) dispatchAdminUsersSub(w http.ResponseWriter, r *http.Request, callerID int64) {
	rest := strings.TrimPrefix(r.URL.Path, "/admin/users/")
	parts := strings.SplitN(rest, "/", 2)
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	action := ""
	if len(parts) == 2 {
		action = parts[1]
	}

	switch {
	case action == "" && r.Method == http.MethodDelete:
		s.handleAdminDeleteUser(w, r, id)
	case action == "quota" && r.Method == http.MethodPatch:
		s.handleAdminSetQuota(w, r, id)
	default:
		http.NotFound(w, r)
	}
}

func (s *server) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request, id int64) {
	if err := s.store.DeleteUser(id); err == ErrNotFound {
		http.Error(w, "not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleAdminSetQuota(w http.ResponseWriter, r *http.Request, id int64) {
	var body struct {
		Quota int64 `json:"quota"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Quota < 0 || body.Quota > 50 {
		http.Error(w, "quota khong hop le (0-50)", http.StatusBadRequest)
		return
	}
	if err := s.store.SetGifQuota(id, body.Quota); err == ErrNotFound {
		http.Error(w, "not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleAdminUnpublish(w http.ResponseWriter, r *http.Request, _ int64) {
	slug := r.URL.Path[len("/admin/public/"):]
	if err := s.store.AdminUnpublish(slug); err == ErrNotFound {
		http.Error(w, "not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// readGifUpload reads a multipart "gif" field and sanity-checks it by
// decoding + re-encoding through the standard gif package, same approach
// the app itself uses for local uploads.
func readGifUpload(r *http.Request) ([]byte, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, maxGifBytes)
	if err := r.ParseMultipartForm(maxGifBytes); err != nil {
		return nil, errTooLarge
	}
	file, _, err := r.FormFile("gif")
	if err != nil {
		return nil, errMissingFile
	}
	defer file.Close()

	raw, err := io.ReadAll(file)
	if err != nil {
		return nil, errTooLarge
	}

	decoded, err := gif.DecodeAll(bytes.NewReader(raw))
	if err != nil {
		return nil, errInvalidGif
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, decoded); err != nil {
		return nil, errInvalidGif
	}
	return buf.Bytes(), nil
}

type httpErr struct{ msg string }

func (e httpErr) Error() string { return e.msg }

var (
	errTooLarge    = httpErr{"file qua nang (toi da 5MB)"}
	errMissingFile = httpErr{"missing file"}
	errInvalidGif  = httpErr{"invalid gif"}
)
