package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:             "Orbit",
		Width:             1280,
		Height:            820,
		MinWidth:          960,
		MinHeight:         640,
		Frameless:         false,
		BackgroundColour:  &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		WindowStartState:  options.Normal,
		AssetServer:       &assetserver.Options{Assets: assets},
		OnStartup:         app.startup,
		OnShutdown:        app.shutdown,
		Bind:              []interface{}{app},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHiddenInset(),
			Appearance:           mac.NSAppearanceNameDarkAqua,
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "Orbit",
				Message: "Smart Runtime Orchestration\n© Akira Foundation",
			},
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
