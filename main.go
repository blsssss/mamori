package main

import (
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--probe" {
		os.Exit(runProbe(os.Args[2:]))
	}
	app := NewApp()
	err := wails.Run(&options.App{
		Title:            "mamori",
		Width:            1180,
		Height:           800,
		MinWidth:         960,
		MinHeight:        640,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 246, G: 245, B: 241, A: 255},
		OnStartup:        app.startup,
		Bind:             []any{app},
		Windows: &windows.Options{
			Theme: windows.SystemDefault,
		},
	})
	if err != nil {
		println("Error:", err.Error())
		os.Exit(1)
	}
}
