package server

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"jargonary/internal/store"
	"jargonary/internal/vocabulary"
)

func withPageAssets(t *testing.T) *Server {
	t.Helper()
	s := setup(t)
	s.Config.SiteName = "Acme"
	s.Assets = fstest.MapFS{
		"index.html":    {Data: []byte(`<!doctype html><html lang="en"><head><!--page-head--><script type="module" src="./assets/app.js"></script></head><body><div id="root" data-bootstrap="<!--page-data-->"><!--page-content--></div></body></html>`)},
		"assets/app.js": {Data: []byte(`console.log("test bundle");`)},
	}
	return s
}

func TestDevelopmentUIKeepsServerPagesAndProductionIsolation(t *testing.T) {
	s := withPageAssets(t)
	s.DevUI = fstest.MapFS{"index.html": {Data: []byte(`<!doctype html><html><head><!--page-head--></head><body><div id="root" data-bootstrap="<!--page-data-->"><!--page-content--></div><script type="module" src="/src/main.tsx"></script></body></html>`)}}
	for _, handler := range []http.Handler{s.Public(), s.Admin()} {
		w := request(handler, "/", "", "")
		body := w.Body.String()
		if w.Code != 200 || !strings.Contains(body, `src="http://127.0.0.1:5173/src/main.tsx"`) || !strings.Contains(body, "/@vite/client") || !strings.Contains(body, "__vite_plugin_react_preamble_installed__") {
			t.Fatalf("missing development modules: %d %s", w.Code, body)
		}
		if bootstrap(t, body).SiteName != "Acme" || strings.Contains(body, "<!--page-") || strings.Contains(body, "./assets/app.js") {
			t.Fatal("development HTML lost bootstrap data or used production assets")
		}
		if !strings.Contains(w.Header().Get("Content-Security-Policy"), "ws://127.0.0.1:5173") || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("development policy or caching incorrect")
		}
	}
	w := request(s.Public(), "/definitions/example/new", "", "")
	if w.Code != 200 || bootstrap(t, w.Body.String()).Entry.ID != "new" || !strings.Contains(w.Body.String(), `property="og:description"`) || !strings.Contains(w.Body.String(), "<h1>") {
		t.Fatal("development definition lost server-rendered content or social metadata")
	}
	if w := request(s.Public(), "/definitions/example/missing", "", ""); w.Code != 404 {
		t.Fatal("development mode changed missing-definition status")
	}
	s.DevUI = nil
	w = request(s.Public(), "/", "", "")
	if strings.Contains(w.Body.String(), "5173") || strings.Contains(w.Header().Get("Content-Security-Policy"), "unsafe-inline") || !strings.Contains(w.Body.String(), "/base/ui/assets/app.js") {
		t.Fatal("development scripts or permissions leaked into production")
	}
}

func TestDevelopmentUIMissingTemplateIsExplicit(t *testing.T) {
	s := withPageAssets(t)
	s.DevUI = fstest.MapFS{}
	w := request(s.Public(), "/", "", "")
	if w.Code != 503 || !strings.Contains(w.Body.String(), "Development template unavailable") {
		t.Fatal("missing development template silently used an embedded build")
	}
}

func TestPublicFreshnessOmitsOperatorDetails(t *testing.T) {
	s := withPageAssets(t)
	last := time.Now().UTC().Add(-time.Hour)
	if err := s.Store.SaveStatus(context.Background(), store.Status{Source: "example", Failures: 2, LastSuccess: last, Recent: []store.Outcome{{Detail: "PRIVATE-DIAGNOSTIC"}}}); err != nil {
		t.Fatal(err)
	}
	w := request(s.Public(), "/?q=New", "", "")
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	data := bootstrap(t, w.Body.String())
	if len(data.Sources) != 1 || !data.Sources[0].Stale || data.Sources[0].LastChecked == nil {
		t.Fatal("last-good source freshness missing")
	}
	if strings.Contains(w.Body.String(), "PRIVATE-DIAGNOSTIC") {
		t.Fatal("public HTML leaked operator history")
	}
}

