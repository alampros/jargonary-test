package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"jargonary/internal/themes"
)

func (c *Config) resolveAppearance(configPath string) error {
	for key, dst := range map[string]*string{"site_name": &c.SiteName, "theme": &c.Theme} {
		name := "JARGONARY_" + strings.ToUpper(key)
		if value, ok := os.LookupEnv(name); ok {
			*dst = value
			c.Origins[key] = "environment: " + name
		}
	}
	for key, dst := range map[string]*[]string{"custom_css": &c.CustomCSS, "custom_js": &c.CustomJS} {
		name := "JARGONARY_" + strings.ToUpper(key)
		if value, ok := os.LookupEnv(name); ok {
			var paths []string
			if err := json.Unmarshal([]byte(value), &paths); err != nil || paths == nil {
				return fmt.Errorf("%s must be a JSON array of file paths", name)
			}
			*dst = paths
			c.Origins[key] = "environment: " + name
		}
	}
	base := "."
	if configPath != "" {
		base = filepath.Dir(configPath)
	}
	for _, paths := range [][]string{c.CustomCSS, c.CustomJS} {
		for i, path := range paths {
			if strings.TrimSpace(path) == "" {
				return fmt.Errorf("custom asset paths must not be empty")
			}
			if !filepath.IsAbs(path) {
				path = filepath.Join(base, path)
			}
			absolute, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("cannot resolve custom asset path")
			}
			paths[i] = absolute
		}
	}
	return nil
}

func (c *Config) validateAppearance() error {
	if c.Theme == "" {
		c.Theme = "default"
	}
	if !themes.Valid(c.Theme) {
		return fmt.Errorf("unknown theme; choose one of: %s", strings.Join(themes.Names, ", "))
	}
	var err error
	c.CustomCSSData, err = readCustomAssets(c.CustomCSS, ".css")
	if err != nil {
		return err
	}
	c.CustomJSData, err = readCustomAssets(c.CustomJS, ".js")
	return err
}

func readCustomAssets(paths []string, extension string) ([][]byte, error) {
	var out [][]byte
	for i, path := range paths {
		if strings.ToLower(filepath.Ext(path)) != extension {
			return nil, fmt.Errorf("custom %s asset %d requires a %s file", extension, i+1, extension)
		}
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("cannot open custom %s asset %d", extension, i+1)
		}
		info, err := f.Stat()
		if err != nil || !info.Mode().IsRegular() {
			f.Close()
			return nil, fmt.Errorf("custom %s asset %d must be a regular file", extension, i+1)
		}
		const maxBytes = 4 << 20
		data, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
		f.Close()
		if err != nil || len(data) > maxBytes {
			return nil, fmt.Errorf("cannot read custom %s asset %d (maximum 4 MiB)", extension, i+1)
		}
		out = append(out, data)
	}
	return out, nil
}
