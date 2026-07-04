package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"quantflow/internal/ws"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create the MarketWSService first. Its Hub field is set during
	// App.ServiceStartup, before the HTTP server starts.
	wsSvc := &ws.MarketWSService{}

	app := application.New(application.Options{
		Name:        "quantflow",
		Description: "QuantFlow Terminal — 双模式量化金融终端",
		Services: []application.Service{
			application.NewService(&App{wsSvc: wsSvc}),
			application.NewServiceWithOptions(wsSvc, application.ServiceOptions{
				Route: "/ws/market",
			}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "QuantFlow Terminal",
		Width:            1400,
		Height:           900,
		MinWidth:         900,
		MinHeight:        600,
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	err := app.Run()
	if err != nil {
		log.Fatal(err.Error())
	}
}
