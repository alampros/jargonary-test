package store

import (
	"context"
	"path/filepath"
	"testing"

	v "jargonary/internal/vocabulary"
)

func TestSnapshotSearchAndRemoval(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	d := v.Dictionary{Version: "1", Name: "Engineering", Entries: []v.Entry{
		{ID: "intent", Term: "Payment Intent", Aliases: []v.Alias{{Text: "PI", Sense: "payment"}}, Senses: []v.Sense{{ID: "payment", Definition: "Tracks an attempt to collect money."}, {ID: "other", Definition: "Another meaning."}}},
		{ID: "pi", Term: "PI", Senses: []v.Sense{{ID: "math", Definition: "A ratio."}}},
		{ID: "cpp", Term: "C++", Senses: []v.Sense{{ID: "language", Definition: "A programming language.", Deprecated: true}}},
	}}
	if _, err := s.Replace(ctx, "engineering", d, "tag", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Replace(ctx, "project", d, "", ""); err != nil {
		t.Fatal(err)
	}
	hits, err := s.Search(ctx, "PI", "", 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 4 || hits[0].Entry != "pi" || hits[1].Entry != "pi" || hits[2].Sense != "payment" || hits[3].Sense != "payment" {
		t.Fatalf("exact and scoped alias ranking: %+v", hits)
	}
	for _, query := range []string{"", "collect"} {
		hits, err := s.Search(ctx, query, "", 20, 0)
		if err != nil {
			t.Fatal(err)
		}
		found := map[string]bool{}
		for _, hit := range hits {
			found[hit.Source] = true
		}
		if !found["engineering"] || !found["project"] {
			t.Fatalf("all-source search %q omitted an enabled source: %+v", query, hits)
		}
	}
	for query, want := range map[string]int{"C++": 1, "collect": 1, "paym": 2, "%": 0, `" OR *`: 0} {
		hits, err := s.Search(ctx, query, "engineering", 20, 0)
		if err != nil {
			t.Fatalf("%q: %v", query, err)
		}
		if len(hits) != want {
			t.Fatalf("%q got %d want %d", query, len(hits), want)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	hits, err = s.Search(ctx, "C++", "engineering", 20, 0)
	if err != nil || len(hits) != 1 || !hits[0].Deprecated {
		t.Fatalf("restart lost data: %+v %v", hits, err)
	}
	bad := d
	bad.Entries = nil
	if _, err := s.Replace(ctx, "engineering", bad, "", ""); err == nil {
		t.Fatal("accepted invalid replacement")
	}
	empty := v.Dictionary{Version: "1", Name: "Engineering", Entries: []v.Entry{}}
	if _, err := s.Replace(ctx, "engineering", empty, "", ""); err != nil {
		t.Fatal(err)
	}
	hits, err = s.Search(ctx, "", "engineering", 20, 0)
	if err != nil || len(hits) != 0 {
		t.Fatal("valid empty snapshot did not clear index")
	}
	if err := s.Keep(ctx, []string{"engineering"}); err != nil {
		t.Fatal(err)
	}
	sn, err := s.Snapshots(ctx)
	if err != nil || len(sn) != 1 {
		t.Fatalf("source retirement: %+v %v", sn, err)
	}
}
