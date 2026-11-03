// Package webui embeds Vite's production build. Run `make ui` before building Go.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var assets embed.FS

func Assets() fs.FS {
	result, err := fs.Sub(assets, "dist")
	if err != nil {
		panic(err)
	}
	return result
}
