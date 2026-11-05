package server

import (
	"context"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"

	"jargonary/internal/store"
	"jargonary/internal/vocabulary"
)

// The previous resolver remains a test oracle and benchmark baseline only.
func snapshotEntry(base string, snapshots []store.Snapshot, source, id string) (EntryView, bool) {
	g := buildGraph(base, snapshots)
	for _, sn := range snapshots {
		if sn.Source != source {
			continue
		}
		for _, e := range sn.Dictionary.Entries {
			if e.ID != id {
				continue
			}
			r := vocabulary.Reference{Source: source, Entry: id}
			n := g[r]
			view := EntryView{Source: source, ID: id, Origin: e.Origin, Term: e.Term, Pronunciation: e.Pronunciation, Aliases: e.Aliases, Deprecated: vocabulary.Deprecated(e.Deprecated, e.Relationships), DeprecationReason: e.DeprecationReason, Relationships: n.relationships, Links: n.links, Senses: []SenseView{}}
			for _, s := range e.Senses {
				r := vocabulary.Reference{Source: source, Entry: id, Sense: s.ID}
				n := g[r]
				sv := SenseView{Source: source, EntryID: id, Term: e.Term, Pronunciation: e.Pronunciation, Sense: s, Deprecated: view.Deprecated || vocabulary.Deprecated(s.Deprecated, s.Relationships), Relationships: n.relationships, Links: n.links}
				view.Senses = append(view.Senses, sv)
				view.Links["sense"] = append(view.Links["sense"], Hyperlink{Href: base + refPath(r), Title: s.Context})
			}
			return view, true
		}
	}
	return EntryView{}, false
}

