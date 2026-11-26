package main

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"embed"
	"encoding/json"
	"fmt"
	"image/gif"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows/registry"
)

// preferredPort is where the local server tries to listen first. Picked
// away from the common 8788-ish / 8080-ish ranges other local dev tools
// often grab, to make a collision less likely in the first place -
// resolveListener below still scans forward and falls back to any free
// port if even this range is all taken.
const preferredPort = 40404
const portScanRange = 50

// activePort is set once, synchronously, before startLocalServer's
// goroutine starts - every reader (LocalPort, the /position etc. HTTP
// handlers don't need it, OpenSettings's redirect page) only runs after
// that point, so a plain string is fine without an atomic.Value.
var activePort string

// resolveListener scans forward from preferredPort for a free one, and
// as a last resort asks the OS for any free port (net.Listen with ":0").
// Bound to 127.0.0.1 only - earlier versions used 0.0.0.0 so a phone on
// the same LAN could reach this page too, but that also meant anyone
// else on that LAN/Wi-Fi could reach it; narrowed to localhost-only.
func resolveListener() (net.Listener, string) {
	for i := 0; i < portScanRange; i++ {
		port := strconv.Itoa(preferredPort + i)
		if ln, err := net.Listen("tcp", "127.0.0.1:"+port); err == nil {
			return ln, port
		}
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		// Every port in range AND the OS-assigned fallback failed -
		// nothing left to try; the app continues without a local server
		// (widget still shows, just without GIF/text/settings working).
		return nil, ""
	}
	return ln, strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
}

//go:embed VERSION
var rawVersion string

var appVersion = strings.TrimSpace(rawVersion)

// githubRepo is checked via GitHub's own public Releases API instead of
// a self-hosted endpoint - the repo is already public there, so this
// needs no server of ours at all. Publishing a release is the normal
// "git tag + gh release create / GitHub's UI", attaching a build named
// exactly githubReleaseAsset - nothing in this app's own infra to keep
// in sync.
const githubRepo = "anbebong/ancol-ani"
const githubReleaseAsset = "ancol-ani.exe"

// githubRelease mirrors the fields this app needs from GitHub's release
// JSON (GET https://api.github.com/repos/{repo}/releases/latest).
type githubRelease struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// version strips a possible leading "v" from a git tag ("v1.1.5" or
// "1.1.5" both compare fine against VERSION's bare "1.1.5").
func (r *githubRelease) version() string {
	return strings.TrimPrefix(r.TagName, "v")
}

// downloadURL finds the release asset literally named githubReleaseAsset
// among whatever files were attached to the release.
func (r *githubRelease) downloadURL() (string, error) {
	for _, a := range r.Assets {
		if a.Name == githubReleaseAsset {
			return a.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("release không có file %s đính kèm", githubReleaseAsset)
}

// fetchLatestRelease calls GitHub's public Releases API. Network/parse
// errors are returned as-is so the HTTP handler can surface something
// useful ("no internet" vs "no release published yet" etc) rather than a
// generic failure. GitHub's API requires a User-Agent header on every
// request or it responds 403, hence the manual http.Request instead of
// a plain http.Get.
func fetchLatestRelease() (*githubRelease, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/"+githubRepo+"/releases/latest", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ancol-ani-updater")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server trả về %d", resp.StatusCode)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// isNewerVersion compares two "x.y.z" strings component by component
// (numeric, not lexicographic - "1.1.10" counts as newer than "1.1.9").
// Any component that fails to parse as a number is treated as 0, so a
// malformed string just loses the comparison instead of panicking.
func isNewerVersion(candidate, current string) bool {
	c := strings.Split(candidate, ".")
	k := strings.Split(current, ".")
	for i := 0; i < len(c) || i < len(k); i++ {
		var cv, kv int
		if i < len(c) {
			cv, _ = strconv.Atoi(c[i])
		}
		if i < len(k) {
			kv, _ = strconv.Atoi(k[i])
		}
		if cv != kv {
			return cv > kv
		}
	}
	return false
}

// applyUpdate downloads the new exe straight from GitHub's release asset
// URL (HTTPS to github.com/objects.githubusercontent.com is the only
// integrity check here - GitHub's API gives no checksum to verify
// against) and arranges for it to replace this exe and relaunch. It
// never touches the real exe path directly: Windows won't let another
// process overwrite a running exe's bytes in place, but it does allow
// renaming the running exe out of the way, which is what the generated
// batch script does after this process has fully exited (so the file is
// no longer open for execution).
func applyUpdate(downloadURL string) error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	resp, err := http.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tải bản mới lỗi, server trả về %d", resp.StatusCode)
	}

	newExePath := exePath + ".new"
	out, err := os.Create(newExePath)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, resp.Body)
	out.Close()
	if copyErr != nil {
		os.Remove(newExePath)
		return copyErr
	}

	return launchUpdateScript(exePath, newExePath)
}

