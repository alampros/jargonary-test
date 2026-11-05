package scraper

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"jargonary/internal/config"
	"jargonary/internal/store"
)

func TestLastSuccessRetentionAndExplicitRemoval(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { st.Close() }()
	var mu sync.Mutex
	code := 200
	body := `{"version":"1","name":"Example","entries":[{"id":"old","term":"Old","senses":[{"id":"one","definition":"Still useful.","deprecated":true}]}]}`
	conditional := false
	token := strings.Repeat("token", 10)
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "application/json, application/yaml, text/yaml" {
			t.Errorf("unexpected source Accept header: %q", got)
		}
		mu.Lock()
		defer mu.Unlock()
		if r.Header.Get("Authorization") != "Bearer "+token {
			t.Error("missing bearer credential")
		}
		conditional = r.Header.Get("If-None-Match") == `"v1"`
		w.Header().Set("ETag", `"v1"`)
		w.WriteHeader(code)
		_, _ = w.Write([]byte(body))
	}))
	defer ts.Close()
	c := config.Config{Timeout: config.Duration(time.Second), MaxBytes: 4096}
	s := New(c, st)
	s.Client = ts.Client()
	source := config.Source{ID: "example", URL: ts.URL, Format: "json", Token: token}
	status := s.Once(ctx, source, store.Status{})
	if status.Failures != 0 || status.Entries != 1 {
		t.Fatalf("first scrape failed: %+v", status)
	}
	set := func(statusCode int, value string) { mu.Lock(); code = statusCode; body = value; mu.Unlock() }
	for _, statusCode := range []int{304, 401, 404, 410, 500} {
		set(statusCode, "do not leak "+token)
		status = s.Once(ctx, source, status)
		mu.Lock()
		usedConditional := conditional
		mu.Unlock()
		if !usedConditional {
			t.Fatal("missing cache validator")
		}
		b, _ := json.Marshal(status)
		if strings.Contains(string(b), token) {
			t.Fatal("scrape status leaked response payload")
		}
		hits, err := st.Search(ctx, "Old", "example", 20, 0)
		if err != nil || len(hits) != 1 {
			t.Fatalf("HTTP %d lost accepted content", statusCode)
		}
	}
	set(200, `{"version":"1","name":"Example","entries":null}`)
	status = s.Once(ctx, source, status)
	if status.Failures == 0 {
		t.Fatal("accepted invalid replacement")
	}
	// A credential/target change must not validate a representation cached for the old resource.
	mu.Lock()
	code = 304
	mu.Unlock()
	rotated := source
	rotated.Format = "yaml"
	status = s.Once(ctx, rotated, status)
	mu.Lock()
	reusedValidator := conditional
	mu.Unlock()
	if status.Failures == 0 || reusedValidator {
		t.Fatal("reused a validator across resource configurations")
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st, err = store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	s.Store = st
	hits, err := st.Search(ctx, "Old", "example", 20, 0)
	if err != nil || len(hits) != 1 {
		t.Fatal("restart lost last successful snapshot")
	}
	set(200, `{"version":"1","name":"Example","entries":[]}`)
	status = s.Once(ctx, source, status)
	if status.Failures != 0 {
		t.Fatal("empty snapshot failed")
	}
	hits, err = st.Search(ctx, "Old", "example", 20, 0)
	if err != nil || len(hits) != 0 {
		t.Fatal("successful withdrawal not applied")
	}
}

func TestRedirectsAndOversizedBodies(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("followed redirect") }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) }))
	defer redirect.Close()
	s := New(config.Config{Timeout: config.Duration(time.Second), MaxBytes: 32}, st)
	if _, _, _, _, err := s.fetch(ctx, config.Source{URL: redirect.URL}, nil); err == nil {
		t.Fatal("accepted redirect")
	}
	big := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(strings.Repeat("x", 100))) }))
	defer big.Close()
	if _, _, _, _, err := s.fetch(ctx, config.Source{URL: big.URL}, nil); err == nil {
		t.Fatal("accepted oversized source")
	}
}
