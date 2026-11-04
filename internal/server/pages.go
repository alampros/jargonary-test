package server

import (
	"bytes"
	"context"
	"encoding/json"
	"html"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"
	"jargonary/internal/store"
	"jargonary/internal/vocabulary"
	"jargonary/internal/webui"
)

type pageData struct {
	Mode          string                    `json:"mode"`
	SiteName      string                    `json:"siteName"`
	Base          string                    `json:"base"`
	Query         string                    `json:"query"`
	Source        string                    `json:"source"`
	Letter        string                    `json:"letter"`
	Sources       []pageSource              `json:"sources"`
	Entries       []pageEntry               `json:"entries"`
	TermPage      bool                      `json:"termPage,omitempty"`
	TermEntries   []EntryView               `json:"termEntries,omitempty"`
	TermURL       string                    `json:"termURL,omitempty"`
	Entry         *EntryView                `json:"entry,omitempty"`
	EntryURL      string                    `json:"entryURL,omitempty"`
	Relationships map[string][]pageRelation `json:"relationships"`
	NotFound      bool                      `json:"notFound"`
	Next          string                    `json:"next,omitempty"`
	Previous      string                    `json:"previous,omitempty"`
}
type pageSource struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Stale       bool       `json:"stale"`
	LastChecked *time.Time `json:"lastChecked,omitempty"`
}

func (s *Server) publicSources(ctx context.Context, reader *store.Reader, sources []store.Source) ([]pageSource, error) {
	states, err := reader.Statuses(ctx)
	if err != nil {
		return nil, err
	}
	byID := map[string]store.Status{}
	for _, state := range states {
		byID[state.Source] = state
	}
	out := []pageSource{}
	for _, snapshot := range sources {
		state := byID[snapshot.ID]
		item := pageSource{ID: snapshot.ID, Name: snapshot.Name, Stale: state.Failures > 0}
		if !state.LastSuccess.IsZero() {
			last := state.LastSuccess
			item.LastChecked = &last
		}
		out = append(out, item)
	}
	return out, nil
}

type pageEntry struct {
	TermKey       string   `json:"termKey"`
	Source        string   `json:"source"`
	ID            string   `json:"id"`
	Term          string   `json:"term"`
	Contexts      []string `json:"contexts"`
	Pronunciation string   `json:"pronunciation,omitempty"`
	PartsOfSpeech []string `json:"partsOfSpeech"`
	Definitions   []string `json:"definitions"`
	Deprecated    bool     `json:"deprecated"`
	TermHref      string   `json:"termHref"`
	Href          string   `json:"href"`
}
type pageRelation struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	Href  string `json:"href"`
}

func definitionURL(base string, r vocabulary.Reference) string {
	p := base + "/definitions/" + url.PathEscape(r.Source) + "/" + url.PathEscape(r.Entry)
	if r.Sense != "" {
		p += "/" + url.PathEscape(r.Sense)
	}
	return p
}

func (s *Server) basePath() string {
	u, err := url.Parse(s.Config.PublicURL)
	if err != nil {
		return ""
	}
	return strings.TrimRight(u.Path, "/")
}

func (s *Server) uiAssets() fs.FS {
	if s.Assets != nil {
		return s.Assets
	}
	return webui.Assets()
}

