package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/url"
	"os"
	"time"

	charmlog "charm.land/log/v2"
	"github.com/charmbracelet/colorprofile"

	"jargonary/internal/config"
)

// newLogger uses Charm's slog handler for both terminal text and JSON lines.
func newLogger(out io.Writer, isTTY bool) *slog.Logger {
	options := charmlog.Options{ReportTimestamp: true, TimeFormat: time.RFC3339, Formatter: charmlog.JSONFormatter}
	profile := colorprofile.NoTTY
	if isTTY {
		options.Formatter = charmlog.TextFormatter
		options.TimeFormat = "15:04:05"
		profile = colorprofile.ANSI256
		if _, noColor := os.LookupEnv("NO_COLOR"); noColor || os.Getenv("TERM") == "dumb" {
			profile = colorprofile.NoTTY
		}
	}
	logger := charmlog.NewWithOptions(out, options)
	logger.SetColorProfile(profile)
	return slog.New(logger)
}

// loggedConfig is indented when formatted as terminal text, but remains a
// structured JSON object in redirected output.
type loggedConfig map[string]any

func (c loggedConfig) String() string {
	b, _ := json.MarshalIndent(map[string]any(c), "", "  ")
	return string(b)
}

func logResolvedConfiguration(logger *slog.Logger, c config.Config, devUI bool) {
	logger.Info("configuration resolved", "dev_ui", devUI,
		"project_glossary_enabled", c.IncludeProjectGlossary(), "config", loggedConfig(c.Diagnostics()))
	if len(c.Sources) == 0 {
		logger.Warn("no glossary sources configured or discovered; serving only the bundled project glossary", "terms_dir", c.TermsDir)
	}
}

func logListenerStarted(logger *slog.Logger, name string, addr net.Addr, publicURL string) {
	// Use the bound address so ephemeral ports are reported correctly.
	host, port, _ := net.SplitHostPort(addr.String())
	if ip := net.ParseIP(host); host == "" || (ip != nil && ip.IsUnspecified()) {
		host = "localhost"
	}
	u := (&url.URL{Scheme: "http", Host: net.JoinHostPort(host, port), Path: "/"}).String()
	if name == "public" && publicURL != "" {
		u = publicURL + "/"
	}
	logger.Info("listener started", "interface", name, "address", addr.String(), "url", u)
}
