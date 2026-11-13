package main

import (
	"context"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	instagoassets "github.com/Rafael-Albernaz-dev/InstaGo-adb-app"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/assets"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/sqlite"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/adapters/wailsapp"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/usecase/editor"
)

const defaultCreativeBase = "/home/toru/Documents/Instagram Rural Conecta"

func main() {
	ctx := context.Background()

	dataRoot := resolveDataRoot()
	dbPath := filepath.Join(dataRoot, "instago.db")
	exportsDir := filepath.Join(dataRoot, "exports")

	store, err := sqlite.Open(ctx, dbPath)
	if err != nil {
		log.Fatalf("instago: failed to open sqlite: %v", err)
	}
	defer func() { _ = store.Close() }()

	creativeBase := strings.TrimSpace(os.Getenv("INSTAGO_CREATIVE_BASE"))
	if creativeBase == "" {
		creativeBase = defaultCreativeBase
	}
	assetResolver := assets.NewResolver(creativeBase)

	appVersion := "0.0.0-dev"
	if rawVer, err := os.ReadFile("VERSION"); err == nil {
		if trimmed := strings.TrimSpace(string(rawVer)); trimmed != "" {
			appVersion = trimmed
		}
	}

	svc := editor.NewService(store, creativeBase, assetResolver.BaseExists, exportsDir, appVersion)
	bridge := wailsapp.NewEditorBridge(svc)

	distFS, err := fs.Sub(instagoassets.FrontendDist, "frontend/dist")
	if err != nil {
		log.Fatalf("instago: failed to open embedded frontend/dist: %v", err)
	}

	app := application.New(application.Options{
		Name:        "InstaGo",
		Description: "Desktop marketing automation and creative studio",
		Services: []application.Service{
			application.NewService(bridge),
		},
		Assets: application.AssetOptions{
			Handler:    application.AssetFileServerFS(distFS),
			Middleware: assetResolver.Middleware,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "main",
		Title:            "InstaGo — Editor Nativo (R0)",
		Width:            1360,
		Height:           880,
		MinWidth:         960,
		MinHeight:        680,
		BackgroundColour: application.NewRGB(247, 248, 242),
		URL:              "/",
	})

	if err := app.Run(); err != nil {
		log.Fatalf("instago: application error: %v", err)
	}
}

func resolveDataRoot() string {
	if custom := strings.TrimSpace(os.Getenv("INSTAGO_DATA_DIR")); custom != "" {
		return custom
	}
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, "instago")
	}
	return filepath.Join(".", ".local")
}
