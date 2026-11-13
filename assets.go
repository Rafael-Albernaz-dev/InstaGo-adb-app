package instagoassets

import "embed"

// FrontendDist embeds the built Svelte/Vite bundle for Wails 3 AssetOptions.
//
//go:embed all:frontend/dist
var FrontendDist embed.FS

// AppIcon embeds the application PNG icon for desktop/window integration.
//
//go:embed assets/appicon.png
var AppIcon []byte
