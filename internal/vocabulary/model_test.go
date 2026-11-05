package vocabulary

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestProjectGlossary(t *testing.T) {
	b, err := os.ReadFile("../../glossary/project.json")
	if err != nil {
		t.Fatal(err)
	}
	d, err := Parse(b, "json")
	if err != nil {
		t.Fatal(err)
	}
	if d.Name != "Jargonary project terms" {
		t.Fatalf("unexpected project dictionary name: %q", d.Name)
	}
	ids := map[string]bool{}
	for _, e := range d.Entries {
		ids[e.ID] = true
	}
	for _, e := range d.Entries {
		for _, s := range e.Senses {
			for _, r := range s.Relationships {
				if !ids[r.Target.Entry] {
					t.Fatalf("%s has missing target %s", e.ID, r.Target.Entry)
				}
			}
		}
	}
}

func TestInvalidDocuments(t *testing.T) {
	for _, body := range []string{
		`{"version":"2","name":"X","entries":[]}`,
		`{"version":"1","name":"X"}`,
		`{"version":"1","name":"X","entries":null}`,
		`{"version":"1","name":"X","entries":[],"typo":true}`,
		`{"version":"1","name":"X","entries":[],"entries":null}`,
		`{"version":"1","name":"X","entries":[]} {}`,
		`{"version":"1","name":"X","entries":[{"id":"x/y","term":"X","senses":[{"id":"one","definition":"X"}]}]}`,
		`{"version":"1","name":"X","entries":[{"id":"x","term":"X","aliases":[{"text":"Y","sense":"missing"}],"senses":[{"id":"one","definition":"X"}]}]}`,
	} {
		if _, err := Parse([]byte(body), "json"); err == nil {
			t.Errorf("accepted invalid document %s", body)
		}
	}
	if _, err := Parse([]byte("version: '1'\nname: X\nentries: []\n---\nname: other\n"), "yaml"); err == nil {
		t.Fatal("accepted extra YAML document")
	}
	if _, err := Parse([]byte("version: '1'\nname: X\nentries: []\ntypo: true\n"), "yaml"); err == nil {
		t.Fatal("accepted unknown YAML field")
	}
}

func TestFormatsAndMeaning(t *testing.T) {
	yaml := `version: "1"
name: Example
entries:
  - id: component
    term: Component
    pronunciation: kuhm-POH-nuhnt
    aliases:
      - text: Widget
        sense: web
    senses:
      - id: web
        partOfSpeech: noun
        definition: A web element.
      - id: physical
        definition: An asset part.
        deprecated: true
`
	d, err := Parse([]byte(yaml), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Entries[0].Senses) != 2 || !d.Entries[0].Senses[1].Deprecated || d.Entries[0].Pronunciation != "kuhm-POH-nuhnt" || d.Entries[0].Senses[0].PartOfSpeech != "noun" {
		t.Fatal("lost senses or deprecation")
	}
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"entry_id", "term", "sense_id", "definition", "aliases", "tags", "deprecated", "pronunciation", "part_of_speech"})
	_ = w.Write([]string{"component", "Component", "web", "A web element.", `["Widget"]`, `["public"]`, "false", "kuhm-POH-nuhnt", "noun"})
	_ = w.Write([]string{"component", "Component", "physical", "An asset part.", "", "", "true", "kuhm-POH-nuhnt", "noun"})
	w.Flush()
	d, err = Parse([]byte(b.String()), "csv")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Entries) != 1 || len(d.Entries[0].Senses) != 2 || d.Entries[0].Aliases[0].Sense != "web" {
		t.Fatal("CSV lost grouping or alias scope")
	}
	if d.Entries[0].Pronunciation != "kuhm-POH-nuhnt" || d.Entries[0].Senses[0].PartOfSpeech != "noun" {
		t.Fatal("CSV lost lexical metadata")
	}
	encoded, _ := json.Marshal(d)
	roundTrip, err := Parse(encoded, "json")
	if err != nil || roundTrip.Entries[0].Pronunciation != "kuhm-POH-nuhnt" || roundTrip.Entries[0].Senses[0].PartOfSpeech != "noun" {
		t.Fatalf("JSON lost lexical metadata: %+v %v", roundTrip, err)
	}
	if Normalize(" CAFÉ ") != Normalize("cafe\u0301") {
		t.Fatal("normalization must combine Unicode folding and NFC")
	}
}

func TestCSVRejectsConflictingPronunciations(t *testing.T) {
	input := "entry_id,term,sense_id,definition,pronunciation\nread,Read,present,To interpret text.,reed\nread,Read,past,Interpreted text.,red\n"
	if _, err := Parse([]byte(input), "csv"); err == nil {
		t.Fatal("conflicting entry pronunciations accepted; use distinct entries for different pronunciations")
	}
}

func TestPartOfSpeechValues(t *testing.T) {
	valid := []string{"noun", "pronoun", "verb", "adjective", "adverb", "preposition", "conjunction", "interjection", "determiner"}
	for _, part := range append(valid, "noun phrase", "Noun", "unknown", " noun", " ") {
		accepted := false
		for _, allowed := range valid {
			accepted = accepted || part == allowed
		}
		for _, format := range []string{"json", "yaml", "csv"} {
			t.Run(format+"/"+part, func(t *testing.T) {
				var input []byte
				switch format {
				case "json":
					input, _ = json.Marshal(map[string]any{"version": "1", "name": "Parts", "entries": []any{map[string]any{"id": "term", "term": "Term", "senses": []any{map[string]any{"id": "meaning", "definition": "A definition.", "partOfSpeech": part}}}}})
				case "yaml":
					encoded, _ := json.Marshal(part)
					input = []byte("version: '1'\nname: Parts\nentries:\n  - id: term\n    term: Term\n    senses:\n      - id: meaning\n        definition: A definition.\n        partOfSpeech: " + string(encoded) + "\n")
				case "csv":
					var out strings.Builder
					writer := csv.NewWriter(&out)
					_ = writer.WriteAll([][]string{{"entry_id", "term", "sense_id", "definition", "part_of_speech"}, {"term", "Term", "meaning", "A definition.", part}})
					input = []byte(out.String())
				}
				document, err := Parse(input, format)
				if accepted {
					if err != nil || document.Entries[0].Senses[0].PartOfSpeech.String() != part {
						t.Fatalf("valid part of speech rejected or changed: %v", err)
					}
				} else if err == nil {
					t.Fatalf("expected invalid part of speech to be rejected, got %v", err)
				}
			})
		}
	}
}
