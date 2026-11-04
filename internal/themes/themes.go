// Package themes contains the bundled, offline appearance palettes.
package themes

import "embed"

//go:embed *.css
var Files embed.FS

// Names is the supported configuration vocabulary, in documentation order.
var Names = []string{"default", "github", "github-dark", "base64-dark", "nord"}

func Valid(name string) bool {
	for _, candidate := range Names {
		if name == candidate {
			return true
		}
	}
	return false
}
