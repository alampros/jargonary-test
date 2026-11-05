package vocabulary

import "github.com/danielgtaylor/huma/v2"

// Schema keeps generated glossary and API enums in sync with the Go type.
// Unspecified is represented by an omitted optional property, not an enum item.
func (PartOfSpeech) Schema(huma.Registry) *huma.Schema {
	parts := PartOfSpeechValues()
	values := make([]any, len(parts))
	for i, part := range parts {
		values[i] = part.String()
	}
	return &huma.Schema{Type: huma.TypeString, Enum: values}
}
