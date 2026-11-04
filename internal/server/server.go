// Package server keeps public and operator APIs on completely separate routers.
package server

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"jargonary/glossary"
	"jargonary/internal/config"
	"jargonary/internal/publishing"
	"jargonary/internal/store"
	"jargonary/internal/vocabulary"
)

type Server struct {
	Config config.Config
	Store  *store.Store
	Assets fs.FS // Optional HTML/asset filesystem override for tests.
	DevUI  fs.FS // Optional source template filesystem; enables local Vite integration.
}

func router() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; frame-ancestors 'none'; base-uri 'none'")
			w.Header().Set("Cache-Control", "no-store")
			next.ServeHTTP(w, r)
		})
	})
	return r
}

type SearchInput struct {
	Q      string `query:"q" maxLength:"256" doc:"Literal term, alias, prefix, or definition text"`
	Source string `query:"source" maxLength:"200" doc:"Source ID; omit to search all enabled sources"`
	Limit  int    `query:"limit" default:"20" minimum:"1" maximum:"100"`
	Offset int    `query:"offset" default:"0" minimum:"0" maximum:"100000"`
}
type SearchHit struct {
	LexicalFields
	store.Hit
	Links Links `json:"_links"`
}
type SearchBody struct {
	Items []SearchHit `json:"items"`
	Links Links       `json:"_links"`
}
type SearchOutput struct{ Body SearchBody }
type EntryInput struct {
	Source string `path:"source"`
	Entry  string `path:"entry"`
}
type SenseInput struct {
	Source string `path:"source"`
	Entry  string `path:"entry"`
	Sense  string `path:"sense"`
}
type EntryOutput struct{ Body EntryView }
type SenseOutput struct{ Body SenseView }
type SourceView struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	Entries             int       `json:"entries"`
	LastContentChange   time.Time `json:"lastContentChange"`
	LastSuccessfulCheck time.Time `json:"lastSuccessfulCheck"`
}
type SourcesOutput struct {
	Body struct {
		Items []SourceView `json:"items"`
	}
}

