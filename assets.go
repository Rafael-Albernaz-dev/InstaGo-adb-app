package instagoassets

import "embed"

// FrontendDist embeds the built Svelte/Vite bundle for Wails 3 AssetOptions.
//
//go:embed all:frontend/dist
var FrontendDist embed.FS
