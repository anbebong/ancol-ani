package main

import (
	"embed"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
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
