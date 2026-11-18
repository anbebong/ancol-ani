package main

import (
	"bytes"
	"context"
	"embed"
	"image/gif"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const clientLocalPort = "8788"

//go:embed VERSION
var rawVersion string

var appVersion = strings.TrimSpace(rawVersion)

// A large GIF (many MB) can freeze the small 70x70 WebView2 panel while it
// decodes/animates, so uploads are capped to keep the widget responsive.
const maxGifBytes = 3 << 20 // 3MB

//go:embed frontend/src/mascot.gif
var defaultGif []byte

// The local settings page (http://localhost:8788) is its own small Vite
// project under web/, built separately (cd web && npm run build) and
// embedded here as static files, instead of living as an inline HTML
// string in Go.
//
//go:embed web/dist
var webDist embed.FS

// sanitizeGif decodes and re-encodes the GIF through Go's standard gif
// package. Some encoders produce frame timing/disposal quirks that make
// Chromium's GIF decoder stall mid-animation after looping for a while;
// round-tripping through a clean encoder normalizes that away.
func sanitizeGif(raw []byte) ([]byte, error) {
	decoded, err := gif.DecodeAll(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}

	for i, delay := range decoded.Delay {
		if delay <= 0 {
			decoded.Delay[i] = 10 // 100ms fallback, same as a stalled frame being likely the bug
		}
	}

	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, decoded); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// customGifPath returns where a user-imported GIF is stored, outside the
// embedded/read-only build so it survives without a rebuild.
func customGifPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	dir = filepath.Join(dir, "ancol-ani")
	os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, "custom.gif")
}

const (
	petWidth  = 70
	petHeight = 100
	margin    = 6
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	hwnd := findWindowHandle()
	removeMinimumSizeConstraint(hwnd)
	hideFromTaskbar(ctx, hwnd)
	runtime.WindowSetSize(ctx, petWidth, petHeight)
	positionBottomRight(ctx, hwnd)
	makeClickThrough(hwnd)

	go watchForMonitorChanges(ctx, hwnd)
	go startLocalServer(ctx)
}

// Serves a single local page (http://localhost:8788): type text in, submit,
// and it shows up directly on the widget's text panel. No server/socket
// involved - this page is the only source of text.
func startLocalServer(ctx context.Context) {
	mux := http.NewServeMux()

	webFS, err := fs.Sub(webDist, "web/dist")
	if err != nil {
		panic(err)
	}
	mux.Handle("/", http.FileServer(http.FS(webFS)))

	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(appVersion))
	})

	mux.HandleFunc("/text", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "use POST", http.StatusMethodNotAllowed)
			return
		}
		text := r.FormValue("text")
		if text != "" {
			runtime.EventsEmit(ctx, "server-text", text)
		}
	})

	mux.HandleFunc("/gif", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			r.Body = http.MaxBytesReader(w, r.Body, maxGifBytes)
			if err := r.ParseMultipartForm(maxGifBytes); err != nil {
				http.Error(w, "File GIF qua nang (toi da 3MB), chon file nho hon", http.StatusRequestEntityTooLarge)
				return
			}
			file, _, err := r.FormFile("gif")
			if err != nil {
				http.Error(w, "missing file", http.StatusBadRequest)
				return
			}
			defer file.Close()

			raw, err := io.ReadAll(file)
			if err != nil {
				http.Error(w, "File GIF qua nang (toi da 3MB), chon file nho hon", http.StatusRequestEntityTooLarge)
				return
			}

			clean, err := sanitizeGif(raw)
			if err != nil {
				http.Error(w, "File GIF loi, khong doc duoc: "+err.Error(), http.StatusBadRequest)
				return
			}

			if err := os.WriteFile(customGifPath(), clean, 0o644); err != nil {
				http.Error(w, "cannot save", http.StatusInternalServerError)
				return
			}

			runtime.EventsEmit(ctx, "gif-changed", "")
			return
		}

		w.Header().Set("Content-Type", "image/gif")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if data, err := os.ReadFile(customGifPath()); err == nil {
			w.Write(data)
			return
		}
		w.Write(defaultGif)
	})

	mux.HandleFunc("/gif/reset", func(w http.ResponseWriter, r *http.Request) {
		os.Remove(customGifPath())
		runtime.EventsEmit(ctx, "gif-changed", "")
	})

	http.ListenAndServe("0.0.0.0:"+clientLocalPort, mux)
}

func findWindowHandle() uintptr {
	classNamePtr, _ := syscall.UTF16PtrFromString("wailsWindow")
	titlePtr, _ := syscall.UTF16PtrFromString("ancol-ani")

	hwnd, _, _ := procFindWindow.Call(
		uintptr(unsafe.Pointer(classNamePtr)),
		uintptr(unsafe.Pointer(titlePtr)),
	)
	return hwnd
}

