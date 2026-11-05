// Package vocabulary defines the portable dictionary document and its validation.
package vocabulary

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"go.yaml.in/yaml/v3"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

type Dictionary struct {
	Version  string  `json:"version" yaml:"version" enum:"1"`
	Name     string  `json:"name" yaml:"name" minLength:"1"`
	Exported bool    `json:"exported,omitempty" yaml:"exported,omitempty"`
	Entries  []Entry `json:"entries" yaml:"entries" nullable:"false"`
}

type Entry struct {
	ID                string         `json:"id" yaml:"id" pattern:"^[A-Za-z0-9][A-Za-z0-9._-]{0,199}$"`
	Origin            *Reference     `json:"origin,omitempty" yaml:"origin,omitempty"`
	Term              string         `json:"term" yaml:"term" minLength:"1"`
	Pronunciation     string         `json:"pronunciation,omitempty" yaml:"pronunciation,omitempty"`
	Aliases           []Alias        `json:"aliases,omitempty" yaml:"aliases,omitempty"`
	Deprecated        bool           `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	DeprecationReason string         `json:"deprecationReason,omitempty" yaml:"deprecationReason,omitempty"`
	Relationships     []Relationship `json:"relationships,omitempty" yaml:"relationships,omitempty"`
	Senses            []Sense        `json:"senses" yaml:"senses" minItems:"1" nullable:"false"`
}

type Alias struct {
	Text  string `json:"text" yaml:"text" minLength:"1"`
	Sense string `json:"sense,omitempty" yaml:"sense,omitempty"`
}

type Sense struct {
	ID                string         `json:"id" yaml:"id" pattern:"^[A-Za-z0-9][A-Za-z0-9._-]{0,199}$"`
	Definition        string         `json:"definition" yaml:"definition" minLength:"1"`
	PartOfSpeech      PartOfSpeech   `json:"partOfSpeech,omitempty" yaml:"partOfSpeech,omitempty"`
	Context           string         `json:"context,omitempty" yaml:"context,omitempty"`
	Examples          []string       `json:"examples,omitempty" yaml:"examples,omitempty"`
	Tags              []string       `json:"tags,omitempty" yaml:"tags,omitempty"`
	Links             []Link         `json:"links,omitempty" yaml:"links,omitempty"`
	Deprecated        bool           `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	DeprecationReason string         `json:"deprecationReason,omitempty" yaml:"deprecationReason,omitempty"`
	Relationships     []Relationship `json:"relationships,omitempty" yaml:"relationships,omitempty"`
}

type Link struct {
	Label string `json:"label" yaml:"label"`
	URL   string `json:"url" yaml:"url"`
}

type Reference struct {
	Source string `json:"source,omitempty" yaml:"source,omitempty"`
	Entry  string `json:"entry" yaml:"entry" pattern:"^[A-Za-z0-9][A-Za-z0-9._-]{0,199}$"`
	Sense  string `json:"sense,omitempty" yaml:"sense,omitempty"`
}

type Relationship struct {
	Type   string    `json:"type" yaml:"type" enum:"abbreviationOf,relatedTo,deprecatedFor"`
	Target Reference `json:"target" yaml:"target"`
}

var Inverse = map[string]string{
	"abbreviationOf": "abbreviatedBy", "abbreviatedBy": "abbreviationOf",
	"relatedTo": "relatedTo", "deprecatedFor": "supersedes", "supersedes": "deprecatedFor",
}

var idPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,199}$`)

func ValidID(id string) bool { return idPattern.MatchString(id) }

func Normalize(text string) string {
	return cases.Fold().String(norm.NFC.String(strings.TrimSpace(text)))
}

func Deprecated(flag bool, relationships []Relationship) bool {
	if flag {
		return true
	}
	for _, r := range relationships {
		if r.Type == "deprecatedFor" {
			return true
		}
	}
	return false
}

func Parse(data []byte, format string) (Dictionary, error) {
	var d Dictionary
	var err error
	switch strings.ToLower(format) {
	case "json", "":
		if err := checkJSON(bytes.NewReader(data)); err != nil {
			return Dictionary{}, errors.New("invalid JSON: duplicate fields, excessive nesting, or malformed document")
		}
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		err = dec.Decode(&d)
		if err == nil {
			var extra any
			if dec.Decode(&extra) != io.EOF {
				err = errors.New("expected one JSON document")
			}
		}
	case "yaml", "yml":
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)
		err = dec.Decode(&d)
		if err == nil {
			var extra any
			if dec.Decode(&extra) != io.EOF {
				err = errors.New("expected one YAML document")
			}
		}
	default:
		return d, errors.New("unsupported dictionary format")
	}
	if err != nil {
		// Parser errors can quote source content. Keep public/operator errors bounded and free of payload text.
		return Dictionary{}, errors.New("cannot parse dictionary; check format, fields, and document syntax")
	}
	if err := d.Validate(); err != nil {
		return Dictionary{}, err
	}
	return d, nil
}

// encoding/json otherwise accepts duplicate keys with last-value-wins behavior.
// Reject ambiguous dictionaries before decoding their typed representation.
func checkJSON(r io.Reader) error {
	dec := json.NewDecoder(r)
	var value func(int) error
	value = func(depth int) error {
		if depth > 64 {
			return errors.New("excessive nesting")
		}
		token, err := dec.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			keys := map[string]bool{}
			for dec.More() {
				key, err := dec.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok || keys[name] {
					return errors.New("duplicate key")
				}
				keys[name] = true
				if err := value(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for dec.More() {
				if err := value(depth + 1); err != nil {
					return err
				}
			}
		default:
			return errors.New("unexpected delimiter")
		}
		_, err = dec.Token()
		return err
	}
	if err := value(0); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		return errors.New("trailing document")
	}
	return nil
}

func (d Dictionary) Validate() error {
	if d.Version != "1" || strings.TrimSpace(d.Name) == "" || d.Entries == nil {
		return errors.New("dictionary requires version 1, a name, and an entries array (which may be empty)")
	}
	seen := map[string]bool{}
	for i, e := range d.Entries {
		if !ValidID(e.ID) || seen[e.ID] || strings.TrimSpace(e.Term) == "" || len(e.Senses) == 0 {
			return fmt.Errorf("entry %d requires a unique valid ID, term, and senses", i+1)
		}
		seen[e.ID] = true
		if d.Exported {
			if e.Origin == nil || !ValidID(e.Origin.Source) || !ValidID(e.Origin.Entry) || e.Origin.Sense != "" {
				return fmt.Errorf("exported entry %d requires origin source and entry IDs", i+1)
			}
		} else if e.Origin != nil {
			return fmt.Errorf("entry %d: origin is reserved for exported dictionaries", i+1)
		}
		senses := map[string]bool{}
		for j, s := range e.Senses {
			if !ValidID(s.ID) || senses[s.ID] || strings.TrimSpace(s.Definition) == "" {
				return fmt.Errorf("entry %d sense %d requires a unique valid ID and definition", i+1, j+1)
			}
			senses[s.ID] = true
			if !s.PartOfSpeech.Valid() {
				return fmt.Errorf("entry %d sense %d: %w", i+1, j+1, ErrInvalidPartOfSpeech)
			}
			if err := validateRelationships(s.Relationships); err != nil {
				return fmt.Errorf("entry %d sense %d: %w", i+1, j+1, err)
			}
			for _, link := range s.Links {
				u, err := url.Parse(link.URL)
				if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil {
					return fmt.Errorf("entry %d sense %d contains an invalid link; use HTTP(S) without credentials", i+1, j+1)
				}
			}
		}
		for _, a := range e.Aliases {
			if strings.TrimSpace(a.Text) == "" || (a.Sense != "" && !senses[a.Sense]) {
				return fmt.Errorf("entry %d has an empty alias or unknown alias sense", i+1)
			}
		}
		if err := validateRelationships(e.Relationships); err != nil {
			return fmt.Errorf("entry %d: %w", i+1, err)
		}
	}
	return nil
}

func validateRelationships(rs []Relationship) error {
	for _, r := range rs {
		if (r.Type != "abbreviationOf" && r.Type != "relatedTo" && r.Type != "deprecatedFor") || !ValidID(r.Target.Entry) ||
			(r.Target.Source != "" && !ValidID(r.Target.Source)) || (r.Target.Sense != "" && !ValidID(r.Target.Sense)) {
			return errors.New("relationship requires a supported type and valid target IDs")
		}
	}
	return nil
}

// FirstLetter selects the glossary navigation bucket for an entry heading.
func FirstLetter(term string) string {
	for _, r := range strings.TrimSpace(term) {
		if r >= 'a' && r <= 'z' {
			return strings.ToUpper(string(r))
		}
		if r >= 'A' && r <= 'Z' {
			return string(r)
		}
		if unicode.IsDigit(r) {
			return "0-9"
		}
		return "#"
	}
	return "#"
}