func TestIndexedRelationshipsMatchSnapshotGraph(t *testing.T) {
	s := setup(t)
	ctx := context.Background()
	for _, source := range []string{"a", "b"} {
		d := vocabulary.Dictionary{Version: "1", Name: source, Exported: true}
		for _, id := range []string{"same", "same-1"} {
			e := vocabulary.Entry{ID: id, Term: source + id, Origin: &vocabulary.Reference{Source: "origin", Entry: id}, Pronunciation: "spoken", Aliases: []vocabulary.Alias{{Text: "global"}, {Text: "scoped", Sense: "one"}}, DeprecationReason: "entry reason", Senses: []vocabulary.Sense{{ID: "one", Definition: "first", PartOfSpeech: vocabulary.PartOfSpeechNoun, Context: "context", Examples: []string{"example"}, Tags: []string{"tag"}, Links: []vocabulary.Link{{Label: "external", URL: "https://example.test"}}}, {ID: "one-1", Definition: "second", Deprecated: true, DeprecationReason: "sense reason"}}}
			for _, originSense := range []string{"", "one", "one-1"} {
				var relationships []vocabulary.Relationship
				for _, targetSource := range []string{"", "a", "b", "absent"} {
					for _, targetEntry := range []string{"same", "same-1", "missing"} {
						for _, targetSense := range []string{"", "one", "one-1", "missing"} {
							for _, typ := range []string{"relatedTo", "abbreviationOf", "deprecatedFor"} {
								rel := vocabulary.Relationship{Type: typ, Target: vocabulary.Reference{Source: targetSource, Entry: targetEntry, Sense: targetSense}}
								relationships = append(relationships, rel, rel) // duplicate authored edges and explicit inverses
							}
						}
					}
				}
				if originSense == "" {
					e.Relationships = relationships
				} else if originSense == "one" {
					e.Senses[0].Relationships = relationships
				} else {
					e.Senses[1].Relationships = relationships
				}
			}
			d.Entries = append(d.Entries, e)
		}
		if _, err := s.Store.Replace(ctx, source, d, "", ""); err != nil {
			t.Fatal(err)
		}
	}
	snapshots, err := s.Store.Snapshots(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, sn := range snapshots {
		for _, e := range sn.Dictionary.Entries {
			want, found := snapshotEntry(s.Config.PublicURL, snapshots, sn.Source, e.ID)
			got, ok, err := s.findEntry(ctx, sn.Source, e.ID)
			if err != nil || ok != found || !reflect.DeepEqual(got, want) {
				t.Fatalf("entry differs from snapshot graph %s/%s: %v", sn.Source, e.ID, err)
			}
			for _, sense := range e.Senses {
				selected, ok, err := s.findEntry(ctx, sn.Source, e.ID, sense.ID)
				if err != nil || !ok || len(selected.Senses) != 1 {
					t.Fatalf("sense lookup: %+v %v", selected, err)
				}
				for _, sv := range want.Senses {
					if sv.Sense.ID == sense.ID && !reflect.DeepEqual(selected.Senses[0], sv) {
						t.Fatalf("sense differs %s/%s/%s", sn.Source, e.ID, sense.ID)
					}
				}
			}
		}
	}
	if _, ok, err := s.findEntry(ctx, "a", "same", "missing"); err != nil || ok {
		t.Fatalf("missing sense: %v %v", ok, err)
	}
}

func TestRelationshipTargetsArriveAndDisappear(t *testing.T) {
	s := setup(t)
	ctx := context.Background()
	referring := vocabulary.Dictionary{Version: "1", Name: "Referring", Entries: []vocabulary.Entry{{ID: "ref", Term: "Ref", Relationships: []vocabulary.Relationship{{Type: "relatedTo", Target: vocabulary.Reference{Source: "target", Entry: "ref"}}}, Senses: []vocabulary.Sense{{ID: "one", Definition: "Ref", Relationships: []vocabulary.Relationship{{Type: "abbreviationOf", Target: vocabulary.Reference{Source: "target", Entry: "ref", Sense: "one"}}}}}}}}
	target := vocabulary.Dictionary{Version: "1", Name: "Target", Entries: []vocabulary.Entry{{ID: "ref", Term: "Target", Senses: []vocabulary.Sense{{ID: "one", Definition: "Target"}}}}}
	if _, err := s.Store.Replace(ctx, "referring", referring, "", ""); err != nil {
		t.Fatal(err)
	}
	check := func(resolved bool) {
		t.Helper()
		got, ok, err := s.findEntry(ctx, "referring", "ref")
		if err != nil || !ok {
			t.Fatal(err)
		}
		if got.Relationships[0].Resolved != resolved || got.Senses[0].Relationships[0].Resolved != resolved || (len(got.Links["relatedTo"]) > 0) != resolved || (len(got.Senses[0].Links["abbreviationOf"]) > 0) != resolved {
			t.Fatalf("resolution: %+v", got)
		}
	}
	check(false)
	if _, err := s.Store.Replace(ctx, "target", target, "", ""); err != nil {
		t.Fatal(err)
	}
	check(true)
	got, _, err := s.findEntry(ctx, "target", "ref")
	if err != nil || len(got.Relationships) != 1 || !got.Relationships[0].Derived || len(got.Senses[0].Relationships) != 1 || got.Senses[0].Relationships[0].Type != "abbreviatedBy" {
		t.Fatalf("inverses: %+v %v", got, err)
	}
	// Removing just the target sense leaves the entry relationship resolved.
	target.Entries[0].Senses[0].ID = "two"
	if _, err := s.Store.Replace(ctx, "target", target, "", ""); err != nil {
		t.Fatal(err)
	}
	got, _, err = s.findEntry(ctx, "referring", "ref")
	if err != nil || !got.Relationships[0].Resolved || got.Senses[0].Relationships[0].Resolved {
		t.Fatalf("sense withdrawal: %+v %v", got, err)
	}
	target.Entries = []vocabulary.Entry{}
	if _, err := s.Store.Replace(ctx, "target", target, "", ""); err != nil {
		t.Fatal(err)
	}
	check(false)
	if _, err := s.Store.Replace(ctx, "target", vocabulary.Dictionary{}, "", ""); err == nil {
		t.Fatal("invalid source accepted")
	}
	check(false)
}

func BenchmarkEntryResolution(b *testing.B) {
	for _, count := range []int{1, 4000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			ctx := context.Background()
			st, err := store.Open(filepath.Join(b.TempDir(), "bench.db"))
			if err != nil {
				b.Fatal(err)
			}
			defer st.Close()
			d := vocabulary.Dictionary{Version: "1", Name: "Benchmark", Entries: []vocabulary.Entry{}}
			for i := 0; i < count; i++ {
				d.Entries = append(d.Entries, vocabulary.Entry{ID: fmt.Sprintf("entry-%04d", i), Term: fmt.Sprintf("Term %04d", i), Senses: []vocabulary.Sense{{ID: "one", Definition: "A definition for the allocation benchmark.", PartOfSpeech: vocabulary.PartOfSpeechNoun}}})
			}
			if _, err = st.Replace(ctx, "bench", d, "", ""); err != nil {
				b.Fatal(err)
			}
			server := &Server{Store: st}
			for _, id := range []string{"entry-0000", "missing"} {
				b.Run("snapshot/"+id, func(b *testing.B) {
					b.ReportAllocs()
					for b.Loop() {
						sn, err := st.Snapshots(ctx)
						if err != nil {
							b.Fatal(err)
						}
						_, ok := snapshotEntry("", sn, "bench", id)
						if ok != (id != "missing") {
							b.Fatal("unexpected lookup")
						}
					}
				})
				b.Run("indexed/"+id, func(b *testing.B) {
					b.ReportAllocs()
					for b.Loop() {
						_, ok, err := server.findEntry(ctx, "bench", id)
						if err != nil || ok != (id != "missing") {
							b.Fatalf("lookup: %v %v", ok, err)
						}
					}
				})
			}
			b.Run("indexed/sense-missing", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_, ok, err := server.findEntry(ctx, "bench", "entry-0000", "missing")
					if err != nil || ok {
						b.Fatalf("lookup: %v %v", ok, err)
					}
				}
			})
			b.Run("HTTP/404", func(b *testing.B) {
				h := server.Public()
				r := httptest.NewRequest("GET", "/api/v1/sources/bench/entries/missing", nil)
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					w := httptest.NewRecorder()
					h.ServeHTTP(w, r)
					if w.Code != 404 {
						b.Fatal(w.Code)
					}
				}
			})
		})
	}
}

func TestSensePagePreservesSiblingNavigation(t *testing.T) {
	s := withPageAssets(t)
	d := vocabulary.Dictionary{Version: "1", Name: "Senses", Entries: []vocabulary.Entry{{ID: "same", Term: "Same", Senses: []vocabulary.Sense{{ID: "one", Definition: "First", Context: "First context"}, {ID: "two", Definition: "Second", Context: "Second context"}}}}}
	if _, err := s.Store.Replace(context.Background(), "senses", d, "", ""); err != nil {
		t.Fatal(err)
	}
	w := request(s.Public(), "/definitions/senses/same/one", "", "")
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	entry := bootstrap(t, w.Body.String()).Entry
	if entry == nil || len(entry.Senses) != 1 || entry.Senses[0].Sense.ID != "one" || len(entry.Links["sense"]) != 2 {
		t.Fatalf("sense page changed navigation: %+v", entry)
	}
}