func (s *Server) Public() http.Handler {
	r := router()
	api := humachi.New(r, huma.DefaultConfig("Jargonary API", "0.1.0"))
	huma.Get(api, "/api/v1/search", func(ctx context.Context, in *SearchInput) (*SearchOutput, error) {
		hits, err := s.Store.Search(ctx, in.Q, in.Source, in.Limit+1, in.Offset)
		if err != nil {
			return nil, huma.Error500InternalServerError("search unavailable")
		}
		out := &SearchOutput{Body: SearchBody{Items: []SearchHit{}, Links: Links{}}}
		pageURL := func(offset int) string {
			q := url.Values{"q": {in.Q}, "source": {in.Source}, "limit": {strconv.Itoa(in.Limit)}, "offset": {strconv.Itoa(offset)}}
			return s.Config.PublicURL + "/api/v1/search?" + q.Encode()
		}
		out.Body.Links["self"] = []Hyperlink{{Href: pageURL(in.Offset)}}
		if len(hits) > in.Limit {
			out.Body.Links["next"] = []Hyperlink{{Href: pageURL(in.Offset + in.Limit)}}
			hits = hits[:in.Limit]
		}
		if in.Offset > 0 {
			out.Body.Links["prev"] = []Hyperlink{{Href: pageURL(max(0, in.Offset-in.Limit))}}
		}
		for _, h := range hits {
			out.Body.Items = append(out.Body.Items, SearchHit{Hit: h, LexicalFields: LexicalFields{h.LexicalPronunciation, h.LexicalPartOfSpeech}, Links: Links{"self": {{Href: s.Config.PublicURL + entryPath(h.Source, h.Entry) + "/senses/" + url.PathEscape(h.Sense)}}, "entry": {{Href: s.Config.PublicURL + entryPath(h.Source, h.Entry)}}}})
		}
		return out, nil
	})
	huma.Get(api, "/api/v1/sources/{source}/entries/{entry}", func(ctx context.Context, in *EntryInput) (*EntryOutput, error) {
		e, ok, err := s.findEntry(ctx, in.Source, in.Entry)
		if err != nil {
			return nil, huma.Error500InternalServerError("dictionary unavailable")
		}
		if !ok {
			return nil, huma.Error404NotFound("entry not found")
		}
		return &EntryOutput{Body: e}, nil
	})
	huma.Get(api, "/api/v1/sources/{source}/entries/{entry}/senses/{sense}", func(ctx context.Context, in *SenseInput) (*SenseOutput, error) {
		e, ok, err := s.findEntry(ctx, in.Source, in.Entry, in.Sense)
		if err != nil {
			return nil, huma.Error500InternalServerError("dictionary unavailable")
		}
		if ok {
			for _, sense := range e.Senses {
				if sense.Sense.ID == in.Sense {
					return &SenseOutput{Body: sense}, nil
				}
			}
		}
		return nil, huma.Error404NotFound("sense not found")
	})
	huma.Get(api, "/api/v1/sources", func(ctx context.Context, _ *struct{}) (*SourcesOutput, error) {
		read, err := s.Store.Read(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("dictionary unavailable")
		}
		defer read.Close()
		sources, err := read.Sources(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("dictionary unavailable")
		}
		states, err := read.Statuses(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("source freshness unavailable")
		}
		last := map[string]time.Time{}
		for _, st := range states {
			last[st.Source] = st.LastSuccess
		}
		out := &SourcesOutput{}
		out.Body.Items = []SourceView{}
		for _, sn := range sources {
			out.Body.Items = append(out.Body.Items, SourceView{sn.ID, sn.Name, sn.Entries, sn.AcceptedAt, last[sn.ID]})
		}
		return out, nil
	})
	r.Get("/.well-known/glossary", func(w http.ResponseWriter, r *http.Request) {
		if !s.Config.IncludeProjectGlossary() {
			http.NotFound(w, r)
			return
		}
		serveDictionary(w, r, glossary.JSON)
	})
	r.Get("/.well-known/glossary/{export}", s.export)
	documentPublishing(api)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, map[string]string{"status": "ok"}) })
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := s.Store.Ping(r.Context()); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		writeJSON(w, map[string]string{"status": "ready"})
	})
	r.Get("/", s.home)
	r.Get("/ui/*", s.static)
	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	r.Get("/definitions/{source}/{entry}", s.definitionPage)
	r.Get("/definitions/{source}/{entry}/{sense}", s.definitionPage)
	return r
}

// Publishing uses raw HTTP handlers to preserve canonical bytes and conditional
// responses; register their contract in the same generated OpenAPI document.
func documentPublishing(api huma.API) {
	o := api.OpenAPI()
	dictionary := o.Components.Schemas.Schema(reflect.TypeFor[vocabulary.Dictionary](), true, "")
	o.Components.SecuritySchemes = map[string]*huma.SecurityScheme{"exportSecret": {Type: "http", Scheme: "bearer", Description: "Static secret for this export; no OAuth exchange"}}
	response := func() map[string]*huma.Response {
		return map[string]*huma.Response{
			"200": {Description: "Complete canonical dictionary", Content: map[string]*huma.MediaType{"application/json": {Schema: dictionary}}, Headers: map[string]*huma.Param{"ETag": {Schema: &huma.Schema{Type: "string"}}}},
			"304": {Description: "Authorized representation is unchanged"},
		}
	}
	conditional := &huma.Param{Name: "If-None-Match", In: "header", Schema: &huma.Schema{Type: "string"}}
	projectResponses := response()
	projectResponses["404"] = &huma.Response{Description: "Project glossary disabled by configuration"}
	o.AddOperation(&huma.Operation{Method: http.MethodGet, Path: "/.well-known/glossary", OperationID: "get-project-glossary", Summary: "Read the project's own glossary when enabled", Parameters: []*huma.Param{conditional}, Responses: projectResponses})
	responses := response()
	responses["401"] = &huma.Response{Description: "Missing or invalid export credential"}
	responses["404"] = &huma.Response{Description: "No configured export with this ID"}
	responses["503"] = &huma.Response{Description: "Export unavailable (for example, waiting for initial source snapshots or forbidden re-export)"}
	o.AddOperation(&huma.Operation{Method: http.MethodGet, Path: "/.well-known/glossary/{export}", OperationID: "get-named-export", Summary: "Read a configured filtered dictionary", Description: "Access policy is configured independently for each export. Authentication is checked before conditional responses.", Parameters: []*huma.Param{{Name: "export", In: "path", Required: true, Schema: &huma.Schema{Type: "string"}}, conditional}, Security: []map[string][]string{{"exportSecret": {}}, {}}, Responses: responses})
}

