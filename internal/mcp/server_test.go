package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"jargonary/internal/store"
	v "jargonary/internal/vocabulary"
)

func setup(t *testing.T) http.Handler {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	d := v.Dictionary{Version: "1", Name: "Example", Entries: []v.Entry{
		{
			ID: "api", Term: "API", Aliases: []v.Alias{{Text: "rest"}},
			Senses: []v.Sense{
				{ID: "web", Definition: "A web service interface.", Context: "Platform team", Relationships: []v.Relationship{{Type: "relatedTo", Target: v.Reference{Entry: "sdk"}}}},
			},
		},
		{
			ID: "sdk", Term: "SDK",
			Senses: []v.Sense{
				{ID: "client", Definition: "A client software development kit."},
			},
		},
	}}
	if _, err := st.Replace(context.Background(), "example", d, "", ""); err != nil {
		t.Fatal(err)
	}
	return New(st)
}

func call(t *testing.T, h http.Handler, method string, params any) map[string]any {
	t.Helper()
	body, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json, text/event-stream")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var payload []byte
	if strings.HasPrefix(w.Header().Get("Content-Type"), "text/event-stream") {
		data := w.Body.String()
		idx := strings.Index(data, "data: ")
		if idx < 0 {
			t.Fatalf("no SSE data in %q", data)
		}
		payload = []byte(strings.TrimSpace(data[idx+len("data: "):]))
	} else {
		payload = w.Body.Bytes()
	}
	var result map[string]any
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatalf("bad JSON-RPC response %q: %v", payload, err)
	}
	return result
}

func callTool(t *testing.T, h http.Handler, name string, args map[string]any) (string, bool) {
	t.Helper()
	resp := call(t, h, "tools/call", map[string]any{"name": name, "arguments": args})
	if err, ok := resp["error"].(map[string]any); ok {
		t.Fatalf("protocol error for %s: %v", name, err)
	}
	result, _ := resp["result"].(map[string]any)
	if result == nil {
		t.Fatalf("no result for %s: %v", name, resp)
	}
	isError, _ := result["isError"].(bool)
	content, _ := result["content"].([]any)
	if len(content) == 0 {
		t.Fatalf("no content for %s", name)
	}
	text, _ := content[0].(map[string]any)["text"].(string)
	return text, isError
}

func decodeJSON(t *testing.T, text string, out any) {
	t.Helper()
	if err := json.Unmarshal([]byte(text), out); err != nil {
		t.Fatalf("bad tool payload %q: %v", text, err)
	}
}

func TestToolList(t *testing.T) {
	h := setup(t)
	resp := call(t, h, "tools/list", map[string]any{})
	result, _ := resp["result"].(map[string]any)
	if result == nil {
		t.Fatalf("no result: %v", resp)
	}
	tools, _ := result["tools"].([]any)
	names := map[string]bool{}
	for _, tool := range tools {
		names[tool.(map[string]any)["name"].(string)] = true
	}
	for _, want := range []string{"list_glossaries", "search", "get_entry", "get_sense", "get_relationships", "browse_glossary", "list_meanings", "resolve_term"} {
		if !names[want] {
			t.Fatalf("missing tool %s; got %v", want, names)
		}
	}
	if len(tools) != 8 {
		t.Fatalf("expected 8 tools, got %d", len(tools))
	}
}

func TestSearch(t *testing.T) {
	h := setup(t)
	text, isError := callTool(t, h, "search", map[string]any{"q": "api"})
	if isError {
		t.Fatalf("unexpected error: %s", text)
	}
	var out struct {
		Items []map[string]any `json:"items"`
	}
	decodeJSON(t, text, &out)
	if len(out.Items) == 0 {
		t.Fatalf("no hits for api: %s", text)
	}
	hit := out.Items[0]
	if hit["source"] != "example" || hit["entry"] != "api" || hit["sense"] != "web" {
		t.Fatalf("unexpected hit: %v", hit)
	}
}

func TestGetEntry(t *testing.T) {
	h := setup(t)
	text, isError := callTool(t, h, "get_entry", map[string]any{"source": "example", "entry": "api"})
	if isError {
		t.Fatalf("unexpected error: %s", text)
	}
	var out struct {
		Term    string   `json:"term"`
		Aliases []string `json:"aliases"`
		Senses  []struct {
			ID  string `json:"id"`
			Def string `json:"definition"`
		} `json:"senses"`
		Outgoing []map[string]any `json:"outgoing"`
	}
	decodeJSON(t, text, &out)
	if out.Term != "API" || len(out.Senses) != 1 || out.Senses[0].ID != "web" || out.Senses[0].Def != "A web service interface." {
		t.Fatalf("unexpected entry: %+v", out)
	}
	if len(out.Outgoing) != 0 {
		t.Fatalf("get_entry must not include cross-references: %+v", out.Outgoing)
	}
}