func (s *Server) static(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "*")
	if !fs.ValidPath(name) || strings.HasPrefix(name, ".") {
		http.NotFound(w, r)
		return
	}
	if s.appearanceAsset(w, r, name) {
		return
	}
	// Only compiled assets are served; source files and the HTML bootstrap template are never exposed.
	if !strings.HasPrefix(name, "assets/") {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeFileFS(w, r, s.uiAssets(), name)
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	read, err := s.Store.Read(r.Context())
	if err != nil {
		http.Error(w, "Dictionary unavailable", 503)
		return
	}
	defer read.Close()
	sources, err := read.Sources(r.Context())
	if err != nil {
		http.Error(w, "Dictionary unavailable", 503)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	source := r.URL.Query().Get("source")
	letter := strings.ToUpper(r.URL.Query().Get("letter"))
	if len(q) > 256 || len(source) > 200 || (letter != "" && letter != "#" && letter != "0-9" && (len(letter) != 1 || letter[0] < 'A' || letter[0] > 'Z')) {
		http.Error(w, "Invalid search or letter filter", 400)
		return
	}
	if q != "" {
		letter = ""
	}
	data := pageData{Mode: "public", SiteName: s.Config.SiteName, Base: s.basePath(), Query: q, Source: source, Letter: letter, Sources: []pageSource{}, Entries: []pageEntry{}}
	data.Sources, err = s.publicSources(r.Context(), read.Reader, sources)
	if err != nil {
		http.Error(w, "Source freshness unavailable", 503)
		return
	}
	// Preserve source-qualified entries; paginate by term so matching headings stay together.
	var matches []pageEntry
	if q != "" {
		totalSenses := 0
		for _, sn := range sources {
			totalSenses += sn.Senses
		}
		hits, err := read.Search(r.Context(), q, source, max(1, totalSenses), 0)
		if err != nil {
			http.Error(w, "Search unavailable", 503)
			return
		}
		indices := map[vocabulary.Reference]int{}
		for _, hit := range hits {
			key := vocabulary.Reference{Source: hit.Source, Entry: hit.Entry}
			i, ok := indices[key]
			if !ok {
				i = len(matches)
				indices[key] = i
				matches = append(matches, pageEntry{Source: hit.Source, ID: hit.Entry, Term: hit.Term, Deprecated: true})
			}
			m := &matches[i]
			fields := LexicalFields{hit.LexicalPronunciation, hit.LexicalPartOfSpeech}
			m.Pronunciation = fields.Pronunciation
			m.PartsOfSpeech = append(m.PartsOfSpeech, fields.PartOfSpeech)
			m.Contexts = append(m.Contexts, hit.Context)
			m.Definitions = append(m.Definitions, hit.Definition)
			m.Deprecated = m.Deprecated && hit.Deprecated
		}
	} else {
		entries, err := read.Browse(r.Context(), source)
		if err != nil {
			http.Error(w, "Dictionary unavailable", 503)
			return
		}
		for _, sn := range entries {
			entry := sn.Entry
			if letter != "" && firstLetter(entry.Term) != letter {
				continue
			}
			m := pageEntry{Source: sn.Source, ID: entry.ID, Term: entry.Term, Pronunciation: entry.Pronunciation, Deprecated: true}
			for _, sense := range entry.Senses {
				m.PartsOfSpeech = append(m.PartsOfSpeech, sense.PartOfSpeech)
				m.Contexts = append(m.Contexts, sense.Context)
				m.Definitions = append(m.Definitions, sense.Definition)
				m.Deprecated = m.Deprecated && (vocabulary.Deprecated(entry.Deprecated, entry.Relationships) || vocabulary.Deprecated(sense.Deprecated, sense.Relationships))
			}
			matches = append(matches, m)
		}
		sort.Slice(matches, func(i, j int) bool {
			a, b := matches[i], matches[j]
			if vocabulary.Normalize(a.Term) == vocabulary.Normalize(b.Term) {
				if a.Source == b.Source {
					return a.ID < b.ID
				}
				return a.Source < b.Source
			}
			return vocabulary.Normalize(a.Term) < vocabulary.Normalize(b.Term)
		})
	}
	page := 0
	if raw := r.URL.Query().Get("page"); raw != "" {
		page, err = strconv.Atoi(raw)
		if err != nil || page < 0 || page > 1000000 {
			http.Error(w, "Invalid page", 400)
			return
		}
	}
	pageURL := func(page int) string {
		values := url.Values{"q": {q}, "source": {source}, "letter": {letter}, "page": {strconv.Itoa(page)}}
		return data.Base + "/?" + values.Encode()
	}
	for i := range matches {
		matches[i].TermKey = vocabulary.Normalize(matches[i].Term)
	}
	groups := groupPageEntries(matches)
	start := min(page*20, len(groups))
	end := min(start+20, len(groups))
	for _, group := range groups[start:end] {
		data.Entries = append(data.Entries, group...)
	}
	if end < len(groups) {
		data.Next = pageURL(page + 1)
	}
	if page > 0 {
		data.Previous = pageURL(page - 1)
	}
	if len(matches) == 1 && q != "" {
		lookup, ok, err := read.LookupEntry(r.Context(), matches[0].Source, matches[0].ID)
		if err != nil {
			http.Error(w, "Dictionary unavailable", 503)
			return
		}
		if !ok {
			_ = read.Close()
			s.renderPublic(w, r, data, 200)
			return
		}
		entry := entryView(s.Config.PublicURL, matches[0].Source, lookup)
		data.Entry = &entry
		if data.Letter == "" {
			data.Letter = firstLetter(entry.Term)
		}
	}
	_ = read.Close() // Release the database connection before writing the response.
	s.renderPublic(w, r, data, 200)
}

// groupPageEntries retains the first occurrence's search rank and each entry's identity.
func groupPageEntries(entries []pageEntry) [][]pageEntry {
	groups := [][]pageEntry{}
	indices := map[string]int{}
	for _, entry := range entries {
		key := vocabulary.Normalize(entry.Term)
		i, ok := indices[key]
		if !ok {
			i = len(groups)
			indices[key] = i
			groups = append(groups, nil)
		}
		groups[i] = append(groups[i], entry)
	}
	return groups
}

func firstLetter(term string) string {
	for _, r := range strings.TrimSpace(term) {
		if r >= 'a' && r <= 'z' {
			return strings.ToUpper(string(r))
		}
		if r >= 'A' && r <= 'Z' {
			return string(r)
		}
		if unicode.IsDigit(r) {
			return "0-9"
		}
		return "#"
	}
	return "#"
}

func termURL(base, term, source string) string {
	path := base + "/definitions/" + url.PathEscape(term)
	if source != "" {
		path += "?" + url.Values{"source": {source}}.Encode()
	}
	return path
}

// termPage presents matching terms together without merging their source identities.
func (s *Server) termPage(w http.ResponseWriter, r *http.Request) {
	term := chi.URLParam(r, "term")
	if r.URL.RawPath != "" {
		var err error
		term, err = url.PathUnescape(term)
		if err != nil {
			http.Error(w, "Invalid term", http.StatusBadRequest)
			return
		}
	}
	term = strings.TrimSpace(term)
	source := r.URL.Query().Get("source")
	if term == "" || len(term) > 256 || len(source) > 200 {
		http.Error(w, "Invalid term", http.StatusBadRequest)
		return
	}
	read, err := s.Store.Read(r.Context())
	if err != nil {
		http.Error(w, "Glossary unavailable", 503)
		return
	}
	defer read.Close()
	sources, err := read.Sources(r.Context())
	if err != nil {
		http.Error(w, "Glossary unavailable", 503)
		return
	}
	data := pageData{Mode: "public", SiteName: s.Config.SiteName, Base: s.basePath(), Source: source, TermPage: true, Sources: []pageSource{}, Entries: []pageEntry{}}
	data.TermURL = termURL(data.Base, term, source)
	data.Sources, err = s.publicSources(r.Context(), read.Reader, sources)
	if err != nil {
		http.Error(w, "Source freshness unavailable", 503)
		return
	}
	entries, err := read.Browse(r.Context(), source)
	if err != nil {
		http.Error(w, "Glossary unavailable", 503)
		return
	}
	for _, candidate := range entries {
		if vocabulary.Normalize(candidate.Entry.Term) != vocabulary.Normalize(term) {
			continue
		}
		lookup, ok, err := read.LookupEntry(r.Context(), candidate.Source, candidate.Entry.ID)
		if err != nil {
			http.Error(w, "Glossary unavailable", 503)
			return
		}
		if ok {
			data.TermEntries = append(data.TermEntries, entryView(s.Config.PublicURL, candidate.Source, lookup))
		}
	}
	status := 200
	if len(data.TermEntries) > 0 {
		data.Entry = &data.TermEntries[0]
		data.TermURL = termURL(data.Base, data.Entry.Term, data.Source)
		data.Letter = firstLetter(data.Entry.Term)
	} else {
		data.NotFound = true
		status = 404
	}
	_ = read.Close()
	s.renderPublic(w, r, data, status)
}

func (s *Server) definitionPage(w http.ResponseWriter, r *http.Request) {
	read, err := s.Store.Read(r.Context())
	if err != nil {
		http.Error(w, "Dictionary unavailable", 503)
		return
	}
	defer read.Close()
	sources, err := read.Sources(r.Context())
	if err != nil {
		http.Error(w, "Dictionary unavailable", 503)
		return
	}
	source, id, sense := chi.URLParam(r, "source"), chi.URLParam(r, "entry"), chi.URLParam(r, "sense")
	data := pageData{Mode: "public", SiteName: s.Config.SiteName, Base: s.basePath(), Source: "", Sources: []pageSource{}, Entries: []pageEntry{}}
	data.Sources, err = s.publicSources(r.Context(), read.Reader, sources)
	if err != nil {
		http.Error(w, "Source freshness unavailable", 503)
		return
	}
	lookup, ok, err := read.LookupEntry(r.Context(), source, id)
	if err != nil {
		http.Error(w, "Dictionary unavailable", 503)
		return
	}
	entry := entryView(s.Config.PublicURL, source, lookup)
	if ok && sense != "" {
		var selected []SenseView
		for _, sv := range entry.Senses {
			if sv.Sense.ID == sense {
				selected = append(selected, sv)
			}
		}
		entry.Senses = selected
		ok = len(selected) > 0
	}
	status := 200
	if ok {
		data.Entry = &entry
		data.Letter = firstLetter(entry.Term)
	} else {
		data.NotFound = true
		status = 404
	}
	_ = read.Close() // Rendering and slow clients must not hold the read transaction.
	s.renderPublic(w, r, data, status)
}

var pageHead = template.Must(template.New("head").Parse(`<title>{{.Title}}</title><meta name="description" content="{{.Description}}"><meta property="og:type" content="website"><meta property="og:title" content="{{.Title}}"><meta property="og:description" content="{{.Description}}">{{if .Canonical}}<link rel="canonical" href="{{.Canonical}}"><meta property="og:url" content="{{.Canonical}}">{{end}}`))
var fallbackPage = template.Must(template.New("fallback").Funcs(template.FuncMap{
	"groupTerms": groupPageEntries,
	"detailEntries": func(data pageData) []EntryView {
		if len(data.TermEntries) > 0 {
			return data.TermEntries
		}
		if data.Entry != nil {
			return []EntryView{*data.Entry}
		}
		return nil
	},
	"multipleSources": func(entries []pageEntry) bool {
		for _, entry := range entries {
			if entry.Source != entries[0].Source {
				return true
			}
		}
		return false
	},
	"sourceName": func(data pageData, id string) string {
		for _, source := range data.Sources {
			if source.ID == id {
				return source.Name
			}
		}
		return id
	},
}).Parse(`<main><h1>{{if .Entry}}{{.Entry.Term}}{{else}}Shared glossary{{end}}</h1>{{if .NotFound}}<p>This definition is no longer available.</p>{{end}}{{if .Entry}}{{range detailEntries .}}{{if .Pronunciation}}<p>{{.Pronunciation}}</p>{{end}}{{range .Senses}}<section>{{if .Sense.Context}}<h2>{{.Sense.Context}}</h2>{{end}}{{if .Deprecated}}<p><strong>Deprecated</strong></p>{{end}}{{if .Sense.PartOfSpeech}}<p><em>{{.Sense.PartOfSpeech}}</em></p>{{end}}<p>{{.Sense.Definition}}</p>{{range .Sense.Examples}}<blockquote>{{.}}</blockquote>{{end}}</section>{{end}}{{end}}{{else}}{{range groupTerms .Entries}}{{$showSources := multipleSources .}}<article><h2>{{(index . 0).Term}}</h2>{{range .}}{{$entry := .}}{{range $i, $definition := .Definitions}}<p>{{if $showSources}}<a href="{{$entry.Href}}">{{sourceName $ $entry.Source}}</a>{{end}}{{with index $entry.Contexts $i}}{{if $showSources}} · {{end}}{{.}}{{end}}</p><p>{{$definition}}</p>{{end}}{{end}}</article>{{end}}{{end}}</main>`))

func (s *Server) renderPublic(w http.ResponseWriter, r *http.Request, data pageData, status int) {
	data.Relationships = map[string][]pageRelation{}
	for i := range data.Entries {
		e := &data.Entries[i]
		e.Href = definitionURL(data.Base, vocabulary.Reference{Source: e.Source, Entry: e.ID})
		e.TermHref = termURL(data.Base, e.Term, data.Source)
	}
	if data.Entry != nil {
		e := data.Entry
		data.TermURL = termURL(data.Base, e.Term, data.Source)
		data.EntryURL = definitionURL(data.Base, vocabulary.Reference{Source: e.Source, Entry: e.ID})
		if sense := chi.URLParam(r, "sense"); sense != "" {
			data.EntryURL += "/" + url.PathEscape(sense)
		}
		add := func(key string, rs []Related, links Links) {
			for _, rel := range rs {
				if !rel.Resolved {
					continue
				}
				title := rel.Target.Entry
				for _, link := range links[rel.Type] {
					if strings.HasSuffix(link.Href, refPath(rel.Target)) {
						title = link.Title
						break
					}
				}
				data.Relationships[key] = append(data.Relationships[key], pageRelation{rel.Type, title, definitionURL(data.Base, rel.Target)})
			}
		}
		entries := data.TermEntries
		if len(entries) == 0 {
			entries = []EntryView{*e}
		} else {
			data.EntryURL = data.TermURL
		}
		for _, entry := range entries {
			prefix := ""
			if len(data.TermEntries) > 0 {
				prefix = entry.Source + "/" + entry.ID + "/"
			}
			add(prefix+"entry", entry.Relationships, entry.Links)
			for _, sense := range entry.Senses {
				add(prefix+"sense:"+sense.Sense.ID, sense.Relationships, sense.Links)
			}
		}
	}
	title := s.Config.SiteName + " — Shared glossary"
	description := "A place for your shared vocabulary."
	canonical := ""
	if data.Entry != nil {
		title = data.Entry.Term + " — " + s.Config.SiteName
		description = data.Entry.Senses[0].Sense.Definition
		if s.Config.PublicURL != "" {
			canonical = s.Config.PublicURL + "/definitions/" + url.PathEscape(data.Entry.Source) + "/" + url.PathEscape(data.Entry.ID)
			if len(data.Entry.Senses) == 1 && chi.URLParam(r, "sense") != "" {
				canonical += "/" + url.PathEscape(data.Entry.Senses[0].Sense.ID)
			}
		}
	}
	if len(data.TermEntries) > 0 && s.Config.PublicURL != "" {
		canonical = termURL(strings.TrimRight(s.Config.PublicURL, "/"), data.Entry.Term, data.Source)
	}
	var fallback bytes.Buffer
	_ = fallbackPage.Execute(&fallback, data)
	s.renderShell(w, r, data, title, description, canonical, fallback.String(), status)
}

func (s *Server) renderShell(w http.ResponseWriter, r *http.Request, data any, title, description, canonical, fallback string, status int) {
	index, err := fs.ReadFile(s.uiAssets(), "index.html")
	if s.DevUI != nil {
		index, err = fs.ReadFile(s.DevUI, "index.html")
		if err != nil {
			http.Error(w, "Development template unavailable; run from the repository root", 503)
			return
		}
		index = []byte(strings.ReplaceAll(string(index), `src="/src/main.tsx"`, `src="http://127.0.0.1:5173/src/main.tsx"`))
		index = []byte(strings.ReplaceAll(string(index), "<!--page-head-->", "<!--page-head-->"+viteDevelopmentScripts))
		// Vite injects styles and connects to its own origin for modules and HMR.
		// These permissions apply only to HTML explicitly served in development mode.
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' http://127.0.0.1:5173; style-src 'self' 'unsafe-inline' http://127.0.0.1:5173; font-src 'self' http://127.0.0.1:5173; img-src 'self' data: http://127.0.0.1:5173; connect-src 'self' http://127.0.0.1:5173 ws://127.0.0.1:5173; frame-ancestors 'none'; base-uri 'none'")
	}
	if err != nil {
		// Plain definition HTML remains readable in API-only builds. A missing web
		// build is explicit on the homepage rather than silently returning a 404.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Path == "/" {
			w.WriteHeader(503)
			_, _ = w.Write([]byte("<!doctype html><html lang=\"en\"><title>Frontend build required</title><h1>Frontend build required</h1><p>Run <code>go tool task ui</code>, then restart the Go server.</p></html>"))
			return
		}
		var head bytes.Buffer
		_ = pageHead.Execute(&head, struct{ Title, Description, Canonical string }{title, description, canonical})
		w.WriteHeader(status)
		_, _ = w.Write([]byte("<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\">" + head.String() + "</head><body>" + fallback + "</body></html>"))
		return
	}
	b, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Page unavailable", 503)
		return
	}
	var head bytes.Buffer
	_ = pageHead.Execute(&head, struct{ Title, Description, Canonical string }{title, description, canonical})
	base := s.basePath()
	if _, ok := data.(adminPageData); ok {
		base = ""
	}
	page := strings.ReplaceAll(string(index), "./assets/", base+"/ui/assets/")
	mode := "public"
	if _, ok := data.(adminPageData); ok {
		mode = "admin"
	}
	theme := s.Config.Theme
	if theme == "" {
		theme = "default"
	}
	page = strings.Replace(page, `<html lang="en">`, `<html lang="en" data-theme="`+html.EscapeString(theme)+`" data-ui="`+mode+`">`, 1)
	page = strings.Replace(page, "</head>", s.appearanceHead(base)+"</head>", 1)
	page = strings.ReplaceAll(page, "<!--page-head-->", head.String())
	page = strings.ReplaceAll(page, "<!--page-data-->", html.EscapeString(string(b)))
	page = strings.ReplaceAll(page, "<!--page-content-->", fallback)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(page))
}

