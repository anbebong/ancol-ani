package main

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"embed"
	"encoding/json"
	"image/gif"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const clientLocalPort = "8788"

//go:embed VERSION
var rawVersion string

var appVersion = strings.TrimSpace(rawVersion)

// Shown on the settings page's "What's new" page (GET /changelog), so the
// changelog a user reads always matches the build they are running.
//
//go:embed CHANGELOG.md
var changelog []byte

// A large GIF (many MB) can freeze the small 70x70 WebView2 panel while it
// decodes/animates, so uploads are capped to keep the widget responsive.
const maxGifBytes = 3 << 20 // 3MB

// Shown on the widget's text bubble until the user sets something else.
const defaultText = "Ancol Hello"

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
	return filepath.Join(appDataDir(), "custom.gif")
}

// settingsPath holds everything except the GIF itself (which stays binary,
// outside JSON) - the widget's text bubble and its screen corner. Used to
// live as two separate small files (custom-text.txt, position.txt); merged
// here so appDataDir() doesn't keep growing one file per setting.
func settingsPath() string {
	return filepath.Join(appDataDir(), "settings.data")
}

const defaultCorner = "br"

var validCorners = map[string]bool{"br": true, "bl": true, "tr": true, "tl": true}

type widgetSettings struct {
	Corner string `json:"corner"`
	Text   string `json:"text"`
}

// currentSettings is read by the window-positioning code (main goroutine +
// watchForMonitorChanges) and the /text, /position HTTP handlers, so it's
// an atomic.Value (storing *widgetSettings, replaced wholesale on save)
// rather than a plain global.
var currentSettings atomic.Value // *widgetSettings

// settingsMu serializes read-modify-write of the settings file so a /text
// save and a /position save can't race and drop one another's change.
var settingsMu sync.Mutex

func getSettings() *widgetSettings {
	if v, ok := currentSettings.Load().(*widgetSettings); ok {
		return v
	}
	return &widgetSettings{Corner: defaultCorner}
}

func getCorner() string {
	if c := getSettings().Corner; validCorners[c] {
		return c
	}
	return defaultCorner
}

func getText() string {
	if t := getSettings().Text; t != "" {
		return t
	}
	return defaultText
}

// settingsKey is a fixed AES-256 key baked into the binary. This is not
// meant to keep a determined attacker out (the key ships in the exe) - it
// just keeps the widget's text/corner from sitting as plain, grep-able
// text on disk.
var settingsKey = []byte("ancol-ani-local-settings-key-321")

// encryptSettings seals data with AES-GCM, prefixing the random nonce.
func encryptSettings(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(settingsKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, data, nil), nil
}