// launchUpdateScript writes a small batch file that waits for this
// process (by PID) to exit, swaps the verified .new file into place, and
// relaunches it - then starts that script detached (via "cmd /c start",
// which spawns an independent process rather than a child of this one,
// so it survives this process quitting right after).
func launchUpdateScript(exePath, newExePath string) error {
	exeName := filepath.Base(exePath)
	logPath := filepath.Join(os.TempDir(), "ancol-ani-update.log")

	// Wait by IMAGENAME, not a single captured PID: the widget and the
	// separate settings window (main.go's "settings" subcommand, its own
	// OS process) are both ancol-ani.exe and both hold the file open for
	// as long as they're running, so waiting on only one of them left
	// "move" hitting "Access is denied" whenever the other was still up
	// (the /apply-update handler now also asks the settings window to
	// close, but this loop is the real guarantee, not that courtesy).
	//
	// "move" can also still fail right after every instance is gone -
	// likely Windows Defender/AV briefly scanning the freshly-downloaded
	// .new file, or the OS not having fully released a handle the
	// instant a process exits - so it's retried too, instead of trusting
	// it to succeed on the first try. Only "start" the result once
	// ".new" is actually gone (meaning move succeeded); if every retry
	// fails, fall back to relaunching the untouched old exe rather than
	// leaving the user with nothing running, and leave both the log and
	// the still-present ".new" file for a human to look at.
	script := fmt.Sprintf(`@echo off
set LOG="%s"
echo [update] started, waiting for all %s instances to exit > %%LOG%%
:wait
tasklist /fi "IMAGENAME eq %s" 2>NUL | find /I "%s" >NUL
if not errorlevel 1 (
    timeout /t 1 /nobreak >NUL
    goto wait
)
echo [update] no instances left, moving new exe into place >> %%LOG%%
set TRIES=0
:moveretry
move /y "%s" "%s" >> %%LOG%% 2>&1
if exist "%s" (
    set /a TRIES+=1
    if %%TRIES%% lss 10 (
        timeout /t 1 /nobreak >NUL
        goto moveretry
    )
    echo [update] FAILED after 10 tries, leaving .new and starting old exe as-is >> %%LOG%%
) else (
    echo [update] move succeeded after %%TRIES%% retries >> %%LOG%%
)
start "" "%s"
timeout /t 2 /nobreak >NUL
start "" "%s" settings checkupdate
del "%%~f0"
`, logPath, exeName, exeName, exeName, newExePath, exePath, newExePath, exePath, exePath)

	scriptPath := filepath.Join(os.TempDir(), "ancol-ani-update.bat")
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		return err
	}

	cmd := exec.Command("cmd", "/c", "start", "/min", "", scriptPath)
	return cmd.Start()
}

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

//go:embed widget/src/mascot.gif
var defaultGif []byte

// The local settings page (http://localhost:<port>, port chosen by
// resolveListener) is its own small Vite
// project under settings-ui/, built separately (cd settings-ui && npm run
// build) and embedded here as static files, instead of living as an inline
// HTML string in Go.
//
//go:embed settings-ui/dist
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

var validCorners = map[string]bool{"br": true, "bl": true, "tr": true, "tl": true, "tc": true, "bc": true}

const defaultTextColor = "#ffffff"
const defaultTextEffect = "none"

var validTextEffects = map[string]bool{"none": true, "fade": true, "bounce": true, "shake": true, "typewriter": true}

var hexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