func (s *Server) Admin() http.Handler {
	if !s.Config.Admin.Enabled {
		return http.NotFoundHandler()
	}
	r := router()
	api := humachi.New(r, huma.DefaultConfig("Glossary operator API", "0.1.0"))
	r.Get("/", s.operatorPage)
	r.Get("/ui/*", s.static)
	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	huma.Get(api, "/api/v1/config", func(ctx context.Context, _ *struct{}) (*struct{ Body map[string]any }, error) {
		return &struct{ Body map[string]any }{Body: s.Config.Diagnostics()}, nil
	})
	huma.Get(api, "/api/v1/scrapes", func(ctx context.Context, _ *struct{}) (*struct{ Body []store.Status }, error) {
		st, err := s.Store.Statuses(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("diagnostics unavailable")
		}
		return &struct{ Body []store.Status }{Body: st}, nil
	})
	huma.Get(api, "/api/v1/diagnostics", func(ctx context.Context, _ *struct{}) (*struct{ Body []Diagnostic }, error) {
		sn, err := s.Store.Snapshots(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("diagnostics unavailable")
		}
		return &struct{ Body []Diagnostic }{Body: diagnostics(sn)}, nil
	})
	return r
}

func (s *Server) export(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "export")
	for _, e := range s.Config.Exports {
		if e.ID == id {
			// Authenticate before reading content or checking validators. Hashing fixes comparison length.
			if !e.Auth.Public {
				want := sha256.Sum256([]byte("Bearer " + e.Token))
				got := sha256.Sum256([]byte(r.Header.Get("Authorization")))
				if e.Token == "" || subtle.ConstantTimeCompare(want[:], got[:]) != 1 {
					w.Header().Set("WWW-Authenticate", `Bearer realm="glossary-export"`)
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
			}
			snapshots, err := s.Store.Snapshots(r.Context())
			if err != nil {
				http.Error(w, "export unavailable", 503)
				return
			}
			// A source with no accepted snapshot is unavailable, not empty. Check
			// the same snapshot set we publish so initialization cannot look like
			// an intentional withdrawal to downstream consumers.
			accepted := make(map[string]bool, len(snapshots))
			for _, snapshot := range snapshots {
				accepted[snapshot.Source] = true
			}
			for _, source := range s.Config.Sources {
				if len(e.Include.Sources) > 0 && !slices.Contains(e.Include.Sources, source.ID) {
					continue
				}
				if !accepted[source.ID] {
					http.Error(w, "export unavailable; waiting for initial source snapshots", http.StatusServiceUnavailable)
					return
				}
			}
			d, err := publishing.Build(e, snapshots)
			if err != nil {
				http.Error(w, "export unavailable; inspect configuration and source provenance", 503)
				return
			}
			b, err := json.Marshal(d)
			if err != nil {
				http.Error(w, "export unavailable", 503)
				return
			}
			serveDictionary(w, r, b)
			return
		}
	}
	http.NotFound(w, r)
}

func serveDictionary(w http.ResponseWriter, r *http.Request, data []byte) {
	hash := sha256.Sum256(data)
	etag := `"` + hex.EncodeToString(hash[:]) + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "private, no-cache")
	for _, candidate := range strings.Split(r.Header.Get("If-None-Match"), ",") {
		v := strings.TrimSpace(candidate)
		if v == "*" || strings.TrimPrefix(v, "W/") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(data)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
