// Package webui embeds Vite's production build. Run `go tool task ui` before building Go.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var assets embed.FS

// Favicon is the outlined wordmark initial, available without a frontend build.
//
//go:embed favicon.svg
var Favicon []byte

func Assets() fs.FS {
	result, err := fs.Sub(assets, "dist")
	if err != nil {
		panic(err)
	}
	return result
}
