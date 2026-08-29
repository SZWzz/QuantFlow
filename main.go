package main

import (
	"embed"
	"log/slog"
	"os"
	"quantflow/internal/crash"
	"quantflow/internal/ws"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// run() carries the real logic so deferred cleanup (panic capture) runs
	// before the process-level os.Exit — os.Exit skips defers.
	os.Exit(run())
}

func run() int {
	defer crash.CapturePanic()

	// Initialize the crash report store and register signal handlers before
	// any other initialization, so panics and fatal signals are always captured.
	store := crash.NewStore(crash.CrashDir())
	crash.StartCrashHandler(store)

	// Create the MarketWSService first. Its Hub field is set during
	// App.ServiceStartup, before the HTTP server starts.
	wsSvc := &ws.MarketWSService{}

	appInstance := &App{wsSvc: wsSvc, crashStore: store}
	app := application.New(application.Options{
		Name:        "quantflow",
		Description: "QuantFlow Terminal — 双模式量化金融终端",
		Services: []application.Service{
			application.NewService(appInstance),
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
	appInstance.wailsApp = app

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
		slog.Error("application run failed", "error", err)
		return 1
	}
	return 0
}