type widgetSettings struct {
	Corner string `json:"corner"`
	Text   string `json:"text"`
	// Display style for Text: a hex color and one named entrance effect,
	// applied by the widget (widget/index.html) each time the text
	// changes. Kept separate from Text itself so re-saving just the text
	// (the common case) doesn't need to know the current style.
	TextColor  string `json:"textColor"`
	TextEffect string `json:"textEffect"`
	// Text may hold several lines (one message per line); the widget then
	// shows them one after another, each for TextInterval seconds.
	TextInterval int `json:"textInterval"`
	// Port is only here so the separate "settings" subprocess (main.go's
	// runSettingsApp, spawned by OpenSettings) can learn which port THIS
	// process actually bound to - it has no other channel back to this
	// process, since activePort below only lives in that process' memory.
	Port string `json:"port"`
	// AdapterEnabled is an explicit on/off switch, separate from whether a
	// .wasm adapter is imported at all (see wasmsource.go): importing one
	// only makes it AVAILABLE, it never silently starts overriding
	// manually typed Text - the user has to deliberately switch to it, and
	// can switch back to manual without losing the imported adapter/config.
	AdapterEnabled bool `json:"adapterEnabled"`
	// ActiveAdapterID: several adapters can be imported at once (see
	// wasmsource.go's adaptersDir), but only this one actually runs.
	// Switching it off (AdapterEnabled=false) or an adapter error never
	// clears this - re-enabling resumes the same one, not nothing.
	ActiveAdapterID string `json:"activeAdapterId"`
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

func getTextColor() string {
	if c := getSettings().TextColor; hexColorRe.MatchString(c) {
		return c
	}
	return defaultTextColor
}

func getTextEffect() string {
	if e := getSettings().TextEffect; validTextEffects[e] {
		return e
	}
	return defaultTextEffect
}

// Seconds each line stays up when Text has several lines.
const (
	defaultTextInterval = 4
	minTextInterval     = 2
	maxTextInterval     = 30
	maxTextLines        = 20
	maxTextLineRunes    = 200
)

func getTextInterval() int {
	if n := getSettings().TextInterval; n >= minTextInterval && n <= maxTextInterval {
		return n
	}
	return defaultTextInterval
}

// normalizeText turns what the settings page sent into the stored form:
// one message per line, each trimmed, blank lines dropped, at most
// maxTextLines lines of maxTextLineRunes characters.
func normalizeText(raw string) string {
	var lines []string
	for _, line := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if r := []rune(line); len(r) > maxTextLineRunes {
			line = string(r[:maxTextLineRunes])
		}
		lines = append(lines, line)
		if len(lines) == maxTextLines {
			break
		}
	}
	return strings.Join(lines, "\n")
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

// LocalPort is bound to the frontend (main.go's Bind) so widget/index.html
// can build its fetch URLs from the single source of truth (activePort)
// instead of hardcoding the port in JS too.
func (a *App) LocalPort() string {
	return activePort
}

// OpenSettings is bound to the widget's frontend (clicked on the GIF/text
// bubble) and spawns a second, independent window for the settings page -
// a fresh run of this same exe with the "settings" argument (see main.go's
// runSettingsApp), so the widget keeps animating in its own window/process
// untouched while the settings window runs its own real WebView2 instance.
func (a *App) OpenSettings() {
	if hwnd := findSettingsWindowHandle(); hwnd != 0 {
		procSetForegroundWindow.Call(hwnd)
		return
	}

	exe, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(exe, "settings")
	cmd.Start()
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

	// Resolved synchronously (not inside startLocalServer's goroutine) so
	// activePort and settings.data's Port are both set before anything -
	// the settings window, a fast double-click on the widget - could
	// possibly read them.
	ln, port := resolveListener()
	activePort = port
	if port != "" {
		saveSettings(func(s *widgetSettings) { s.Port = port })
	}
	go startLocalServer(ctx, ln)

	// Resumes whatever text adapter (see wasmsource.go) was imported in a
	// previous run, if any - a no-op if none was.
	go func() {
		if err := startWasmAdapter(ctx); err != nil {
			runtime.LogError(ctx, "startWasmAdapter: "+err.Error())
			logError("startWasmAdapter: %s", err.Error())
		}
	}()
}

// Serves the local settings page: type text in, submit, and it shows up
// directly on the widget's text panel. No server/socket involved - this
// page is the only source of text. ln is nil if resolveListener couldn't
// bind anything, in which case the widget still runs, just without
// GIF/text/settings.
func startLocalServer(ctx context.Context, ln net.Listener) {
	if ln == nil {
		return
	}
	mux := http.NewServeMux()

	webFS, err := fs.Sub(webDist, "settings-ui/dist")
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
			text := normalizeText(r.FormValue("text"))
			if text == "" {
				return
			}
			if err := saveSettings(func(s *widgetSettings) { s.Text = text }); err != nil {
				http.Error(w, "cannot save", http.StatusInternalServerError)
				return
			}
			// While the adapter is switched on, the widget should only
			// ever show ITS output, not whatever manual text happens to
			// be saved underneath - saving still happens above so it's
			// there waiting once the user switches back to "Thủ công".
			if !getSettings().AdapterEnabled {
				emitServerText(ctx, text)
			}
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write([]byte(currentText()))
	})

	mux.HandleFunc("/text/style", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			color := strings.TrimSpace(r.FormValue("color"))
			effect := strings.TrimSpace(r.FormValue("effect"))
			if !hexColorRe.MatchString(color) {
				http.Error(w, "invalid color", http.StatusBadRequest)
				return
			}
			if !validTextEffects[effect] {
				http.Error(w, "invalid effect", http.StatusBadRequest)
				return
			}
			// interval is optional (older settings pages don't send it)
			interval := getTextInterval()
			if raw := strings.TrimSpace(r.FormValue("interval")); raw != "" {
				n, err := strconv.Atoi(raw)
				if err != nil || n < minTextInterval || n > maxTextInterval {
					http.Error(w, "invalid interval", http.StatusBadRequest)
					return
				}
				interval = n
			}
			if err := saveSettings(func(s *widgetSettings) {
				s.TextColor = color
				s.TextEffect = effect
				s.TextInterval = interval
			}); err != nil {
				http.Error(w, "cannot save", http.StatusInternalServerError)
				return
			}
			runtime.EventsEmit(ctx, "server-text-style", map[string]any{"color": color, "effect": effect, "interval": interval})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(map[string]any{"color": getTextColor(), "effect": getTextEffect(), "interval": getTextInterval()})
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

	// --- text adapter (see wasmsource.go): an optional .wasm file that
	// supplies the widget's text itself (API polling, whatever), imported
	// here instead of needing a rebuilt client per customer/source. ---

	mux.HandleFunc("/text-adapter", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			r.Body = http.MaxBytesReader(w, r.Body, maxWasmAdapterBytes)
			if err := r.ParseMultipartForm(maxWasmAdapterBytes); err != nil {
				http.Error(w, "File qua nang (toi da 10MB)", http.StatusRequestEntityTooLarge)
				return
			}
			file, header, err := r.FormFile("wasm")
			if err != nil {
				http.Error(w, "missing file", http.StatusBadRequest)
				return
			}
			defer file.Close()
			raw, err := io.ReadAll(file)
			if err != nil {
				http.Error(w, "File qua nang (toi da 10MB)", http.StatusRequestEntityTooLarge)
				return
			}
			id, schema, err := importWasmAdapter(ctx, header.Filename, raw)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			json.NewEncoder(w).Encode(map[string]any{"id": id, "schema": json.RawMessage(schema)})
			return

		case http.MethodDelete:
			id := strings.TrimSpace(r.URL.Query().Get("id"))
			if id == "" {
				http.Error(w, "missing id", http.StatusBadRequest)
				return
			}
			if err := removeAdapterByID(id); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}

		list, _ := listAdapters()
		settings := getSettings()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]any{
			"enabled":  settings.AdapterEnabled,
			"activeId": settings.ActiveAdapterID,
			"error":    lastAdapterErrorString(),
			"status":   currentAdapterStatus(),
			"log":      currentAdapterLog(),
			"list":     list,
		})
	})

	// Switches which imported adapter is actually running, and turns
	// adapter mode on for it (see selectAdapter in wasmsource.go).
	mux.HandleFunc("/text-adapter/select", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.TrimSpace(r.FormValue("id"))
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		if err := selectAdapter(ctx, id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	})

	mux.HandleFunc("/text-adapter/enabled", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		enabled := strings.TrimSpace(r.FormValue("enabled")) == "true"
		if enabled && getSettings().ActiveAdapterID == "" {
			http.Error(w, "chưa chọn module nào", http.StatusBadRequest)
			return
		}
		if err := saveSettings(func(s *widgetSettings) { s.AdapterEnabled = enabled }); err != nil {
			http.Error(w, "cannot save", http.StatusInternalServerError)
			return
		}
		if err := startWasmAdapter(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !enabled {
			// Switching back to "Thủ công": startWasmAdapter above already
			// stopped the poll loop and cleared the adapter's last output,
			// but nothing pushes the now-current (manual) text to the
			// widget on its own until the next edit - do that now instead
			// of leaving the adapter's last line on screen.
			emitServerText(ctx, getText())
		}
	})

	mux.HandleFunc("/text-adapter/schema", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		schema, err := currentWasmAdapterSchema(ctx, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write([]byte(schema))
	})

	mux.HandleFunc("/text-adapter/config", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}

		if r.Method == http.MethodPost {
			body, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
			if err != nil {
				http.Error(w, "cannot read body", http.StatusBadRequest)
				return
			}
			var probe map[string]any
			if json.Unmarshal(body, &probe) != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}
			if err := saveWasmAdapterConfig(ctx, id, string(body)); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if raw, err := readAdapterJSON(adapterConfigPath(id)); err == nil && raw != nil {
			w.Write(raw)
			return
		}
		w.Write([]byte("{}"))
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

	mux.HandleFunc("/autostart", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			enabled := strings.TrimSpace(r.FormValue("enabled")) == "true"
			if err := setAutoStart(enabled); err != nil {
				http.Error(w, "cannot save", http.StatusInternalServerError)
				return
			}
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if isAutoStartEnabled() {
			w.Write([]byte("true"))
		} else {
			w.Write([]byte("false"))
		}
	})

	mux.HandleFunc("/check-update", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		rel, err := fetchLatestRelease()
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"hasUpdate": isNewerVersion(rel.version(), appVersion),
			"version":   rel.version(),
			"notes":     rel.Body,
		})
	})

	mux.HandleFunc("/apply-update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		rel, err := fetchLatestRelease()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		url, err := rel.downloadURL()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := applyUpdate(url); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write([]byte("ok"))
		go func() {
			time.Sleep(300 * time.Millisecond)
			// This request itself came from the settings window (a
			// separate process/exe invocation) - that process is still
			// holding its own open handle on this exe's file, same as
			// the widget. Ask it to close too, so launchUpdateScript's
			// wait-for-no-running-instances doesn't just sit there
			// retrying "Access is denied" until it gives up.
			if hwnd := findSettingsWindowHandle(); hwnd != 0 {
				procPostMessage.Call(hwnd, wmClose, 0, 0)
			}
			runtime.Quit(ctx)
		}()
	})

	http.Serve(ln, mux)
}