func TestGetEntryNotFound(t *testing.T) {
	h := setup(t)
	text, isError := callTool(t, h, "get_entry", map[string]any{"source": "example", "entry": "missing"})
	if !isError {
		t.Fatalf("expected an error result, got %s", text)
	}
	if !strings.Contains(text, "not found") {
		t.Fatalf("expected not-found message, got %s", text)
	}
}

func TestGetSense(t *testing.T) {
	h := setup(t)
	text, isError := callTool(t, h, "get_sense", map[string]any{"source": "example", "entry": "api", "sense": "web"})
	if isError {
		t.Fatalf("unexpected error: %s", text)
	}
	var out struct {
		Term  string `json:"term"`
		Sense struct {
			ID  string `json:"id"`
			Def string `json:"definition"`
			Ctx string `json:"context"`
		} `json:"sense"`
	}
	decodeJSON(t, text, &out)
	if out.Sense.ID != "web" || out.Sense.Def != "A web service interface." || out.Sense.Ctx != "Platform team" {
		t.Fatalf("unexpected sense: %+v", out)
	}
}

func TestGetRelationships(t *testing.T) {
	h := setup(t)
	text, isError := callTool(t, h, "get_relationships", map[string]any{"source": "example", "entry": "api", "direction": "outgoing"})
	if isError {
		t.Fatalf("unexpected error: %s", text)
	}
	var out struct {
		Outgoing []map[string]any `json:"outgoing"`
		Incoming []map[string]any `json:"incoming"`
	}
	decodeJSON(t, text, &out)
	if len(out.Outgoing) != 1 || out.Outgoing[0]["entry"] != "sdk" || out.Outgoing[0]["term"] != "SDK" || out.Outgoing[0]["resolved"] != true {
		t.Fatalf("unexpected outgoing edges: %+v", out.Outgoing)
	}
	if len(out.Incoming) != 0 {
		t.Fatalf("outgoing-only call returned incoming edges: %+v", out.Incoming)
	}
}

func TestBrowseGlossary(t *testing.T) {
	h := setup(t)
	text, isError := callTool(t, h, "browse_glossary", map[string]any{"source": "example", "limit": 1})
	if isError {
		t.Fatalf("unexpected error: %s", text)
	}
	var out struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
		More  bool             `json:"more"`
	}
	decodeJSON(t, text, &out)
	if out.Total != 2 || len(out.Items) != 1 || !out.More {
		t.Fatalf("unexpected browse page: %+v", out)
	}
}

func TestListMeanings(t *testing.T) {
	h := setup(t)
	text, isError := callTool(t, h, "list_meanings", map[string]any{"term": "api"})
	if isError {
		t.Fatalf("unexpected error: %s", text)
	}
	var out []struct {
		Source string `json:"source"`
		Entry  string `json:"entry"`
		Term   string `json:"term"`
		Senses []struct {
			ID  string `json:"id"`
			Def string `json:"definition"`
		} `json:"senses"`
	}
	decodeJSON(t, text, &out)
	if len(out) != 1 || out[0].Entry != "api" || len(out[0].Senses) != 1 || out[0].Senses[0].ID != "web" {
		t.Fatalf("unexpected meanings: %+v", out)
	}
}

func TestResolveTerm(t *testing.T) {
	h := setup(t)
	text, isError := callTool(t, h, "resolve_term", map[string]any{"source": "example", "entry": "api"})
	if isError {
		t.Fatalf("unexpected error: %s", text)
	}
	var out struct {
		Term     string           `json:"term"`
		Outgoing []map[string]any `json:"outgoing"`
		Incoming []map[string]any `json:"incoming"`
	}
	decodeJSON(t, text, &out)
	if out.Term != "API" || len(out.Outgoing) != 1 || out.Outgoing[0]["term"] != "SDK" {
		t.Fatalf("unexpected resolution: %+v", out)
	}
	if len(out.Incoming) != 0 {
		t.Fatalf("api has no incoming edges: %+v", out.Incoming)
	}
	// The reverse direction: sdk's incoming edge points back at api, resolved.
	text, isError = callTool(t, h, "resolve_term", map[string]any{"source": "example", "entry": "sdk"})
	if isError {
		t.Fatalf("unexpected error: %s", text)
	}
	decodeJSON(t, text, &out)
	if len(out.Incoming) != 1 || out.Incoming[0]["entry"] != "api" || out.Incoming[0]["term"] != "API" || out.Incoming[0]["resolved"] != true {
		t.Fatalf("unexpected incoming edge: %+v", out.Incoming)
	}
}
