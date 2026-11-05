package vocabulary

import (
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"go.yaml.in/yaml/v3"
	_ "modernc.org/sqlite"
)

func TestPartOfSpeechConversions(t *testing.T) {
	values := append([]PartOfSpeech{PartOfSpeechUnspecified}, PartOfSpeechValues()...)
	for _, part := range values {
		t.Run(part.String(), func(t *testing.T) {
			if !part.Valid() {
				t.Fatal("supported value is invalid")
			}
			parsed, err := ParsePartOfSpeech(part.String())
			if err != nil || parsed != part {
				t.Fatalf("string round trip: %v %v", parsed, err)
			}
			text, err := part.MarshalText()
			var fromText PartOfSpeech
			if err != nil {
				t.Fatal(err)
			}
			if err := fromText.UnmarshalText(text); err != nil || fromText != part {
				t.Fatalf("text round trip: %v %v", fromText, err)
			}
			for _, format := range []string{"json", "yaml"} {
				t.Run(format, func(t *testing.T) {
					var data []byte
					var fromDocument PartOfSpeech
					var err error
					if format == "json" {
						data, err = json.Marshal(part)
						if err == nil {
							err = json.Unmarshal(data, &fromDocument)
						}
					} else {
						data, err = yaml.Marshal(part)
						if err == nil {
							err = yaml.Unmarshal(data, &fromDocument)
						}
					}
					if err != nil || fromDocument != part {
						t.Fatalf("document round trip: %s %v %v", data, fromDocument, err)
					}
				})
			}
		})
	}
}

func TestPartOfSpeechRejectsUnknownValues(t *testing.T) {
	for _, text := range []string{"unknown", "Noun", "noun phrase", " noun", "noun ", "名詞"} {
		t.Run(text, func(t *testing.T) {
			if _, err := ParsePartOfSpeech(text); !errors.Is(err, ErrInvalidPartOfSpeech) {
				t.Fatalf("unchecked string conversion: %v", err)
			}
			part := PartOfSpeechNoun
			if err := part.UnmarshalText([]byte(text)); !errors.Is(err, ErrInvalidPartOfSpeech) || part != PartOfSpeechNoun {
				t.Fatalf("failed text conversion changed receiver: %v %v", part, err)
			}
			encoded, _ := json.Marshal(text)
			if err := json.Unmarshal(encoded, &part); !errors.Is(err, ErrInvalidPartOfSpeech) || part != PartOfSpeechNoun {
				t.Fatalf("failed JSON conversion changed receiver: %v %v", part, err)
			}
			if err := yaml.Unmarshal(encoded, &part); !errors.Is(err, ErrInvalidPartOfSpeech) || part != PartOfSpeechNoun {
				t.Fatalf("failed YAML conversion changed receiver: %v %v", part, err)
			}
			// Explicit casts remain legal Go. Guard every serialization boundary.
			invalid := PartOfSpeech(text)
			if invalid.Valid() {
				t.Fatal("unchecked cast is considered valid")
			}
			if _, err := invalid.MarshalText(); !errors.Is(err, ErrInvalidPartOfSpeech) {
				t.Fatalf("invalid text serialization: %v", err)
			}
			if _, err := json.Marshal(invalid); !errors.Is(err, ErrInvalidPartOfSpeech) {
				t.Fatalf("invalid JSON serialization: %v", err)
			}
			if _, err := yaml.Marshal(invalid); !errors.Is(err, ErrInvalidPartOfSpeech) {
				t.Fatalf("invalid YAML serialization: %v", err)
			}
			if _, err := invalid.Value(); !errors.Is(err, ErrInvalidPartOfSpeech) {
				t.Fatalf("invalid SQL serialization: %v", err)
			}
		})
	}
}

func TestPartOfSpeechOptionalField(t *testing.T) {
	value := struct {
		Part PartOfSpeech `json:"partOfSpeech,omitempty" yaml:"partOfSpeech,omitempty"`
	}{}
	for _, marshal := range []func(any) ([]byte, error){json.Marshal, yaml.Marshal} {
		data, err := marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]any
		if err := yaml.Unmarshal(data, &fields); err != nil || len(fields) != 0 {
			t.Fatalf("unspecified field must be omitted: %s %v", data, err)
		}
	}
	parts := PartOfSpeechValues()
	if len(parts) != 9 {
		t.Fatalf("expected nine specified categories, got %v", parts)
	}
	parts[0] = PartOfSpeech("corrupt")
	if PartOfSpeechValues()[0] != PartOfSpeechNoun || !PartOfSpeechNoun.Valid() {
		t.Fatal("caller mutated the enum's supported values")
	}
}

func TestPartOfSpeechSQLConversions(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, part := range append([]PartOfSpeech{PartOfSpeechUnspecified}, PartOfSpeechValues()...) {
		var fromDatabase PartOfSpeech
		if err := db.QueryRow("SELECT ?", part).Scan(&fromDatabase); err != nil || fromDatabase != part {
			t.Fatalf("SQL round trip for %q: %v %v", part, fromDatabase, err)
		}
	}
	if _, err := db.Exec("SELECT ?", PartOfSpeech("unknown")); err == nil {
		t.Fatal("SQL driver accepted an invalid cast")
	}
	part := PartOfSpeechVerb
	for _, value := range []any{"unknown", []byte("unknown"), 42, true} {
		if err := part.Scan(value); err == nil || part != PartOfSpeechVerb {
			t.Fatalf("failed SQL conversion changed receiver: %v %v", part, err)
		}
	}
	if err := db.QueryRow("SELECT 'unknown'").Scan(&part); err == nil || part != PartOfSpeechVerb {
		t.Fatalf("database read accepted invalid text or changed receiver: %v %v", part, err)
	}
	if err := part.Scan([]byte("adjective")); err != nil || part != PartOfSpeechAdjective {
		t.Fatalf("SQL byte conversion failed: %v %v", part, err)
	}
	if err := db.QueryRow("SELECT NULL").Scan(&part); err != nil || part != PartOfSpeechUnspecified {
		t.Fatalf("SQL NULL must be unspecified: %v %v", part, err)
	}
}

func TestPartOfSpeechValidationRejectsUncheckedCast(t *testing.T) {
	document := Dictionary{Version: "1", Name: "Team", Entries: []Entry{{ID: "api", Term: "API", Senses: []Sense{{ID: "software", Definition: "An interface.", PartOfSpeech: PartOfSpeech("unknown")}}}}}
	if err := document.Validate(); !errors.Is(err, ErrInvalidPartOfSpeech) {
		t.Fatalf("dictionary validation accepted an invalid cast: %v", err)
	}
}

func TestPartOfSpeechRejectsNonStringJSON(t *testing.T) {
	for _, input := range []string{"42", "true", "{}", "[]"} {
		part := PartOfSpeechVerb
		if err := json.Unmarshal([]byte(input), &part); err == nil || part != PartOfSpeechVerb {
			t.Fatalf("invalid JSON %s changed the enum or was accepted: %v %v", input, part, err)
		}
	}
}