type rect struct {
	Left, Top, Right, Bottom int32
}

type monitorInfo struct {
	CbSize    uint32
	RcMonitor rect
	RcWork    rect
	DwFlags   uint32
}

const monitorDefaultToPrimary = 1

var (
	procMonitorFromWindow = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfo    = user32.NewProc("GetMonitorInfoW")
)

// Wails' WindowSetPosition adds the window's own monitor work-area origin
// (rcWork.Left/Top) to whatever x/y we pass (see ControlBase.SetPos), so
// x/y here must be computed from that *same* rcWork - fetched via the same
// hwnd and the same Win32 call Wails itself uses - otherwise a DPI/API
// mismatch shifts the result off-screen.
func positionBottomRight(ctx context.Context, hwnd uintptr) {
	if hwnd == 0 {
		return
	}

	work := currentWorkArea(hwnd)
	workWidth := int(work.Right - work.Left)
	workHeight := int(work.Bottom - work.Top)

	x := workWidth - petWidth - margin
	y := workHeight - petHeight - margin
	runtime.WindowSetPosition(ctx, x, y)
}

// Re-checks the monitor's work area every couple of seconds and snaps the
// window back to the bottom-right corner whenever it changed (monitor
// turned off/on, resolution change, taskbar resize, etc). Wails has no
// WM_DISPLAYCHANGE hook exposed, so this polls instead of reacting to the
// event directly.
func watchForMonitorChanges(ctx context.Context, hwnd uintptr) {
	var lastWork rect

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		work := currentWorkArea(hwnd)
		if work != lastWork {
			lastWork = work
			positionBottomRight(ctx, hwnd)
		}
	}
}

func currentWorkArea(hwnd uintptr) rect {
	hMonitor, _, _ := procMonitorFromWindow.Call(hwnd, uintptr(monitorDefaultToPrimary))

	info := monitorInfo{}
	info.CbSize = uint32(unsafe.Sizeof(info))
	procGetMonitorInfo.Call(hMonitor, uintptr(unsafe.Pointer(&info)))

	return info.RcWork
}

// --- Win32 interop: make the window click-through so the animation never
// intercepts mouse input meant for whatever is behind it. ---

const (
	wsExLayered     = 0x80000
	wsExTransparent = 0x20
)

// Typed as a variable (not a const) so the int32(-20) -> uintptr conversion
// below is a runtime sign-extension, not a compile-time overflow check.
var gwlExStyle = int32(-20)

var (
	user32            = syscall.NewLazyDLL("user32.dll")
	procFindWindow    = user32.NewProc("FindWindowW")
	procGetWindowLong = user32.NewProc("GetWindowLongW")
	procSetWindowLong = user32.NewProc("SetWindowLongW")
)

const (
	wsExToolWindow = 0x80
	wsExAppWindow  = 0x40000
)

// Windows enforces a minimum window width (GetSystemMetrics(SM_CXMIN), ~136px)
// for ordinary top-level windows regardless of DisableResize - it does NOT
// enforce the same for height, which is why a requested 70x70 window actually
// comes out ~136x70. Marking it a "tool window" removes that constraint.
//
// Wails also hard-codes WS_EX_APPWINDOW on every window it creates (see
// winc/form.go), which forces a taskbar entry even with WS_EX_TOOLWINDOW
// set. That bit has to be cleared explicitly to actually hide it.
func removeMinimumSizeConstraint(hwnd uintptr) {
	if hwnd == 0 {
		return
	}

	style, _, _ := procGetWindowLong.Call(hwnd, uintptr(gwlExStyle))
	newStyle := (style &^ uintptr(wsExAppWindow)) | uintptr(wsExToolWindow)
	procSetWindowLong.Call(hwnd, uintptr(gwlExStyle), newStyle)
}

// The WS_EX_TOOLWINDOW style (set in removeMinimumSizeConstraint) hides a
// window from the taskbar, but only if applied before Windows has already
// shown it there. Hiding and re-showing forces Windows to recreate the
// taskbar entry using the now-updated style.
func hideFromTaskbar(ctx context.Context, hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	runtime.WindowHide(ctx)
	runtime.WindowShow(ctx)
}

func makeClickThrough(hwnd uintptr) {
	if hwnd == 0 {
		return
	}

	style, _, _ := procGetWindowLong.Call(hwnd, uintptr(gwlExStyle))
	procSetWindowLong.Call(hwnd, uintptr(gwlExStyle), style|uintptr(wsExLayered)|uintptr(wsExTransparent))
}
