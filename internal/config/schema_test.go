package config

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v3"
)

func TestConfigurationSchema(t *testing.T) {
	data, err := json.Marshal(Schema())
	if err != nil {
		t.Fatal(err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	const location = "https://example.com/config.schema.json"
	if err := compiler.AddResource(location, doc); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile(location)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, yaml string
		valid      bool
	}{
		{"defaults", `{}`, true},
		{"empty sources", `sources: []`, true},
		{"remote source", `sources: [{id: team, url: 'https://example.com/glossary', scrape_interval: 1m30s, auth: {shared_secret: {env: TEAM_SECRET}}}]`, true},
		{"file source", `sources: [{id: team, file: terms.yaml}]`, true},
		{"public export", `exports: [{id: public, name: Public, include: all, auth: {public: true}}]`, true},
		{"filtered export", `exports: [{id: team, name: Team, include: {sources: [team], sense_tags: {any: [public]}}, auth: {shared_secret: {file: secret.txt}}}]`, true},
		{"object all with empty tags", `exports: [{id: team, name: Team, include: {all: true, sense_tags: {}}, auth: {public: true}}]`, true},
		{"object all", `exports: [{id: team, name: Team, include: {all: true}, auth: {public: true}}]`, true},
		{"unknown field", `typo: true`, false},
		{"runtime field", `origins: {}`, false},
		{"nested unknown field", `server: {port: 8080}`, false},
		{"null sources", `sources: null`, false},
		{"integer duration", `timeout: 15`, false},
		{"invalid duration", `timeout: tomorrow`, false},
		{"invalid theme", `theme: missing`, false},
		{"oversized fetch", `max_bytes: 67108865`, false},
		{"missing source location", `sources: [{id: team}]`, false},
		{"both source locations", `sources: [{id: team, file: terms.yaml, url: 'https://example.com'}]`, false},
		{"reserved source", `sources: [{id: project, file: terms.yaml}]`, false},
		{"invalid source id", `sources: [{id: 'bad id', file: terms.yaml}]`, false},
		{"runtime token", `sources: [{id: team, file: terms.yaml, token: abc}]`, false},
		{"source public auth", `sources: [{id: team, file: terms.yaml, auth: {public: true}}]`, false},
		{"blank site name", `site_name: '   '`, false},
		{"blank export name", `exports: [{id: team, name: ' ', include: all, auth: {public: true}}]`, false},
		{"blank tag", `exports: [{id: team, name: Team, include: {sense_tags: {any: [' ']}}, auth: {public: true}}]`, false},
		{"invalid selector id", `exports: [{id: team, name: Team, include: {sources: ['bad id'], sense_tags: {any: [public]}}, auth: {public: true}}]`, false},
		{"empty secret", `sources: [{id: team, url: 'https://example.com', auth: {shared_secret: {}}}]`, false},
		{"hidden asset data", `custom_css_data: []`, false},
		{"invalid include scalar", `exports: [{id: team, name: Team, include: everything, auth: {public: true}}]`, false},
		{"unsupported format", `sources: [{id: team, file: terms.yaml, format: xml}]`, false},
		{"two secret references", `sources: [{id: team, url: 'https://example.com', auth: {shared_secret: {env: TOKEN, file: token}}}]`, false},
		{"missing export auth", `exports: [{id: team, name: Team, include: all}]`, false},
		{"empty export auth", `exports: [{id: team, name: Team, include: all, auth: {}}]`, false},
		{"conflicting auth", `exports: [{id: team, name: Team, include: all, auth: {public: true, shared_secret: {env: TOKEN}}}]`, false},
		{"conflicting selectors", `exports: [{id: team, name: Team, include: {all: true, sources: [team]}, auth: {public: true}}]`, false},
		{"empty selectors", `exports: [{id: team, name: Team, include: {}, auth: {public: true}}]`, false},
		{"empty tags", `exports: [{id: team, name: Team, include: {sense_tags: {any: []}}, auth: {public: true}}]`, false},
		{"reserved selector", `exports: [{id: team, name: Team, include: {sources: [project], sense_tags: {any: [public]}}, auth: {public: true}}]`, false},
	}
	example, err := os.ReadFile("../../examples/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	tests = append(tests, struct {
		name, yaml string
		valid      bool
	}{"example config", string(example), true})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var value any
			if err := yaml.Unmarshal([]byte(tt.yaml), &value); err != nil {
				t.Fatal(err)
			}
			if err := schema.Validate(value); (err == nil) != tt.valid {
				t.Fatalf("valid = %v, want %v: %v", err == nil, tt.valid, err)
			}
		})
	}
}
