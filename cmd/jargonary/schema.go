package main

import (
	"encoding/json"
	"errors"
	"io"
	"reflect"

	"github.com/danielgtaylor/huma/v2"
	"github.com/invopop/jsonschema"
	"golang.org/x/mod/semver"

	"jargonary/internal/config"
	"jargonary/internal/vocabulary"
)

// writeSchema keeps release and publishing metadata out of the config model.
func writeSchema(w io.Writer, version string, configuration bool) error {
	if version == "" {
		return errors.New("-schema-version is required with -schema or -config-schema")
	}
	if !semver.IsValid(version) || semver.Canonical(version) != version {
		return errors.New("-schema-version must be a full release version such as v1.2.3 or v1.2.3-rc.1")
	}
	base := "https://alampros.github.io/jargonary/schema/" + version + "/"
	var schema any
	if configuration {
		s := config.Schema()
		s.ID = jsonschema.ID(base + "config.schema.json")
		schema = s
	} else {
		registry := huma.NewMapRegistry("#/$defs/", huma.DefaultSchemaNamer)
		root := registry.Schema(reflect.TypeFor[vocabulary.Dictionary](), true, "")
		schema = map[string]any{
			"$id":     base + "glossary.schema.json",
			"$schema": "https://json-schema.org/draft/2020-12/schema",
			"$ref":    root.Ref,
			"$defs":   registry.Map(),
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(schema)
}
