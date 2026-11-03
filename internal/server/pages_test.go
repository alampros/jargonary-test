package server

import (
	"context"
	"encoding/json"
	"html"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"jargonary/internal/store"
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
	w = request(h, "/?letter=N", "", "")
	data = bootstrap(t, w.Body.String())
	if data.Entry == nil || data.Entry.ID != "new" {
		t.Fatal("letter browse not selecting matching entry")
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
