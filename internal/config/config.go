// Package config resolves startup configuration. Runtime reload is intentionally deferred.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
	"jargonary/internal/vocabulary"
)

type Duration time.Duration

func (d *Duration) UnmarshalYAML(n *yaml.Node) error {
	v, err := time.ParseDuration(n.Value)
	*d = Duration(v)
	return err
}

func (d Duration) MarshalJSON() ([]byte, error) { return json.Marshal(time.Duration(d).String()) }

type Listener struct {
	Listen string `yaml:"listen" json:"listen"`
}
type Admin struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Listen  string `yaml:"listen" json:"listen"`
}
type Secret struct {
	Env  string `yaml:"env" json:"env,omitempty"`
	File string `yaml:"file" json:"file,omitempty"`
}
type Auth struct {
	Public       bool    `yaml:"public" json:"public,omitempty"`
	SharedSecret *Secret `yaml:"shared_secret" json:"shared_secret,omitempty"`
}
type Source struct {
	ID             string   `yaml:"id" json:"id"`
	URL            string   `yaml:"url" json:"-"`
	File           string   `yaml:"file" json:"file,omitempty"`
	Format         string   `yaml:"format" json:"format"`
	ScrapeInterval Duration `yaml:"scrape_interval" json:"scrape_interval"`
	Auth           Auth     `yaml:"auth" json:"auth"`
	Token          string   `yaml:"-" json:"-"`
}
type Tags struct {
	Any []string `yaml:"any" json:"any"`
}
type Include struct {
	All       bool     `yaml:"all" json:"all,omitempty"`
	SenseTags Tags     `yaml:"sense_tags" json:"sense_tags"`
	Sources   []string `yaml:"sources" json:"sources,omitempty"`
}

func (i *Include) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		if n.Value != "all" {
			return errors.New("include scalar must be all")
		}
		i.All = true
		return nil
	}
	// Re-enter the strict decoder rather than Node.Decode, which ignores unknown fields.
	type plain Include
	b, err := yaml.Marshal(n)
	if err != nil {
		return err
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	return dec.Decode((*plain)(i))
}

type Export struct {
	ID      string  `yaml:"id" json:"id"`
	Name    string  `yaml:"name" json:"name"`
	Include Include `yaml:"include" json:"include"`
	Auth    Auth    `yaml:"auth" json:"auth"`
	Token   string  `yaml:"-" json:"-"`
}
type Config struct {
	Theme                 string            `yaml:"theme" json:"theme"`
	CustomCSS             []string          `yaml:"custom_css" json:"custom_css"`
	CustomJS              []string          `yaml:"custom_js" json:"custom_js"`
	CustomCSSData         [][]byte          `yaml:"-" json:"-"`
	CustomJSData          [][]byte          `yaml:"-" json:"-"`
	HideJargonaryGlossary bool              `yaml:"hide_jargonary_glossary" json:"hide_jargonary_glossary"`
	SiteName              string            `yaml:"site_name" json:"site_name"`
	Server                Listener          `yaml:"server" json:"server"`
	Admin                 Admin             `yaml:"admin" json:"admin"`
	Database              string            `yaml:"database" json:"database"`
	TermsDir              string            `yaml:"terms_dir" json:"terms_dir"`
	PublicURL             string            `yaml:"public_url" json:"public_url,omitempty"`
	ScrapeInterval        Duration          `yaml:"scrape_interval" json:"scrape_interval"`
	Timeout               Duration          `yaml:"timeout" json:"timeout"`
	MaxBytes              int64             `yaml:"max_bytes" json:"max_bytes"`
	Sources               []Source          `yaml:"sources" json:"sources"`
	Exports               []Export          `yaml:"exports" json:"exports"`
	Origins               map[string]string `yaml:"-" json:"origins"`
}

// IncludeProjectGlossary keeps the bundled source as a fallback when no company
// sources are configured or discovered, irrespective of scrape success.
func (c Config) IncludeProjectGlossary() bool {
	return !c.HideJargonaryGlossary || len(c.Sources) == 0
}

