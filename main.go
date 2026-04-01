package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	log.SetOutput(os.NewFile(0, os.DevNull))

	db, err := NewDatabase()
	if err != nil {
		log.Fatal("数据库初始化失败:", err)
	}
	defer db.Close()

	search := NewSearchService(db)
	llm := NewLLMClient(db)
	service := NewNoteService(db, search, llm)
	handler := NewAPIHandler(service, search, llm, db)
	app := NewApp(handler, search)

	err = wails.Run(&options.App{
		Title:     "AI笔记助手",
		Width:     1200,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,

		AssetServer: &assetserver.Options{
			Assets: assets,
		},

		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},

		OnStartup: func(ctx context.Context) {
			app.OnStartup(ctx)
		},

		OnBeforeClose: func(ctx context.Context) bool {
			return app.OnBeforeClose(ctx)
		},

		Bind: []interface{}{
			app,
		},

		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
	})

	if err != nil {
		fmt.Println("Error:", err.Error())
	}
}