// autoStartRegPath/Name follow Windows' standard per-user autostart
// mechanism (HKCU, not HKLM, so no admin rights needed): any value under
// this key whose data is a path gets launched once at login.
const (
	autoStartRegPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	autoStartName    = "ancol-ani"
)

// isAutoStartEnabled reports whether the Run key currently has our value
// set, regardless of what path it points to (it's not re-validated
// against the exe's current location - see setAutoStart's comment).
func isAutoStartEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, autoStartRegPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()

	_, _, err = k.GetStringValue(autoStartName)
	return err == nil
}

// setAutoStart writes (or removes) the Run value, pointing at wherever
// this exe is currently running from (os.Executable()) - there's no fixed
// install location, so the path is only known at the moment the user
// flips the toggle. If the exe is later moved, autostart silently stops
// working until the user toggles it off and back on from the new
// location; this is intentionally not auto-repaired on startup.
func setAutoStart(enabled bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, autoStartRegPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	if !enabled {
		err := k.DeleteValue(autoStartName)
		if err != nil && err != registry.ErrNotExist {
			return err
		}
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return k.SetStringValue(autoStartName, exe)
}

func findWindowHandle() uintptr {
	return findWindowByTitle("ancol-ani")
}

// settingsWindowTitle is the title of the second window opened by
// OpenSettings (runSettingsApp in main.go) - distinct from "ancol-ani" so
// findWindowHandle (used for the click-through widget) never matches it.
const settingsWindowTitle = "Ancol-Ani - Cài đặt"

func findSettingsWindowHandle() uintptr {
	return findWindowByTitle(settingsWindowTitle)
}

func findWindowByTitle(title string) uintptr {
	classNamePtr, _ := syscall.UTF16PtrFromString("wailsWindow")
	titlePtr, _ := syscall.UTF16PtrFromString(title)

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
	case "tc":
		x, y = work.Left+(work.Right-work.Left-w)/2, work.Top+m
	case "bc":
		x, y = work.Left+(work.Right-work.Left-w)/2, work.Bottom-h-m
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
	user32                  = syscall.NewLazyDLL("user32.dll")
	procFindWindow          = user32.NewProc("FindWindowW")
	procGetWindowLong       = user32.NewProc("GetWindowLongW")
	procSetWindowLong       = user32.NewProc("SetWindowLongW")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procPostMessage         = user32.NewProc("PostMessageW")
)

const wmClose = 0x0010

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

// makeClickThrough used to also set WS_EX_TRANSPARENT so every click
// passed through to whatever sat behind the widget. That's gone now that
// clicking the widget opens the settings window (OpenSettings) - only
// WS_EX_LAYERED stays, needed for the GIF's alpha transparency.
func makeClickThrough(hwnd uintptr) {
	if hwnd == 0 {
		return
	}

	style, _, _ := procGetWindowLong.Call(hwnd, uintptr(gwlExStyle))
	procSetWindowLong.Call(hwnd, uintptr(gwlExStyle), style|uintptr(wsExLayered))
}
