package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func cleanAppearanceEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{"JARGONARY_SITE_NAME", "JARGONARY_THEME", "JARGONARY_CUSTOM_CSS", "JARGONARY_CUSTOM_JS"} {
		value, exists := os.LookupEnv(name)
		os.Unsetenv(name)
		t.Cleanup(func() {
			if exists {
				os.Setenv(name, value)
			} else {
				os.Unsetenv(name)
			}
		})
	}
}

func TestAppearanceDefaultsAndEnvironmentPrecedence(t *testing.T) {
	cleanAppearanceEnv(t)
	c, err := Load("")
	if err != nil || c.SiteName != "Jargonary" || c.Theme != "default" {
		t.Fatalf("appearance defaults: %+v %v", c, err)
	}
	path := writeConfig(t, "site_name: File brand\ntheme: github\ncustom_css: [missing.css]\n")
	base := filepath.Dir(path)
	for name, data := range map[string]string{"first.css": ":root { --jg-accent: purple; }", "second.css": "body { margin: 0; }", "hooks.js": "console.log('custom brand');"} {
		if err := os.WriteFile(filepath.Join(base, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("JARGONARY_SITE_NAME", "Environment brand")
	t.Setenv("JARGONARY_THEME", "github-dark")
	t.Setenv("JARGONARY_CUSTOM_CSS", `["first.css", "second.css"]`)
	t.Setenv("JARGONARY_CUSTOM_JS", `["hooks.js"]`)
	c, err = Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.SiteName != "Environment brand" || c.Theme != "github-dark" || c.Origins["theme"] != "environment: JARGONARY_THEME" {
		t.Fatal("environment did not override YAML or record its origin")
	}
	if c.CustomCSS[0] != filepath.Join(base, "first.css") || len(c.CustomCSSData) != 2 || !strings.Contains(string(c.CustomCSSData[1]), "margin") || len(c.CustomJSData) != 1 {
		t.Fatal("custom files were not resolved relative to config and loaded in order")
	}
	if _, leaked := c.Diagnostics()["CustomCSSData"]; leaked {
		t.Fatal("resolved configuration includes file contents")
	}
	t.Setenv("JARGONARY_CUSTOM_CSS", `[]`)
	c, err = Load(path)
	if err != nil || len(c.CustomCSSData) != 0 {
		t.Fatal("empty environment array did not clear configured assets")
	}
}

func TestAppearanceRejectsInvalidThemeAndFiles(t *testing.T) {
	cleanAppearanceEnv(t)
	for _, body := range []string{"theme: unknown\n", "theme: ../github\n", "custom_css: [missing.css]\n", "custom_js: [file.css]\n", "custom_css: ['']\n"} {
		if _, err := Load(writeConfig(t, body)); err == nil {
			t.Fatalf("invalid appearance accepted: %s", body)
		}
	}
	for _, value := range []string{"not-an-array", "null", `[4]`} {
		t.Setenv("JARGONARY_CUSTOM_CSS", value)
		if _, err := Load(""); err == nil {
			t.Fatalf("invalid custom CSS environment array accepted: %s", value)
		}
	}
}
