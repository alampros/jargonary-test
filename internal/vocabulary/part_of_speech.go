package vocabulary

import (
	"database/sql"
	"database/sql/driver"
	"encoding"
	"errors"
	"fmt"
)

// PartOfSpeech is a grammatical category. Its zero value means unspecified and
// is omitted from authored JSON/YAML. Use ParsePartOfSpeech for external strings;
// a direct Go cast can bypass the type's supported-value checks.
type PartOfSpeech string

const (
	PartOfSpeechUnspecified  PartOfSpeech = ""
	PartOfSpeechNoun         PartOfSpeech = "noun"
	PartOfSpeechPronoun      PartOfSpeech = "pronoun"
	PartOfSpeechVerb         PartOfSpeech = "verb"
	PartOfSpeechAdjective    PartOfSpeech = "adjective"
	PartOfSpeechAdverb       PartOfSpeech = "adverb"
	PartOfSpeechPreposition  PartOfSpeech = "preposition"
	PartOfSpeechConjunction  PartOfSpeech = "conjunction"
	PartOfSpeechInterjection PartOfSpeech = "interjection"
	PartOfSpeechDeterminer   PartOfSpeech = "determiner"
)

var ErrInvalidPartOfSpeech = errors.New("unsupported part of speech")

// PartOfSpeechValues returns the supported, specified categories in display
// order. Each call returns a new slice; callers cannot change the allowed set.
func PartOfSpeechValues() []PartOfSpeech {
	return []PartOfSpeech{
		PartOfSpeechNoun, PartOfSpeechPronoun, PartOfSpeechVerb,
		PartOfSpeechAdjective, PartOfSpeechAdverb, PartOfSpeechPreposition,
		PartOfSpeechConjunction, PartOfSpeechInterjection, PartOfSpeechDeterminer,
	}
}

// ParsePartOfSpeech accepts canonical, case-sensitive values. An empty string
// represents an unspecified category, as in an empty optional CSV cell.
func ParsePartOfSpeech(value string) (PartOfSpeech, error) {
	part := PartOfSpeech(value)
	if !part.Valid() {
		return PartOfSpeechUnspecified, ErrInvalidPartOfSpeech
	}
	return part, nil
}

// Valid reports whether p is supported, including the unspecified zero value.
func (p PartOfSpeech) Valid() bool {
	if p == PartOfSpeechUnspecified {
		return true
	}
	for _, part := range PartOfSpeechValues() {
		if p == part {
			return true
		}
	}
	return false
}

// String returns the canonical text. Use Valid or MarshalText when values might
// have been created by an unchecked cast.
func (p PartOfSpeech) String() string { return string(p) }

// MarshalText validates even values created by unchecked casts. JSON and YAML
// use this interface to keep the document representation as a string.
func (p PartOfSpeech) MarshalText() ([]byte, error) {
	if !p.Valid() {
		return nil, ErrInvalidPartOfSpeech
	}
	return []byte(p), nil
}

// UnmarshalText leaves the receiver unchanged when parsing fails.
func (p *PartOfSpeech) UnmarshalText(text []byte) error {
	if p == nil {
		return errors.New("cannot decode part of speech into a nil receiver")
	}
	part, err := ParsePartOfSpeech(string(text))
	if err != nil {
		return err
	}
	*p = part
	return nil
}

// Value preserves SQLite's existing TEXT representation and empty-string
// convention, rejecting unsupported values before they reach the driver.
func (p PartOfSpeech) Value() (driver.Value, error) {
	if !p.Valid() {
		return nil, ErrInvalidPartOfSpeech
	}
	return p.String(), nil
}

// Scan validates database text before assigning it. NULL and empty text both
// represent an unspecified category; unsupported SQL types are rejected.
func (p *PartOfSpeech) Scan(value any) error {
	switch value := value.(type) {
	case nil:
		return p.UnmarshalText(nil)
	case string:
		return p.UnmarshalText([]byte(value))
	case []byte:
		return p.UnmarshalText(value)
	default:
		return fmt.Errorf("cannot scan part of speech from %T", value)
	}
}

var (
	_ fmt.Stringer             = PartOfSpeechUnspecified
	_ encoding.TextMarshaler   = PartOfSpeechUnspecified
	_ encoding.TextUnmarshaler = (*PartOfSpeech)(nil)
	_ driver.Valuer            = PartOfSpeechUnspecified
	_ sql.Scanner              = (*PartOfSpeech)(nil)
)
