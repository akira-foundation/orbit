package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func buildMenu(app *App) *menu.Menu {
	m := menu.NewMenu()
	m.Append(menu.AppMenu())
	m.Append(menu.EditMenu())

	system := m.AddSubmenu("System")
	system.AddText("Open Settings…", keys.CmdOrCtrl(","), func(_ *menu.CallbackData) {
		wailsruntime.EventsEmit(app.ctx, "menu:open-settings")
	})
	system.AddSeparator()
	system.AddText("Set up Local Domains", nil, func(_ *menu.CallbackData) {
		wailsruntime.EventsEmit(app.ctx, "menu:system-setup")
	})
	system.AddText("Reset Local Domains", nil, func(_ *menu.CallbackData) {
		wailsruntime.EventsEmit(app.ctx, "menu:system-reset")
	})

	m.Append(menu.WindowMenu())
	return m
}

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:            "Orbit",
		Menu:             buildMenu(app),
		Width:            1280,
		Height:           820,
		MinWidth:         960,
		MinHeight:        640,
		Frameless:        false,
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		WindowStartState: options.Normal,
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind:             []interface{}{app},
		// Closing the window must NOT terminate the runtime daemon. The
		// proxy, runtime manager and *.orbit.test routing keep running in
		// the background; only the UI hides. Dock icon click reopens it.
		// Quit Orbit (Cmd+Q) goes through the App menu and triggers the
		// real shutdown via OnShutdown.
		HideWindowOnClose: true,
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