func bootstrap(t *testing.T, body string) pageData {
	t.Helper()
	m := regexp.MustCompile(`data-bootstrap="([^"]+)"`).FindStringSubmatch(body)
	if len(m) != 2 {
		t.Fatal("missing bootstrap attribute")
	}
	var data pageData
	if err := json.Unmarshal([]byte(html.UnescapeString(m[1])), &data); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestHomeAndDefinitionsUseRealData(t *testing.T) {
	s := withPageAssets(t)
	h := s.Public()
	w := request(h, "/", "", "")
	if w.Code != 200 {
		t.Fatalf("home: %d %s", w.Code, w.Body)
	}
	data := bootstrap(t, w.Body.String())
	if data.Mode != "public" || len(data.Entries) != 2 || data.SiteName != "Acme" {
		t.Fatalf("unexpected bootstrap: %+v", data)
	}
	if strings.Contains(w.Body.String(), "/api/v1/config") || !strings.Contains(w.Body.String(), `src="/base/ui/assets/app.js"`) {
		t.Fatal("operator data leaked or subpath assets broken")
	}
	w = request(h, "/?q=New", "", "")
	data = bootstrap(t, w.Body.String())
	if data.Entry == nil || data.Entry.ID != "new" || !strings.Contains(w.Body.String(), `property="og:description"`) {
		t.Fatal("single exact result or metadata missing")
	}
	if len(data.Relationships["sense:one"]) != 1 || data.Relationships["sense:one"][0].Type != "supersedes" || !strings.HasPrefix(data.Relationships["sense:one"][0].Href, "/base/definitions/") {
		t.Fatal("relationship lacks server-provided UI navigation")
	}
	for _, path := range []string{"/?letter=N", "/?letter=N&source=example"} {
		w = request(h, path, "", "")
		data = bootstrap(t, w.Body.String())
		if w.Code != 200 || data.Entry != nil || len(data.TermEntries) != 0 || len(data.Entries) != 1 || data.Entries[0].ID != "new" || data.Letter != "N" {
			t.Fatal("single-term letter browse must remain a list")
		}
	}
	w = request(h, "/?letter=Z", "", "")
	data = bootstrap(t, w.Body.String())
	if len(data.Entries) != 0 || data.Entry != nil {
		t.Fatal("empty letter returned a definition")
	}
	w = request(h, "/definitions/example/old/one", "", "")
	if w.Code != 200 || strings.Contains(w.Body.String(), "<script>danger</script>") {
		t.Fatal("definition markup is unsafe")
	}
	w = request(h, "/definitions/example/missing", "", "")
	if w.Code != 404 || !bootstrap(t, w.Body.String()).NotFound {
		t.Fatal("missing definition must render a 404 view")
	}
	for _, path := range []string{"/?letter=ABC", "/?page=-1", "/?q=" + strings.Repeat("a", 257)} {
		if w := request(h, path, "", ""); w.Code != 400 {
			t.Fatalf("invalid request %s => %d", path, w.Code)
		}
	}
}

func TestAllSourcesIncludesProjectGlossary(t *testing.T) {
	s := withPageAssets(t)
	d := vocabulary.Dictionary{Version: "1", Name: "Project glossary", Entries: []vocabulary.Entry{{ID: "project-term", Term: "Project term", Pronunciation: "PROJ-ekt turm", Senses: []vocabulary.Sense{{ID: "one", Definition: "A bundled project definition.", PartOfSpeech: vocabulary.PartOfSpeechNoun}}}}}
	if _, err := s.Store.Replace(context.Background(), "project", d, "", ""); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/", "/?letter=P", "/?q=bundled", "/?source=project"} {
		w := request(s.Public(), path, "", "")
		data := bootstrap(t, w.Body.String())
		found := false
		for _, entry := range data.Entries {
			found = found || entry.Source == "project"
			if entry.Source == "project" && (entry.Pronunciation != "PROJ-ekt turm" || len(entry.PartsOfSpeech) != 1 || entry.PartsOfSpeech[0] != "noun") {
				t.Fatalf("page lost lexical metadata: %+v", entry)
			}
		}
		if w.Code != 200 || !found {
			t.Fatalf("project glossary omitted from %s: %+v", path, data.Entries)
		}
	}
	w := request(s.Public(), "/api/v1/search?q=bundled", "", "")
	var out SearchBody
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || w.Code != 200 || len(out.Items) != 1 || out.Items[0].Source != "project" || out.Items[0].Pronunciation != "PROJ-ekt turm" || out.Items[0].PartOfSpeech != "noun" {
		t.Fatalf("API all-source search omitted project: %s, %v", w.Body.String(), err)
	}
	w = request(s.Public(), "/?source=example", "", "")
	for _, entry := range bootstrap(t, w.Body.String()).Entries {
		if entry.Source == "project" {
			t.Fatal("explicit company source included project terms")
		}
	}
}

func TestOperatorPageAndAssetBoundaries(t *testing.T) {
	s := withPageAssets(t)
	pub, admin := s.Public(), s.Admin()
	w := request(admin, "/", "", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Operator console") || bootstrap(t, w.Body.String()).Mode != "admin" {
		t.Fatal("operator root unavailable")
	}
	if bootstrap(t, request(pub, "/", "", "").Body.String()).Mode == "admin" {
		t.Fatal("public root returned operator payload")
	}
	w = request(pub, "/ui/assets/app.js", "", "")
	if w.Code != 200 || !strings.Contains(w.Header().Get("Cache-Control"), "immutable") {
		t.Fatal("hashed assets not served")
	}
	for _, path := range []string{"/ui/index.html", "/ui/.keep", "/ui/assets/missing.js"} {
		if w := request(pub, path, "", ""); w.Code != 404 {
			t.Fatalf("unexpected static file: %s => %d", path, w.Code)
		}
	}
	s.Config.Admin.Enabled = false
	if w := request(s.Admin(), "/", "", ""); w.Code != 404 {
		t.Fatal("disabled operator UI remained reachable")
	}
	if w := request(admin, "/ui/assets/app.js", "", ""); w.Code != 200 {
		t.Fatal("operator assets unavailable")
	}
}

