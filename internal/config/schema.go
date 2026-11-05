package config

import (
	"github.com/invopop/jsonschema"
	orderedmap "github.com/pb33f/ordered-map/v2"

	"jargonary/internal/themes"
)

// Schema describes authored YAML configuration, rather than the redacted JSON
// diagnostics. Startup validation also checks files, secrets, and semantic rules.
func Schema() *jsonschema.Schema {
	r := jsonschema.Reflector{
		FieldNameTag:               "yaml",
		RequiredFromJSONSchemaTags: true,
		Anonymous:                  true,
	}
	s := r.Reflect(&Config{})
	s.Title = "Jargonary configuration"
	return s
}

func (Duration) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:        "string",
		Pattern:     `^[+-]?(0|([0-9]+(\.[0-9]*)?|\.[0-9]+)(ns|us|µs|μs|ms|s|m|h))(([0-9]+(\.[0-9]*)?|\.[0-9]+)(ns|us|µs|μs|ms|s|m|h))*$`,
		Description: "Go duration string, for example 5m or 1m30s. Startup validates the allowed range.",
	}
}

func (Config) JSONSchemaExtend(s *jsonschema.Schema) {
	theme := s.Properties.Value("theme")
	theme.Enum = []any{""}
	for _, name := range themes.Names {
		theme.Enum = append(theme.Enum, name)
	}
}

func (Source) JSONSchemaExtend(s *jsonschema.Schema) {
	s.Properties.Value("id").Not = &jsonschema.Schema{Const: "project"}
	// Sources may fetch with a secret but cannot declare public export access.
	s.AllOf = []*jsonschema.Schema{
		propertyRule("auth", propertyRule("public", &jsonschema.Schema{Const: false})),
	}
}

func (Auth) JSONSchemaExtend(s *jsonschema.Schema) {
	conflict := propertyRule("public", &jsonschema.Schema{Const: true})
	conflict.Required = []string{"public", "shared_secret"}
	s.Not = conflict
}

func (Export) JSONSchemaExtend(s *jsonschema.Schema) {
	public := propertyRule("public", &jsonschema.Schema{Const: true})
	public.Required = []string{"public"}
	s.AllOf = []*jsonschema.Schema{
		propertyRule("auth", &jsonschema.Schema{AnyOf: []*jsonschema.Schema{
			public,
			{Required: []string{"shared_secret"}},
		}}),
	}
}

func (Include) JSONSchemaExtend(s *jsonschema.Schema) {
	// Preserve the reflected object as one branch; YAML also accepts "all".
	selectors := *s
	selectors.Properties.Value("sources").Items.Not = &jsonschema.Schema{Const: "project"}
	zero, one := uint64(0), uint64(1)
	all := propertyRule("all", &jsonschema.Schema{Const: true})
	all.Required = []string{"all"}
	all.AllOf = []*jsonschema.Schema{
		propertyRule("sources", &jsonschema.Schema{MaxItems: &zero}),
		propertyRule("sense_tags", propertyRule("any", &jsonschema.Schema{MaxItems: &zero})),
	}
	tags := propertyRule("any", &jsonschema.Schema{MinItems: &one})
	tags.Required = []string{"any"}
	filtered := propertyRule("sense_tags", tags)
	filtered.Required = []string{"sense_tags"}
	filtered.AllOf = []*jsonschema.Schema{propertyRule("all", &jsonschema.Schema{Const: false})}
	selectors.AnyOf = []*jsonschema.Schema{all, filtered}
	*s = jsonschema.Schema{OneOf: []*jsonschema.Schema{
		{Const: "all"},
		&selectors,
	}}
}

// propertyRule constrains a property without making it required or closing the
// object. The reflected schema supplies its type and allowed field names.
func propertyRule(name string, rule *jsonschema.Schema) *jsonschema.Schema {
	properties := orderedmap.New[string, *jsonschema.Schema]()
	properties.Set(name, rule)
	return &jsonschema.Schema{Properties: properties}
}