func Load(path string) (Config, error) {
	c := Config{SiteName: "Jargonary", Theme: "default", Server: Listener{Listen: "0.0.0.0:8080"}, Admin: Admin{Listen: "127.0.0.1:9090"}, Database: "data/glossary.db", TermsDir: "/terms", ScrapeInterval: Duration(5 * time.Minute), Timeout: Duration(15 * time.Second), MaxBytes: 8 << 20, Origins: map[string]string{"defaults": "built-in"}}
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return c, errors.New("cannot read configuration file")
		}
		dec := yaml.NewDecoder(bytes.NewReader(b))
		dec.KnownFields(true)
		if err := dec.Decode(&c); err != nil {
			return c, errors.New("invalid configuration: check field names and value types")
		}
		var extra any
		if dec.Decode(&extra) != io.EOF {
			return c, errors.New("configuration must contain one document")
		}
		// Paths are relative to the configuration file, not the executable's working directory.
		base, err := filepath.Abs(filepath.Dir(path))
		if err != nil {
			return c, err
		}
		resolve := func(p string) string {
			if p == "" || filepath.IsAbs(p) {
				return p
			}
			return filepath.Join(base, p)
		}
		c.Database = resolve(c.Database)
		c.TermsDir = resolve(c.TermsDir)
		for i := range c.Sources {
			c.Sources[i].File = resolve(c.Sources[i].File)
			if s := c.Sources[i].Auth.SharedSecret; s != nil {
				s.File = resolve(s.File)
			}
		}
		for i := range c.Exports {
			if s := c.Exports[i].Auth.SharedSecret; s != nil {
				s.File = resolve(s.File)
			}
		}
		var node yaml.Node
		if yaml.Unmarshal(b, &node) == nil && len(node.Content) > 0 {
			recordOrigins(node.Content[0], "", c.Origins)
		}
	}
	if err := c.resolveAppearance(path); err != nil {
		return c, err
	}
	if _, explicit := c.Origins["sources"]; explicit && c.Sources == nil {
		return c, errors.New("sources must be an array; use [] to explicitly disable discovery")
	}
	if c.Sources == nil {
		files, err := os.ReadDir(c.TermsDir)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return c, errors.New("cannot discover terms directory")
		}
		for _, f := range files {
			if f.IsDir() || f.Type()&os.ModeSymlink != 0 {
				continue
			}
			format := strings.TrimPrefix(strings.ToLower(filepath.Ext(f.Name())), ".")
			if format != "json" && format != "yaml" && format != "yml" && format != "csv" {
				continue
			}
			id := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
			c.Sources = append(c.Sources, Source{ID: id, File: filepath.Join(c.TermsDir, f.Name()), Format: format})
		}
		if len(c.Sources) > 0 {
			c.Origins["sources"] = "terms-directory discovery (startup only)"
		}
	}
	if err := c.Validate(); err != nil {
		return c, err
	}
	return c, nil
}

func recordOrigins(n *yaml.Node, prefix string, out map[string]string) {
	if n.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i < len(n.Content); i += 2 {
		key := n.Content[i].Value
		if prefix != "" {
			key = prefix + "." + key
		}
		out[key] = "configuration file"
		recordOrigins(n.Content[i+1], key, out)
	}
}

