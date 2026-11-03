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
    aliases:
      - text: Widget
        sense: web
    senses:
      - id: web
        definition: A web element.
      - id: physical
        definition: An asset part.
        deprecated: true
`
	d, err := Parse([]byte(yaml), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Entries[0].Senses) != 2 || !d.Entries[0].Senses[1].Deprecated {
		t.Fatal("lost senses or deprecation")
	}
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"entry_id", "term", "sense_id", "definition", "aliases", "tags", "deprecated"})
	_ = w.Write([]string{"component", "Component", "web", "A web element.", `["Widget"]`, `["public"]`, "false"})
	_ = w.Write([]string{"component", "Component", "physical", "An asset part.", "", "", "true"})
	w.Flush()
	d, err = Parse([]byte(b.String()), "csv")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Entries) != 1 || len(d.Entries[0].Senses) != 2 || d.Entries[0].Aliases[0].Sense != "web" {
		t.Fatal("CSV lost grouping or alias scope")
	}
	encoded, _ := json.Marshal(d)
	if _, err := Parse(encoded, "json"); err != nil {
		t.Fatal(err)
	}
	if Normalize(" CAFÉ ") != Normalize("cafe\u0301") {
		t.Fatal("normalization must combine Unicode folding and NFC")
	}
}
