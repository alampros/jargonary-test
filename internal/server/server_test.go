package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"jargonary/internal/config"
	"jargonary/internal/publishing"
	"jargonary/internal/store"
	v "jargonary/internal/vocabulary"
)

func setup(t *testing.T) *Server {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	d := v.Dictionary{Version: "1", Name: "Example", Entries: []v.Entry{
		{ID: "old", Term: `Old <script>alert(1)</script>`, Senses: []v.Sense{{ID: "one", Definition: `Old "name" <script>danger</script>`, Tags: []string{"public"}, Relationships: []v.Relationship{{Type: "deprecatedFor", Target: v.Reference{Entry: "new", Sense: "one"}}}}}},
		{ID: "new", Term: "New", Senses: []v.Sense{{ID: "one", Definition: "The current term.", Tags: []string{"public"}}}},
	}}
	if _, err := st.Replace(context.Background(), "example", d, "", ""); err != nil {
		t.Fatal(err)
	}
	c := config.Config{PublicURL: "https://glossary.example.test/base", Admin: config.Admin{Enabled: true}, Exports: []config.Export{{ID: "public", Name: "Public", Token: strings.Repeat("a", 32), Include: config.Include{SenseTags: config.Tags{Any: []string{"public"}}}}, {ID: "partner", Name: "Partner", Token: strings.Repeat("b", 32), Include: config.Include{All: true}}}}
	return &Server{Config: c, Store: st}
}

func request(h http.Handler, path, token, etag string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	if etag != "" {
		r.Header.Set("If-None-Match", etag)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestInterfacesAndRelationshipTraversal(t *testing.T) {
	s := setup(t)
	pub := s.Public()
	admin := s.Admin()
	for _, path := range []string{"/api/v1/config", "/api/v1/scrapes", "/api/v1/diagnostics"} {
		if w := request(pub, path, "", ""); w.Code != 404 {
			t.Fatalf("operator route on public listener: %s => %d", path, w.Code)
		}
		if w := request(admin, path, "", ""); w.Code != 200 {
			t.Fatalf("operator route unavailable: %s => %d %s", path, w.Code, w.Body)
		}
	}
	if w := request(admin, "/api/v1/search", "", ""); w.Code != 404 {
		t.Fatal("public routes on admin router")
	}
	s.Config.Admin.Enabled = false
	if w := request(s.Admin(), "/api/v1/config", "", ""); w.Code != 404 {
		t.Fatal("disabled admin still exposed")
	}
	w := request(pub, "/api/v1/sources/example/entries/old/senses/one", "", "")
	if w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	var sense SenseView
	if err := json.Unmarshal(w.Body.Bytes(), &sense); err != nil {
		t.Fatal(err)
	}
	if !sense.Deprecated || len(sense.Links["deprecatedFor"]) != 1 {
		t.Fatalf("missing replacement: %+v", sense)
	}
	if sense.Links["deprecatedFor"][0].Href != "https://glossary.example.test/base/api/v1/sources/example/entries/new/senses/one" {
		t.Fatal("incorrect replacement URL")
	}
	w = request(pub, "/api/v1/sources/example/entries/new/senses/one", "", "")
	if err := json.Unmarshal(w.Body.Bytes(), &sense); err != nil {
		t.Fatal(err)
	}
	if sense.Deprecated || len(sense.Links["supersedes"]) != 1 {
		t.Fatal("reverse relationship deprecates replacement or is missing")
	}
	for _, path := range []string{"/openapi.json", "/api/v1/search?q=New", "/api/v1/sources", "/healthz", "/readyz", "/.well-known/glossary"} {
		if w := request(pub, path, "", ""); w.Code != 200 {
			t.Fatalf("%s => %d %s", path, w.Code, w.Body)
		}
	}
	w = request(pub, "/definitions/example/old/one", "", "")
	if w.Code != 200 || strings.Contains(w.Body.String(), "<script>") || !strings.Contains(w.Body.String(), `property="og:description"`) {
		t.Fatalf("unsafe or missing initial metadata: %s", w.Body)
	}
}

func TestExportAuthenticationPrecedesConditionalResponse(t *testing.T) {
	s := setup(t)
	pub := s.Public()
	path := "/.well-known/glossary/public"
	w := request(pub, path, strings.Repeat("a", 32), "")
	if w.Code != 200 {
		t.Fatalf("export: %d %s", w.Code, w.Body)
	}
	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Fatal("missing export validator")
	}
	var d v.Dictionary
	if err := json.Unmarshal(w.Body.Bytes(), &d); err != nil {
		t.Fatal(err)
	}
	if d.Entries[0].ID != publishing.ExportID("example", "old") {
		t.Fatal("export identity mismatch")
	}
	if w := request(pub, path, "", etag); w.Code != 401 {
		t.Fatal("conditional response bypassed authentication")
	}
	if w := request(pub, "/.well-known/glossary/partner", strings.Repeat("a", 32), ""); w.Code != 401 {
		t.Fatal("credential granted access to another export")
	}
	if w := request(pub, path, strings.Repeat("a", 32), etag); w.Code != 304 {
		t.Fatalf("expected authenticated 304 got %d", w.Code)
	}
	ctx := context.Background()
	snapshots, _ := s.Store.Snapshots(ctx)
	d = snapshots[0].Dictionary
	d.Entries = []v.Entry{}
	if _, err := s.Store.Replace(ctx, "example", d, "", ""); err != nil {
		t.Fatal(err)
	}
	w = request(pub, path, strings.Repeat("a", 32), etag)
	if w.Code != 200 || w.Header().Get("ETag") == etag {
		t.Fatal("old validator concealed withdrawal")
	}
	if w := request(pub, "/definitions/example/old", "", ""); w.Code != 404 {
		t.Fatal("deleted definition still served")
	}
}