// React Refresh must be initialized before the entry module when Go serves HTML.
// See https://vite.dev/guide/backend-integration.
const viteDevelopmentScripts = `<script type="module">
import RefreshRuntime from "http://127.0.0.1:5173/@react-refresh";
RefreshRuntime.injectIntoGlobalHook(window);
window.$RefreshReg$ = () => {};
window.$RefreshSig$ = () => (type) => type;
window.__vite_plugin_react_preamble_installed__ = true;
</script>
<script type="module" src="http://127.0.0.1:5173/@vite/client"></script>`

type adminPageData struct {
	Mode        string         `json:"mode"`
	SiteName    string         `json:"siteName"`
	Config      map[string]any `json:"config"`
	Scrapes     []store.Status `json:"scrapes"`
	Diagnostics []Diagnostic   `json:"diagnostics"`
}

func (s *Server) operatorPage(w http.ResponseWriter, r *http.Request) {
	states, err := s.Store.Statuses(r.Context())
	if err != nil {
		http.Error(w, "Diagnostics unavailable", 503)
		return
	}
	snapshots, err := s.Store.Snapshots(r.Context())
	if err != nil {
		http.Error(w, "Diagnostics unavailable", 503)
		return
	}
	data := adminPageData{Mode: "admin", SiteName: s.Config.SiteName, Config: s.Config.Diagnostics(), Scrapes: states, Diagnostics: diagnostics(snapshots)}
	s.renderShell(w, r, data, "Operator console — "+s.Config.SiteName, "Configuration and source synchronization diagnostics.", "", "<main><h1>Operator console</h1><p>Read-only source and configuration diagnostics.</p></main>", 200)
}