// decryptSettings reverses encryptSettings.
func decryptSettings(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(settingsKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(data) < gcm.NonceSize() {
		return nil, aes.KeySizeError(len(data))
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// loadSettings reads the persisted settings file (if any) into
// currentSettings - called once at startup before the first positionWidget
// call.
func loadSettings() {
	s := &widgetSettings{Corner: defaultCorner}
	if raw, err := os.ReadFile(settingsPath()); err == nil {
		if data, err := decryptSettings(raw); err == nil {
			json.Unmarshal(data, s)
		}
	}
	if !validCorners[s.Corner] {
		s.Corner = defaultCorner
	}
	currentSettings.Store(s)
}

// saveSettings applies mutate to a copy of the current settings, persists
// it (AES-GCM sealed) to disk, and only then makes it the new in-memory
// value.
func saveSettings(mutate func(*widgetSettings)) error {
	settingsMu.Lock()
	defer settingsMu.Unlock()

	next := *getSettings()
	mutate(&next)

	data, err := json.Marshal(&next)
	if err != nil {
		return err
	}
	sealed, err := encryptSettings(data)
	if err != nil {
		return err
	}
	if err := os.WriteFile(settingsPath(), sealed, 0o644); err != nil {
		return err
	}
	currentSettings.Store(&next)
	return nil
}

func appDataDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	dir = filepath.Join(dir, "ancol-ani")
	os.MkdirAll(dir, 0o755)
	return dir
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

// LocalPort is bound to the frontend (main.go's Bind) so frontend/index.html
// can build its fetch URLs from the single source of truth (clientLocalPort)
// instead of hardcoding the port in JS too.
func (a *App) LocalPort() string {
	return clientLocalPort
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	loadSettings()

	hwnd := findWindowHandle()
	removeMinimumSizeConstraint(hwnd)
	hideFromTaskbar(ctx, hwnd)
	runtime.WindowSetSize(ctx, petWidth, petHeight)
	positionWidget(hwnd)
	makeClickThrough(hwnd)

	go watchForMonitorChanges(hwnd)
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

	mux.HandleFunc("/changelog", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Write(changelog)
	})

	mux.HandleFunc("/text", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			text := strings.TrimSpace(r.FormValue("text"))
			if text == "" {
				return
			}
			if err := saveSettings(func(s *widgetSettings) { s.Text = text }); err != nil {
				http.Error(w, "cannot save", http.StatusInternalServerError)
				return
			}
			runtime.EventsEmit(ctx, "server-text", text)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write([]byte(getText()))
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

	mux.HandleFunc("/position", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			corner := strings.TrimSpace(r.FormValue("corner"))
			if !validCorners[corner] {
				http.Error(w, "invalid corner", http.StatusBadRequest)
				return
			}
			if err := saveSettings(func(s *widgetSettings) { s.Corner = corner }); err != nil {
				http.Error(w, "cannot save", http.StatusInternalServerError)
				return
			}
			positionWidget(findWindowHandle())
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(getCorner()))
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
	procGetWindowRect     = user32.NewProc("GetWindowRect")
	procGetDpiForWindow   = user32.NewProc("GetDpiForWindow")
	procSetWindowPos      = user32.NewProc("SetWindowPos")
)

const (
	swpNoSize     = 0x0001
	swpNoMove     = 0x0002
	swpNoActivate = 0x0010
)

// HWND_TOPMOST is (HWND)-1; kept as a variable so the conversion to
// uintptr is a runtime sign-extension (same trick as gwlExStyle).
var hwndTopmost = int32(-1)

// positionWidget snaps the window to whichever corner of its monitor's
// work area (the screen minus the taskbar) is currently selected
// (getCorner - "br"/"bl"/"tr"/"tl", default bottom-right) and keeps it
// topmost.
//
// Everything here is in *physical* pixels: GetMonitorInfo, GetWindowRect
// and SetWindowPos all use them for this (per-monitor DPI aware) process.
// Mixing physical work-area size with petWidth/petHeight in logical
// pixels (e.g. via Wails' WindowSetPosition) breaks at scaling != 100% -
// at 125% that put the widget 11px past the right edge and ~20px under
// the taskbar, where the (also topmost) taskbar hid it.
func positionWidget(hwnd uintptr) {
	if hwnd == 0 {
		return
	}

	work := currentWorkArea(hwnd)

	var win rect
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&win)))
	w := win.Right - win.Left
	h := win.Bottom - win.Top

	m := int32(scaleForDPI(hwnd, margin))

	var x, y int32
	switch getCorner() {
	case "bl":
		x, y = work.Left+m, work.Bottom-h-m
	case "tr":
		x, y = work.Right-w-m, work.Top+m
	case "tl":
		x, y = work.Left+m, work.Top+m
	default: // "br"
		x, y = work.Right-w-m, work.Bottom-h-m
	}

	procSetWindowPos.Call(hwnd, uintptr(hwndTopmost),
		uintptr(x), uintptr(y), 0, 0,
		swpNoSize|swpNoActivate)
}

// keepOnTop re-asserts HWND_TOPMOST without moving/resizing/activating.
// Windows can push a topmost window down the z-order (e.g. when another
// topmost window - the taskbar, a fullscreen app - is activated), and
// WS_EX_TOPMOST set once at creation never re-applies itself.
func keepOnTop(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	procSetWindowPos.Call(hwnd, uintptr(hwndTopmost), 0, 0, 0, 0,
		swpNoMove|swpNoSize|swpNoActivate)
}

// scaleForDPI converts a length in logical (96-DPI) pixels to physical
// pixels for the monitor the window is on.
func scaleForDPI(hwnd uintptr, logical int) int {
	dpi, _, _ := procGetDpiForWindow.Call(hwnd)
	if dpi == 0 {
		dpi = 96
	}
	return logical * int(dpi) / 96
}

// Every couple of seconds: snap the window back to the bottom-right
// corner if the monitor's work area or the window's own size changed
// (monitor turned off/on, resolution or scaling change, taskbar resize,
// etc), and re-assert topmost. Wails has no WM_DISPLAYCHANGE hook
// exposed, so this polls instead of reacting to the event directly.
func watchForMonitorChanges(hwnd uintptr) {
	var lastWork, lastWin rect

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		work := currentWorkArea(hwnd)
		var win rect
		procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&win)))
		size := rect{Right: win.Right - win.Left, Bottom: win.Bottom - win.Top}

		if work != lastWork || size != lastWin {
			lastWork, lastWin = work, size
			positionWidget(hwnd)
		} else {
			keepOnTop(hwnd)
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
