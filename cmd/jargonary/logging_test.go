package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jargonary/internal/config"
)

func TestLogResolvedConfiguration(t *testing.T) {
	const secret = "a-secret-that-must-never-appear-in-logs"
	t.Setenv("JARGONARY_TEST_LOG_SECRET", secret)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(`database: cache.db
terms_dir: terms
sources:
  - id: remote
    url: https://example.test/terms.json?token=hidden-query
    auth:
      shared_secret:
        env: JARGONARY_TEST_LOG_SECRET
exports:
  - id: shared
    name: Shared glossary
    include: all
    auth:
      shared_secret:
        env: JARGONARY_TEST_LOG_SECRET
`), 0600)
	if err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Sources[0].Token != secret || c.Exports[0].Token != secret {
		t.Fatal("expected resolved source and export secrets")
	}
	var out bytes.Buffer
	logResolvedConfiguration(newLogger(&out, false), c, true)
	if strings.Count(out.String(), "\n") != 1 || strings.Contains(out.String(), "\x1b[") {
		t.Fatalf("configuration is not a single JSON line: %s", out.String())
	}
	for _, hidden := range []string{secret, "hidden-query", "?token="} {
		if strings.Contains(out.String(), hidden) {
			t.Fatalf("log contains sensitive value %q", hidden)
		}
	}
	var record struct {
		Config struct {
			Database string
			Server   config.Listener
			Sources  []struct {
				URL            string
				ScrapeInterval string `json:"scrape_interval"`
				Auth           config.Auth
			}
			Origins map[string]string
		}
		DevUI          bool `json:"dev_ui"`
		ProjectEnabled bool `json:"project_glossary_enabled"`
	}
	if err := json.Unmarshal(out.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record.Config.Database != filepath.Join(dir, "cache.db") || record.Config.Server.Listen != "0.0.0.0:8080" || !record.DevUI || !record.ProjectEnabled {
		t.Fatalf("missing resolved startup settings: %s", out.String())
	}
	source := record.Config.Sources[0]
	if source.URL != "https://example.test/terms.json" || source.ScrapeInterval != "5m0s" || source.Auth.SharedSecret.Env != "JARGONARY_TEST_LOG_SECRET" || record.Config.Origins["database"] != "configuration file" {
		t.Fatalf("missing resolved source settings or origins: %s", out.String())
	}
	out.Reset()
	t.Setenv("NO_COLOR", "1")
	logResolvedConfiguration(newLogger(&out, true), c, true)
	if !strings.Contains(out.String(), "│ {\n  │   \"admin\": {") {
		t.Fatalf("terminal configuration is not indented: %s", out.String())
	}
	for _, hidden := range []string{secret, "hidden-query", "?token="} {
		if strings.Contains(out.String(), hidden) {
			t.Fatalf("terminal log contains sensitive value %q", hidden)
		}
	}
}

func TestLoggerOutputModes(t *testing.T) {
	for _, tc := range []struct {
		name, term              string
		tty, noColor, wantColor bool
	}{
		{"terminal", "xterm-256color", true, false, true},
		{"redirected", "xterm-256color", false, false, false},
		{"no color", "xterm-256color", true, true, false},
		{"dumb terminal", "dumb", true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TERM", tc.term)
			t.Setenv("NO_COLOR", "")
			if !tc.noColor {
				if err := os.Unsetenv("NO_COLOR"); err != nil {
					t.Fatal(err)
				}
			}
			var out bytes.Buffer
			logger := newLogger(&out, tc.tty)
			logger.Info("listener started", "url", "http://localhost:8080/")
			logger.Warn("warning message")
			logger.Error("error message")
			if strings.Contains(out.String(), "\x1b[") != tc.wantColor {
				t.Fatalf("unexpected color output: %q", out.String())
			}
			if !strings.Contains(out.String(), "http://localhost:8080/") {
				t.Fatalf("missing URL: %q", out.String())
			}
			if tc.tty {
				if strings.HasPrefix(out.String(), "{") || !strings.Contains(out.String(), "listener started ") {
					t.Fatalf("expected human-readable terminal output: %q", out.String())
				}
			} else {
				for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
					if !json.Valid([]byte(line)) {
						t.Fatalf("invalid JSON log line: %q", line)
					}
				}
			}
		})
	}
}

func TestNoSourcesStartupWarning(t *testing.T) {
	for _, tc := range []struct {
		name, settings string
		discovered     bool
		wantWarning    bool
	}{
		{"empty discovery", "", false, true},
		{"explicit empty sources", "sources: []\n", false, true},
		{"hidden glossary fallback", "sources: []\nhide_jargonary_glossary: true\n", false, true},
		{"configured source", "sources:\n  - id: local\n    file: local.json\n", false, false},
		{"discovered source", "", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			termsDir := filepath.Join(dir, "terms")
			if err := os.Mkdir(termsDir, 0700); err != nil {
				t.Fatal(err)
			}
			if tc.discovered {
				if err := os.WriteFile(filepath.Join(termsDir, "local.json"), []byte(`{"version":"1","name":"Local","entries":[]}`), 0600); err != nil {
					t.Fatal(err)
				}
			}
			path := filepath.Join(dir, "config.yaml")
			if err := os.WriteFile(path, []byte("terms_dir: terms\n"+tc.settings), 0600); err != nil {
				t.Fatal(err)
			}
			c, err := config.Load(path)
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			logResolvedConfiguration(newLogger(&out, false), c, false)
			warnings := 0
			for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
				var record struct {
					Level    string
					Msg      string
					TermsDir string `json:"terms_dir"`
				}
				if err := json.Unmarshal([]byte(line), &record); err != nil {
					t.Fatal(err)
				}
				if strings.EqualFold(record.Level, "warn") {
					warnings++
					if !strings.Contains(record.Msg, "serving only the bundled project glossary") || record.TermsDir != termsDir {
						t.Fatalf("warning missing fallback or resolved terms directory: %s", line)
					}
				}
			}
			want := 0
			if tc.wantWarning {
				want = 1
			}
			if warnings != want {
				t.Fatalf("got %d warnings, want %d: %s", warnings, want, out.String())
			}
		})
	}
}

func TestLogListenerStarted(t *testing.T) {
	for _, tc := range []struct {
		name, address, publicURL, want string
	}{
		{"public", "0.0.0.0:8080", "", "http://localhost:8080/"},
		{"public", "[::]:8080", "", "http://localhost:8080/"},
		{"public", "127.0.0.1:8081", "", "http://127.0.0.1:8081/"},
		{"admin", "[::1]:9090", "", "http://[::1]:9090/"},
		{"public", "192.0.2.1:8080", "", "http://192.0.2.1:8080/"},
		{"public", "0.0.0.0:8080", "https://glossary.example.test/team", "https://glossary.example.test/team/"},
		{"admin", "127.0.0.1:9090", "https://glossary.example.test/team", "http://127.0.0.1:9090/"},
		{"mcp", "127.0.0.1:9091", "", "http://127.0.0.1:9091/"},
	} {
		t.Run(tc.name+tc.address+tc.publicURL, func(t *testing.T) {
			addr, err := net.ResolveTCPAddr("tcp", tc.address)
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			logListenerStarted(slog.New(slog.NewJSONHandler(&out, nil)), tc.name, addr, tc.publicURL)
			var record map[string]string
			if err := json.Unmarshal(out.Bytes(), &record); err != nil {
				t.Fatal(err)
			}
			if record["url"] != tc.want || record["interface"] != tc.name || record["address"] != tc.address {
				t.Fatalf("unexpected listener log: %s", out.String())
			}
		})
	}
}
