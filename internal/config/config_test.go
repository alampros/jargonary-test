package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if c.Admin.Enabled || c.Server.Listen != "0.0.0.0:8080" || c.Admin.Listen != "127.0.0.1:9090" {
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
