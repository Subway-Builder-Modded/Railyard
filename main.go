package main

import (
	"embed"
	"os"

	"railyard/internal/deeplink"
	"railyard/internal/paths"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	if startupTarget, ok := deeplink.ParseArgs(os.Args[1:]); ok {
		app.HandleDeepLinkTarget(startupTarget)
	}

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "Railyard",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               paths.LockFilePath(),
			OnSecondInstanceLaunch: app.onSecondInstanceLaunch,
		},
		Mac: &mac.Options{
			OnUrlOpen: app.onURLOpen,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
			app.Registry,
			app.Config,
			app.Downloader,
			app.Profiles,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
