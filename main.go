package main

import (
	"context"
	"embed"
	"goFarmacia/backend"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	db := backend.GetDbInstance()
	app := NewApp(db)
	gmailSvc := backend.NewGmailService(db)
	bancolombiasSvc := backend.NewBancolombiaService(db)

	err := wails.Run(&options.App{
		Title:            "goFarmacia",
		WindowStartState: options.Maximised,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnStartup: func(ctx context.Context) {
			app.startup(ctx)
			gmailSvc.Startup(ctx)
			bancolombiasSvc.Startup(ctx)
		},
		OnShutdown: func(ctx context.Context) {
			bancolombiasSvc.Shutdown()
			app.shutdown(ctx)
		},
		Bind: []any{
			app,
			db,
			gmailSvc,
			bancolombiasSvc,
		},
		Windows: &windows.Options{
			Theme: windows.SystemDefault,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