func TestLexicalMetadataDocumentedInOpenAPI(t *testing.T) {
	s := withPageAssets(t)
	w := request(s.Public(), "/openapi.json", "", "")
	var spec struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]json.RawMessage `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &spec); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"pronunciation", "partOfSpeech"} {
		if _, ok := spec.Components.Schemas["SearchHit"].Properties[field]; !ok {
			t.Fatalf("search metadata %s is absent from OpenAPI", field)
		}
	}
}

func TestTermGroupsStayTogetherAcrossPages(t *testing.T) {
	s := withPageAssets(t)
	ctx := context.Background()
	d := vocabulary.Dictionary{Version: "1", Name: "Primary"}
	for i := 0; i < 21; i++ {
		d.Entries = append(d.Entries, vocabulary.Entry{
			ID: fmt.Sprintf("term-%02d", i), Term: fmt.Sprintf("Term %02d", i),
			Senses: []vocabulary.Sense{{ID: "one", Definition: "Primary meaning.", Context: "Primary context"}},
		})
	}
	if _, err := s.Store.Replace(ctx, "example", d, "", ""); err != nil {
		t.Fatal(err)
	}
	d.Name = "Secondary"
	d.Entries = []vocabulary.Entry{d.Entries[0], d.Entries[19]}
	for i := range d.Entries {
		d.Entries[i].Senses = []vocabulary.Sense{{ID: "one", Definition: "Secondary meaning.", Context: "Secondary context"}}
	}
	if _, err := s.Store.Replace(ctx, "secondary", d, "", ""); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"", "&q=Term"} {
		t.Run(query, func(t *testing.T) {
			w := request(s.Public(), "/?page=0"+query, "", "")
			data := bootstrap(t, w.Body.String())
			if w.Code != 200 || len(data.Entries) != 22 || data.Next == "" {
				t.Fatalf("first page split term groups: %d entries, next %q", len(data.Entries), data.Next)
			}
			for _, term := range []string{"Term 00", "Term 19"} {
				seen := map[string]bool{}
				for _, entry := range data.Entries {
					if entry.Term == term {
						seen[entry.Source] = true
						if entry.TermKey != vocabulary.Normalize(term) || entry.Href == "" {
							t.Fatal("group lost its normalized term or source-qualified link")
						}
					}
				}
				if !seen["example"] || !seen["secondary"] || strings.Count(w.Body.String(), "<h2>"+term+"</h2>") != 1 {
					t.Fatalf("term %q not grouped with both sources", term)
				}
			}
			w = request(s.Public(), "/?page=1"+query, "", "")
			data = bootstrap(t, w.Body.String())
			if len(data.Entries) != 1 || data.Entries[0].Term != "Term 20" || data.Next != "" || data.Previous == "" {
				t.Fatalf("incorrect second page: %+v", data)
			}
		})
	}
	w := request(s.Public(), "/?source=secondary", "", "")
	if data := bootstrap(t, w.Body.String()); len(data.Entries) != 2 || data.Entries[0].Source != "secondary" || data.Entries[1].Source != "secondary" {
		t.Fatal("term grouping changed source filtering")
	}
}

func TestTermDefinitionIncludesSelectedSources(t *testing.T) {
	s := withPageAssets(t)
	ctx := context.Background()
	d := vocabulary.Dictionary{Version: "1", Name: "Other team", Entries: []vocabulary.Entry{
		{ID: "new", Term: "NEW", Aliases: []vocabulary.Alias{{Text: "Other alias", Sense: "one"}}, Senses: []vocabulary.Sense{
			{ID: "one", Definition: "Another team's definition.", Relationships: []vocabulary.Relationship{
				{Type: "relatedTo", Target: vocabulary.Reference{Source: "example", Entry: "old"}},
			}},
		}},
		{ID: "near-match", Term: "Newer", Senses: []vocabulary.Sense{{ID: "one", Definition: "A different term."}}},
		{ID: "punctuation", Term: "C++ / API", Senses: []vocabulary.Sense{{ID: "one", Definition: "Punctuation test."}}},
	}}
	if _, err := s.Store.Replace(ctx, "other", d, "", ""); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path    string
		sources string
	}{
		{"/definitions/New", "example,other"},
		{"/definitions/new?source=example", "example"},
		{"/definitions/NEW?source=other", "other"},
	} {
		w := request(s.Public(), tc.path, "", "")
		data := bootstrap(t, w.Body.String())
		var sources []string
		for _, entry := range data.TermEntries {
			sources = append(sources, entry.Source)
			if vocabulary.Normalize(entry.Term) != "new" || len(entry.Senses) != 1 {
				t.Fatal("term page included unrelated entries or lost senses")
			}
		}
		if w.Code != 200 || strings.Join(sources, ",") != tc.sources || data.EntryURL != data.TermURL {
			t.Fatalf("%s: code %d, sources %v, copy URL %q", tc.path, w.Code, sources, data.EntryURL)
		}
		if len(data.TermEntries) == 2 && len(data.Relationships["other/new/sense:one"]) == 0 {
			t.Fatal("combined page lost source-qualified relationships")
		}
		if !strings.Contains(w.Body.String(), `rel="canonical" href="https://glossary.example.test`+data.TermURL+`"`) {
			t.Fatal("combined canonical URL does not preserve selected source")
		}
	}
	for _, path := range []string{"/definitions/example/new", "/definitions/example/new/one"} {
		w := request(s.Public(), path, "", "")
		data := bootstrap(t, w.Body.String())
		if w.Code != 200 || len(data.TermEntries) != 0 || data.Entry.Source != "example" || data.TermURL != "/base/definitions/New" {
			t.Fatal("source-qualified deep link changed its identity")
		}
	}
	for _, path := range []string{"/definitions/missing", "/definitions/New?source=missing"} {
		if w := request(s.Public(), path, "", ""); w.Code != 404 {
			t.Fatalf("missing term/source should be 404: %s => %d", path, w.Code)
		}
	}
	w := request(s.Public(), "/definitions/"+url.PathEscape("C++ / API"), "", "")
	if w.Code != 200 || bootstrap(t, w.Body.String()).Entry.Term != "C++ / API" {
		t.Fatal("term URL lost punctuation")
	}
}

func TestPagesIncludeCachedLettersForEverySource(t *testing.T) {
	s := withPageAssets(t)
	d := vocabulary.Dictionary{Version: "1", Name: "Other", Entries: []vocabulary.Entry{{ID: "apple", Term: "Apple", Senses: []vocabulary.Sense{{ID: "one", Definition: "Fruit."}}}}}
	if _, err := s.Store.Replace(context.Background(), "other", d, "", ""); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/", "/?q=New", "/?letter=N", "/?source=example", "/definitions/New?source=example", "/definitions/example/new"} {
		w := request(s.Public(), path, "", "")
		data := bootstrap(t, w.Body.String())
		if w.Code != 200 || len(data.Sources) != 2 || strings.Join(data.Sources[0].Letters, ",") != "N,O" || strings.Join(data.Sources[1].Letters, ",") != "A" {
			t.Fatalf("%s: missing source letter sets: %+v", path, data.Sources)
		}
	}
}

func TestFaviconOnPublicAndOperatorListeners(t *testing.T) {
	s := withPageAssets(t)
	s.Config.PublicURL = "https://example.com/base"
	for name, handler := range map[string]http.Handler{"public": s.Public(), "admin": s.Admin()} {
		t.Run(name, func(t *testing.T) {
			w := request(handler, "/favicon.svg", "", "")
			if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "image/svg+xml" || !strings.Contains(w.Body.String(), "<svg") {
				t.Fatalf("favicon unavailable: %d %s", w.Code, w.Body.String())
			}
			page := request(handler, "/", "", "")
			href := "/favicon.svg"
			if name == "public" {
				href = "/base/favicon.svg"
			}
			if !strings.Contains(page.Body.String(), `rel="icon" type="image/svg+xml" href="`+href+`"`) {
				t.Fatalf("page missing favicon link %s", href)
			}
		})
	}
}

func TestEditorPageIsOnlyALocalWorkspaceShell(t *testing.T) {
	s := withPageAssets(t)
	// This route must not depend on cached glossary data or query the store.
	s.Store = nil
	w := request(s.Public(), "/editor", "", "")
	body := w.Body.String()
	if w.Code != http.StatusOK || bootstrap(t, body).Mode != "editor" || !strings.Contains(body, "/base/ui/assets/app.js") || !strings.Contains(body, "Glossary editor") {
		t.Fatalf("editor shell missing: %d %s", w.Code, body)
	}
	if strings.Contains(body, `"entries"`) || strings.Contains(body, `"config"`) {
		t.Fatal("editor bootstrap contains application data")
	}
	if w := request(s.Admin(), "/editor", "", ""); w.Code != http.StatusNotFound {
		t.Fatal("editor should be available on the public UI listener only")
	}
}