func (c *Config) Validate() error {
	if err := c.validateAppearance(); err != nil {
		return err
	}
	c.SiteName = strings.TrimSpace(c.SiteName)
	if c.SiteName == "" || utf8.RuneCountInString(c.SiteName) > 120 {
		return errors.New("site_name must contain 1–120 characters")
	}
	for _, addr := range []string{c.Server.Listen, c.Admin.Listen} {
		if _, _, err := net.SplitHostPort(addr); err != nil {
			return errors.New("listeners require host:port addresses")
		}
	}
	if c.Admin.Enabled && c.Admin.Listen == c.Server.Listen {
		return errors.New("public and admin listen addresses must differ")
	}
	if c.Database == "" || c.ScrapeInterval < Duration(time.Second) || c.ScrapeInterval > Duration(24*time.Hour) || c.Timeout < Duration(time.Second) || c.Timeout > Duration(time.Minute) || c.MaxBytes < 1 || c.MaxBytes > 64<<20 {
		return errors.New("invalid database, interval, timeout (1s–1m), or max_bytes (1–64 MiB)")
	}
	if c.PublicURL != "" {
		u, err := url.Parse(c.PublicURL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return errors.New("public_url must be an HTTP(S) base URL without credentials, query, or fragment")
		}
		c.PublicURL = strings.TrimRight(c.PublicURL, "/")
	}
	ids := map[string]bool{"project": true}
	for i := range c.Sources {
		s := &c.Sources[i]
		if !vocabulary.ValidID(s.ID) || ids[s.ID] {
			return fmt.Errorf("source %d has an invalid, duplicate, or reserved ID", i+1)
		}
		ids[s.ID] = true
		if (s.URL == "") == (s.File == "") {
			return fmt.Errorf("source %d requires exactly one URL or file", i+1)
		}
		if s.Format == "" {
			location := s.File
			if s.URL != "" {
				u, err := url.Parse(s.URL)
				if err != nil {
					return errors.New("invalid source URL")
				}
				location = u.Path
			}
			s.Format = strings.TrimPrefix(strings.ToLower(filepath.Ext(location)), ".")
			if s.Format == "" {
				s.Format = "json"
			}
		}
		if s.Format != "json" && s.Format != "yaml" && s.Format != "yml" && s.Format != "csv" {
			return errors.New("unsupported source format")
		}
		if s.ScrapeInterval == 0 {
			s.ScrapeInterval = c.ScrapeInterval
		}
		if s.ScrapeInterval < Duration(time.Second) || s.ScrapeInterval > Duration(24*time.Hour) {
			return errors.New("source interval must be between 1s and 24h")
		}
		if s.URL != "" {
			u, err := url.Parse(s.URL)
			if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") {
				return errors.New("source URL must be HTTP(S), without credentials or fragment")
			}
		}
		if s.Auth.Public {
			return errors.New("public auth policy applies to exports, not sources")
		}
		token, err := resolveSecret(s.Auth)
		if err != nil {
			return fmt.Errorf("source %d: %w", i+1, err)
		}
		s.Token = token
		if token != "" && (!strings.HasPrefix(s.URL, "https://")) {
			return errors.New("authenticated sources require HTTPS")
		}
	}
	exports := map[string]bool{}
	for i := range c.Exports {
		e := &c.Exports[i]
		if !vocabulary.ValidID(e.ID) || exports[e.ID] || strings.TrimSpace(e.Name) == "" {
			return fmt.Errorf("export %d requires a unique valid ID and name", i+1)
		}
		exports[e.ID] = true
		if e.Include.All && (len(e.Include.SenseTags.Any) > 0 || len(e.Include.Sources) > 0) {
			return errors.New("include all cannot combine with other selectors")
		}
		if !e.Include.All && len(e.Include.SenseTags.Any) == 0 {
			return errors.New("export requires include: all or nonempty sense_tags.any")
		}
		for _, tag := range e.Include.SenseTags.Any {
			if strings.TrimSpace(tag) == "" {
				return errors.New("export tag selectors cannot be empty")
			}
		}
		for _, id := range e.Include.Sources {
			if id == "project" && !c.IncludeProjectGlossary() {
				return errors.New("export selector references disabled project glossary")
			}
			if !ids[id] {
				return errors.New("export selector references unknown source")
			}
		}
		if !e.Auth.Public && e.Auth.SharedSecret == nil {
			return errors.New("export requires explicit public or shared-secret access policy")
		}
		token, err := resolveSecret(e.Auth)
		if err != nil {
			return fmt.Errorf("export %d: %w", i+1, err)
		}
		e.Token = token
	}
	sort.Slice(c.Sources, func(i, j int) bool { return c.Sources[i].ID < c.Sources[j].ID })
	return nil
}

func resolveSecret(a Auth) (string, error) {
	if a.Public && a.SharedSecret != nil {
		return "", errors.New("access policy must choose public or shared_secret")
	}
	if a.SharedSecret == nil {
		return "", nil
	}
	s := a.SharedSecret
	if (s.Env == "") == (s.File == "") {
		return "", errors.New("shared_secret requires exactly one env or file reference")
	}
	var value string
	if s.Env != "" {
		value = os.Getenv(s.Env)
	} else {
		b, err := os.ReadFile(s.File)
		if err != nil {
			return "", errors.New("cannot read secret file")
		}
		value = strings.TrimSpace(string(b))
	}
	if len(value) < 32 || strings.ContainsAny(value, "\r\n\t ") {
		return "", errors.New("shared secret must contain at least 32 characters without whitespace")
	}
	return value, nil
}

// Diagnostics returns a copy without URL query strings or secret values. No raw configuration is served.
func (c Config) Diagnostics() map[string]any {
	b, _ := json.Marshal(c)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	sources, _ := out["sources"].([]any)
	for i, s := range c.Sources {
		if s.URL != "" {
			u, _ := url.Parse(s.URL)
			u.RawQuery = ""
			u.Fragment = ""
			sources[i].(map[string]any)["url"] = u.String()
		}
	}
	return out
}
