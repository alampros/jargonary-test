package publishing

import (
	"encoding/json"
	"strings"
	"testing"

	"jargonary/internal/config"
	"jargonary/internal/store"
	v "jargonary/internal/vocabulary"
)

func TestFilteredExportIdentityAndDeprecation(t *testing.T) {
	entries := []v.Entry{
		{ID: "component", Term: "Component", Pronunciation: "kuhm-POH-nuhnt", Aliases: []v.Alias{{Text: "Widget", Sense: "public"}, {Text: "SECRET-ALIAS", Sense: "private"}}, Senses: []v.Sense{
			{ID: "public", PartOfSpeech: "noun", Definition: "A public meaning.", Tags: []string{"public"}, Relationships: []v.Relationship{{Type: "deprecatedFor", Target: v.Reference{Entry: "secret"}}, {Type: "relatedTo", Target: v.Reference{Source: "sales", Entry: "component"}}}},
			{ID: "private", Definition: "SECRET-DEFINITION"},
		}},
		{ID: "secret", Term: "SECRET-TARGET", Senses: []v.Sense{{ID: "one", Definition: "Secret."}}},
	}
	snapshots := []store.Snapshot{
		{Source: "engineering", Dictionary: v.Dictionary{Version: "1", Name: "Engineering", Entries: entries}},
		{Source: "sales", Dictionary: v.Dictionary{Version: "1", Name: "Sales", Entries: []v.Entry{{ID: "component", Term: "Sales component", Senses: []v.Sense{{ID: "one", Definition: "Sales meaning.", Tags: []string{"public"}}}}}}},
	}
	export := config.Export{Name: "Public", Include: config.Include{SenseTags: config.Tags{Any: []string{"public"}}}}
	d, err := Build(export, snapshots)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Entries) != 2 || d.Entries[0].ID == d.Entries[1].ID {
		t.Fatal("colliding entry IDs were merged")
	}
	e := d.Entries[0]
	if !e.Senses[0].Deprecated || len(e.Senses) != 1 || len(e.Aliases) != 1 || e.Origin.Source != "engineering" || e.Pronunciation != "kuhm-POH-nuhnt" || e.Senses[0].PartOfSpeech != "noun" {
		t.Fatalf("lost status, filtering, or origin: %+v", e)
	}
	if len(e.Senses[0].Relationships) != 1 || e.Senses[0].Relationships[0].Target.Entry != d.Entries[1].ID || e.Senses[0].Relationships[0].Target.Source != "" {
		t.Fatal("cross-source reference was not rewritten")
	}
	b, _ := json.Marshal(d)
	if strings.Contains(string(b), "SECRET") {
		t.Fatalf("private content leaked: %s", b)
	}
	if _, err := v.Parse(b, "json"); err != nil {
		t.Fatal("consumer cannot parse export", err)
	}
	if len(snapshots[0].Dictionary.Entries[0].Senses) != 2 || snapshots[0].Dictionary.Entries[0].Senses[0].Deprecated {
		t.Fatal("export mutated source snapshot")
	}
	if _, err := Build(export, []store.Snapshot{{Source: "consumer", Dictionary: d}}); err == nil {
		t.Fatal("re-export chain was enabled")
	}
	export.Include.Sources = []string{"sales"}
	one, err := Build(export, snapshots)
	if err != nil || len(one.Entries) != 1 {
		t.Fatal("source allowlist not enforced")
	}
	if ExportID(strings.Repeat("a", 200), strings.Repeat("b", 200)) == ExportID("a", "b") {
		t.Fatal("identity collision")
	}
}

func TestExportsExcludeProject(t *testing.T) {
	dictionary := v.Dictionary{Version: "1", Name: "Glossary", Entries: []v.Entry{{ID: "term", Term: "Term", Relationships: []v.Relationship{{Type: "relatedTo", Target: v.Reference{Source: "project", Entry: "term"}}}, Senses: []v.Sense{{ID: "one", Definition: "A definition.", Tags: []string{"public"}}}}}}
	snapshots := []store.Snapshot{{Source: "project", Dictionary: dictionary}, {Source: "company", Dictionary: dictionary}}
	for _, tc := range []struct {
		name    string
		include config.Include
		want    int
	}{
		{name: "all", include: config.Include{All: true}, want: 1},
		{name: "tag", include: config.Include{SenseTags: config.Tags{Any: []string{"public"}}}, want: 1},
		{name: "company", include: config.Include{All: true, Sources: []string{"company"}}, want: 1},
		{name: "explicit project", include: config.Include{All: true, Sources: []string{"project"}}, want: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Build(config.Export{Name: "Export", Include: tc.include}, snapshots)
			if err != nil || len(d.Entries) != tc.want {
				t.Fatalf("unexpected export: %+v %v", d, err)
			}
			for _, entry := range d.Entries {
				if entry.Origin.Source != "company" || len(entry.Relationships) != 0 {
					t.Fatalf("project content or reference leaked: %+v", entry)
				}
			}
		})
	}
}
