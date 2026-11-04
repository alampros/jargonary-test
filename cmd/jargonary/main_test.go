package main

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"jargonary/internal/config"
	"jargonary/internal/server"
	"jargonary/internal/store"
	"jargonary/internal/vocabulary"
)

func TestProjectGlossaryStartupTransitions(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "glossary.db")
	company := vocabulary.Dictionary{Version: "1", Name: "Company", Entries: []vocabulary.Entry{{ID: "company-term", Term: "Company term", Senses: []vocabulary.Sense{{ID: "one", Definition: "Company definition."}}}}}
	for _, tc := range []struct {
		name        string
		hide        bool
		sources     []config.Source
		wantProject bool
	}{
		{"default", false, []config.Source{{ID: "company"}}, true},
		{"hidden on restart", true, []config.Source{{ID: "company"}}, false},
		{"reenabled on restart", false, []config.Source{{ID: "company"}}, true},
		{"empty fallback", true, nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st, err := store.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer st.Close()
			if len(tc.sources) > 0 {
				if _, err := st.Replace(ctx, "company", company, "", ""); err != nil {
					t.Fatal(err)
				}
			}
			c := config.Config{HideJargonaryGlossary: tc.hide, Sources: tc.sources}
			if err := initializeSources(ctx, c, st); err != nil {
				t.Fatal(err)
			}
			for _, query := range []string{"", "entry"} {
				hits, err := st.Search(ctx, query, "project", 100, 0)
				if err != nil || (len(hits) > 0) != tc.wantProject {
					t.Fatalf("cached project query %q after restart: %d hits, %v", query, len(hits), err)
				}
			}
			snapshots, err := st.Snapshots(ctx)
			if err != nil {
				t.Fatal(err)
			}
			want := len(tc.sources)
			if tc.wantProject {
				want++
			}
			if len(snapshots) != want {
				t.Fatalf("source snapshots = %d, want %d", len(snapshots), want)
			}
			svc := &server.Server{Config: c, Store: st}
			w := httptest.NewRecorder()
			svc.Public().ServeHTTP(w, httptest.NewRequest("GET", "/.well-known/glossary", nil))
			wantStatus := 404
			if tc.wantProject {
				wantStatus = 200
			}
			if w.Code != wantStatus {
				t.Fatalf("project publication = %d, want %d", w.Code, wantStatus)
			}
		})
	}
}
