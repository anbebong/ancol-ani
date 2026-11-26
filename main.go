package main

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing/fstest"
	"unsafe"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:widget/dist
var assets embed.FS

//go:embed all:settings-main
var settingsWinAssets embed.FS

func main() {
	// OpenSettings (app.go) spawns `<this exe> settings` so the settings
	// window runs as a second, independent window/process/WebView2
	// instance instead of replacing or sharing the widget's own window -
	// wails.Run only ever manages one window per process.
	if len(os.Args) > 1 && os.Args[1] == "settings" {
		runSettingsApp()
		return
	}

	// Only one widget instance should ever run - a second launch (user
	// double-clicking the exe again, a shortcut, etc.) just brings the
	// existing widget's window to the foreground instead of spawning a
	// second always-on-top GIF in the same corner.
	if !acquireSingleInstanceLock() {
		if hwnd := findWindowHandle(); hwnd != 0 {
			procSetForegroundWindow.Call(hwnd)
		}
		return
	}

	// WebView2/Chromium normally freezes rendering (including GIF animation)
	// once a window is detected as occluded by another window on top of it -
	// easy to trigger for a small always-on-top corner widget. Disable that
	// so the GIF keeps animating even when something else briefly covers it.
	os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS",
		"--disable-features=CalculateNativeWinOcclusion "+
			"--disable-backgrounding-occluded-windows "+
			"--disable-renderer-backgrounding "+
			"--disable-background-timer-throttling")

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:         "ancol-ani",
		Width:         petWidth,
		Height:        petHeight,
		DisableResize: true,
		Frameless:     true,
		AlwaysOnTop:   true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              true,
			WindowIsTranslucent:               true,
			DisableFramelessWindowDecorations: true,
			WebviewGpuIsDisabled:              false,
			// Left unset, WebView2 defaults to a folder named after the
			// exe (including ".exe") for its own cache/cookies/local
			// storage - a second, oddly-named sibling next to our own
			// appDataDir() ("ancol-ani"). Put it under that instead.
			WebviewUserDataPath: filepath.Join(appDataDir(), "webview2"),
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

const errorAlreadyExists = syscall.Errno(183)

var (
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutex = kernel32.NewProc("CreateMutexW")
)

// acquireSingleInstanceLock creates (or finds) a named OS mutex unique to
// this app. It returns true for whichever process creates it first; any
// later launch sees it already exists (ERROR_ALREADY_EXISTS) and gets
// false. The mutex handle is intentionally never released/closed - it's
// held for the process's whole lifetime and Windows cleans it up when the
// process exits, which is exactly when a second launch should be allowed
// to become the new primary instance.
func acquireSingleInstanceLock() bool {
	namePtr, _ := syscall.UTF16PtrFromString("ancol-ani-single-instance")
	_, _, errno := procCreateMutex.Call(0, 1, uintptr(unsafe.Pointer(namePtr)))
	return errno != errorAlreadyExists
}

// runSettingsApp is the second window opened by App.OpenSettings (app.go),
// run in a separate process (see main's "settings" branch). It's a plain
// resizable Wails window - not frameless/topmost/click-through like the
// widget - whose only page immediately navigates its real WebView2 to the
// already-running local settings server (startLocalServer in app.go), the
// same page a browser would open. Loading a single inline redirect page
// like this, instead of embedding settings-ui/dist a second time, means the
// settings page's own fetch('/text'), fetch('/position') etc. resolve
// against that server's origin like they already do for any browser tab
// open on it - not against Wails' own internal asset origin.
//
// This window has no Bind, so its page can't call a Go method the way
// widget/index.html calls LocalPort(). It's also a separate OS process
// from the widget, so it can't read activePort (an in-memory var) either
// - the widget process saves the port it actually bound to into
// settings.data's Port field (resolveListener/startup in app.go), and
// loadSettings/getSettings here (same package, same encryption) read it
// back from there.
func runSettingsApp() {
	loadSettings()
	port := getSettings().Port
	if port == "" {
		println("Error: widget not running or has no local server yet")
		return
	}

	raw, err := settingsWinAssets.ReadFile("settings-main/index.html")
	if err != nil {
		println("Error:", err.Error())
		return
	}
	// `ancol-ani.exe settings checkupdate` (launchUpdateScript in app.go,
	// reopening the settings window right after an update) lands the page
	// straight back on the Về Ancol-Ani tab's version section, instead of
	// the normal default (Widget page) a plain `settings` launch gets.
	target := ""
	if len(os.Args) > 2 && os.Args[2] == "checkupdate" {
		target = "#/changelog?section=version"
	}
	html := strings.ReplaceAll(string(raw), "__PORT__", port)
	html = strings.ReplaceAll(html, "__TARGET__", target)
	settingsAssets := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte(html)},
	}

	err = wails.Run(&options.App{
		Title:  settingsWindowTitle,
		Width:  960,
		Height: 700,
		AssetServer: &assetserver.Options{
			Assets: settingsAssets,
		},
		Windows: &windows.Options{
			WebviewUserDataPath: filepath.Join(appDataDir(), "webview2-settings"),
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
