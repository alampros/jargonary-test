package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(p, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDefaultsAndSecretRedaction(t *testing.T) {
	c, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if c.Admin.Enabled || c.Server.Listen != "0.0.0.0:8080" || c.Admin.Listen != "127.0.0.1:9090" || c.MCP.Enabled || c.MCP.Listen != "127.0.0.1:9091" {
		t.Fatal("unsafe listener defaults")
	}
	secret := strings.Repeat("secret", 8)
	t.Setenv("EXPORT_SECRET", secret)
	p := writeConfig(t, `admin:
  enabled: true
sources:
  - id: payments
    url: https://example.test/terms?credential=hidden-query
    auth:
      shared_secret:
        env: EXPORT_SECRET
exports:
  - id: public
    name: Public
    include:
      sources: [payments]
      sense_tags:
        any: [public]
    auth:
      shared_secret:
        env: EXPORT_SECRET
`)
	c, err = Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.Sources[0].Token != secret || c.Exports[0].Token != secret {
		t.Fatal("secret was not resolved")
	}
	b, err := json.Marshal(c.Diagnostics())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), secret) || strings.Contains(string(b), "hidden-query") {
		t.Fatal("diagnostics leaked credentials")
	}
	if c.Origins["admin.enabled"] != "configuration file" {
		t.Fatal("missing setting origin")
	}
}

func TestRejectUnsafeConfiguration(t *testing.T) {
	for _, body := range []string{
		"admni:\n  enabled: true\n",
		"admin:\n  enabled: true\n  listen: '0.0.0.0:8080'\n",
		"mcp:\n  enabled: true\n  listen: '0.0.0.0:8080'\n",
		"mcp:\n  enabled: true\n  listen: '127.0.0.1:9090'\n",
		"sources:\n  - id: project\n    file: terms.json\n",
		"exports:\n  - id: public\n    name: Public\n    include: all\n",
		"exports:\n  - id: public\n    name: Public\n    include:\n      typo: true\n    auth:\n      public: true\n",
		"exports:\n  - id: public\n    name: Public\n    include:\n      sense_tags:\n        any: [public]\n      sources: [missing]\n    auth:\n      public: true\n",
	} {
		if _, err := Load(writeConfig(t, body)); err == nil {
			t.Errorf("accepted invalid configuration: %s", body)
		}
	}
}

func TestRejectProjectExportSelector(t *testing.T) {
	for _, hidden := range []string{"false", "true"} {
		body := "hide_jargonary_glossary: " + hidden + "\nsources:\n  - id: company\n    file: company.json\nexports:\n  - id: public\n    name: Public\n    include:\n      sense_tags:\n        any: [public]\n      sources: [project]\n    auth:\n      public: true\n"
		if _, err := Load(writeConfig(t, body)); err == nil || !strings.Contains(err.Error(), "Jargonary project terms cannot be included in exports") {
			t.Fatalf("project export selector should be rejected: %v", err)
		}
	}
}

func TestProjectGlossaryInclusion(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want bool
	}{
		{"default with company", "sources:\n  - id: company\n    file: company.json\n", true},
		{"hidden with company", "hide_jargonary_glossary: true\nsources:\n  - id: company\n    file: company.json\n", false},
		{"empty default", "sources: []\n", true},
		{"empty fallback", "hide_jargonary_glossary: true\nsources: []\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := Load(writeConfig(t, tc.body))
			if err != nil {
				t.Fatal(err)
			}
			if c.IncludeProjectGlossary() != tc.want {
				t.Fatalf("project inclusion = %v, want %v", c.IncludeProjectGlossary(), tc.want)
			}
			if c.HideJargonaryGlossary && (c.Diagnostics()["hide_jargonary_glossary"] != true || c.Origins["hide_jargonary_glossary"] != "configuration file") {
				t.Fatal("hide setting missing from resolved configuration or origins")
			}
		})
	}
	path := writeConfig(t, "hide_jargonary_glossary: true\nterms_dir: terms\n")
	if err := os.Mkdir(filepath.Join(filepath.Dir(path), "terms"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "terms/company.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil || c.IncludeProjectGlossary() {
		t.Fatalf("discovered company source should suppress fallback: %+v %v", c, err)
	}
	if _, err := Load(writeConfig(t, "hide_jargonary_glossary: true\nsources:\n  - id: company\n    file: company.json\nexports:\n  - id: public\n    name: Public\n    include:\n      all: true\n      sources: [project]\n    auth:\n      public: true\n")); err == nil {
		t.Fatal("export accepted a disabled project source")
	}
}

func TestExplicitEmptySourcesDisablesDiscovery(t *testing.T) {
	dir := t.TempDir()
	terms := filepath.Join(dir, "terms")
	if err := os.Mkdir(terms, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(terms, "company.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(p, []byte("terms_dir: terms\nsources: []\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Sources) != 0 {
		t.Fatal("empty sources unexpectedly rediscovered files")
	}
	if err := os.WriteFile(p, []byte("terms_dir: terms\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c, err = Load(p)
	if err != nil || len(c.Sources) != 1 || c.Sources[0].ID != "company" {
		t.Fatalf("startup discovery: %+v %v", c.Sources, err)
	}
}

func TestScrapeIntervalBounds(t *testing.T) {
	for _, scope := range []string{"global", "source"} {
		for _, tc := range []struct {
			interval string
			valid    bool
		}{
			{"1s", false},
			{"59s", false},
			{"1m", true},
			{"5m", true},
			{"24h", true},
			{"24h1s", false},
		} {
			t.Run(scope+"/"+tc.interval, func(t *testing.T) {
				body := "sources:\n  - id: company\n    file: company.json\n"
				if scope == "global" {
					body = fmt.Sprintf("scrape_interval: %s\n", tc.interval) + body
				} else {
					body += fmt.Sprintf("    scrape_interval: %s\n", tc.interval)
				}
				c, err := Load(writeConfig(t, body))
				if !tc.valid {
					if err == nil || !strings.Contains(err.Error(), "1m") {
						t.Fatalf("expected interval bounds error, got %v", err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				want, err := time.ParseDuration(tc.interval)
				if err != nil {
					t.Fatal(err)
				}
				if c.Sources[0].ScrapeInterval != Duration(want) {
					t.Fatalf("source interval = %v, want %v", c.Sources[0].ScrapeInterval, want)
				}
			})
		}
	}
}
