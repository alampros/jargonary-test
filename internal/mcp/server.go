// Package mcp exposes read-only glossary content as MCP tools for AI agents.
// It is opt-in and runs in-process on a private listener; it never mutates the store.
package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	mcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"jargonary/internal/store"
	"jargonary/internal/vocabulary"
)

// New returns a Streamable HTTP handler for MCP clients. The server is
// stateless: every request is independent and no session state is kept.
func New(st *store.Store) http.Handler {
	s := &Server{store: st}
	m := server.NewMCPServer("jargonary", "0.1.0")
	m.AddTool(mcp.NewTool(
		"list_glossaries",
		mcp.WithDescription("List all enabled glossary sources with their entry and sense counts. Use the returned ids to qualify other tool calls."),
	), s.listGlossaries)
	m.AddTool(mcp.NewTool(
		"search",
		mcp.WithDescription("Search glossary definitions by literal term, alias, prefix, or definition text across all or one source."),
		mcp.WithString("q", mcp.Required(), mcp.Description("The query text")),
		mcp.WithString("source", mcp.Description("Source ID; omit to search all sources")),
		mcp.WithNumber("limit", mcp.Description("Maximum results, default 20, max 100")),
		mcp.WithNumber("offset", mcp.Description("Result offset for paging, default 0")),
	), s.search)
	m.AddTool(mcp.NewTool(
		"get_entry",
		mcp.WithDescription("Read one entry with all of its senses, definitions, and context, without cross-references."),
		mcp.WithString("source", mcp.Required(), mcp.Description("Source ID")),
		mcp.WithString("entry", mcp.Required(), mcp.Description("Entry ID")),
	), s.getEntry)
	m.AddTool(mcp.NewTool(
		"get_sense",
		mcp.WithDescription("Read one sense (a single meaning of a term) with its definition and context."),
		mcp.WithString("source", mcp.Required(), mcp.Description("Source ID")),
		mcp.WithString("entry", mcp.Required(), mcp.Description("Entry ID")),
		mcp.WithString("sense", mcp.Required(), mcp.Description("Sense ID")),
	), s.getSense)
	m.AddTool(mcp.NewTool(
		"get_relationships",
		mcp.WithDescription("List the cross-references (abbreviationOf, relatedTo, deprecatedFor) around an entry or sense, optionally in one direction only."),
		mcp.WithString("source", mcp.Required(), mcp.Description("Source ID")),
		mcp.WithString("entry", mcp.Required(), mcp.Description("Entry ID")),
		mcp.WithString("sense", mcp.Description("Sense ID; omit for the whole entry")),
		mcp.WithString("direction", mcp.Description("outgoing, incoming, or both (default)")),
	), s.getRelationships)
	m.AddTool(mcp.NewTool(
		"browse_glossary",
		mcp.WithDescription("Page through the entry headings of one source (term and first definition) without full details."),
		mcp.WithString("source", mcp.Required(), mcp.Description("Source ID")),
		mcp.WithNumber("limit", mcp.Description("Maximum entries, default 50, max 200")),
		mcp.WithNumber("offset", mcp.Description("Entry offset for paging, default 0")),
	), s.browse)
	m.AddTool(mcp.NewTool(
		"list_meanings",
		mcp.WithDescription("Find every meaning of a term across sources: each matching entry with all of its senses and definitions. Unlike search, it groups senses by entry rather than returning a flat hit list."),
		mcp.WithString("term", mcp.Required(), mcp.Description("The term or alias to look up")),
		mcp.WithString("source", mcp.Description("Source ID; omit to search all sources")),
	), s.listMeanings)
	m.AddTool(mcp.NewTool(
		"resolve_term",
		mcp.WithDescription("Resolve an entry's cross-references to their target terms, returning the entry, its senses, and every outgoing and incoming relationship with the resolved target term. Use it to follow a term's network of related terms."),
		mcp.WithString("source", mcp.Required(), mcp.Description("Source ID")),
		mcp.WithString("entry", mcp.Required(), mcp.Description("Entry ID")),
		mcp.WithString("sense", mcp.Description("Sense ID; omit for the whole entry")),
	), s.resolveTerm)
	return server.NewStreamableHTTPServer(m, server.WithStateLess(true))
}

// Server answers tool calls against the shared read-only store.
type Server struct {
	store *store.Store
}

func decode[Input any](request mcp.CallToolRequest) (Input, error) {
	var in Input
	args := request.GetArguments()
	if len(args) == 0 {
		return in, nil
	}
	b, err := json.Marshal(args)
	if err != nil {
		return in, err
	}
	return in, json.Unmarshal(b, &in)
}

func respond(payload any) (*mcp.CallToolResult, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{mcp.NewTextContent(string(b))}}, nil
}

func fail(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{mcp.NewTextContent(message)}}
}

func (s *Server) listGlossaries(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sources, err := s.store.Sources(ctx)
	if err != nil {
		return nil, err
	}
	type sourceView struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Entries int    `json:"entries"`
		Senses  int    `json:"senses"`
	}
	out := []sourceView{}
	for _, src := range sources {
		out = append(out, sourceView{ID: src.ID, Name: src.Name, Entries: src.Entries, Senses: src.Senses})
	}
	return respond(out)
}

type searchInput struct {
	Q      string `json:"q"`
	Source string `json:"source"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

func (s *Server) search(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	in, err := decode[searchInput](request)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Q) == "" {
		return fail("q is required"), nil
	}
	limit := in.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if in.Offset < 0 {
		in.Offset = 0
	}
	hits, err := s.store.Search(ctx, in.Q, in.Source, limit+1, in.Offset)
	if err != nil {
		return nil, err
	}
	out := struct {
		Items []store.Hit `json:"items"`
		More  bool        `json:"more"`
	}{Items: []store.Hit{}}
	out.More = len(hits) > limit
	if out.More {
		hits = hits[:limit]
	}
	out.Items = append(out.Items, hits...)
	return respond(out)
}

type entryInput struct {
	Source string `json:"source"`
	Entry  string `json:"entry"`
	Sense  string `json:"sense"`
}

// senseView is one meaning of a term.
type senseView struct {
	ID           string                  `json:"id"`
	Definition   string                  `json:"definition"`
	PartOfSpeech vocabulary.PartOfSpeech `json:"partOfSpeech,omitempty"`
	Context      string                  `json:"context,omitempty"`
	Examples     []string                `json:"examples,omitempty"`
	Deprecated   bool                    `json:"deprecated,omitempty"`
}

// edgeView is one cross-reference with its resolved target term, when available.
type edgeView struct {
	Source   string `json:"source"`
	Entry    string `json:"entry"`
	Sense    string `json:"sense,omitempty"`
	Type     string `json:"type"`
	Term     string `json:"term,omitempty"`
	Resolved bool   `json:"resolved"`
}

// entryView is the full view returned by get_entry and resolve_term.
type entryView struct {
	Source        string      `json:"source"`
	ID            string      `json:"id"`
	Term          string      `json:"term"`
	Pronunciation string      `json:"pronunciation,omitempty"`
	Aliases       []string    `json:"aliases,omitempty"`
	Deprecated    bool        `json:"deprecated"`
	Senses        []senseView `json:"senses"`
	Outgoing      []edgeView  `json:"outgoing,omitempty"`
	Incoming      []edgeView  `json:"incoming,omitempty"`
}

// entryFromLookup maps a store lookup (entry, senses, and incident edges) to
// the shared view, attaching resolved target terms to each edge.
func entryFromLookup(source string, l store.Lookup) entryView {
	e := l.Entry
	view := entryView{
		Source:        source,
		ID:            e.ID,
		Term:          e.Term,
		Pronunciation: e.Pronunciation,
		Deprecated:    e.Deprecated,
		Senses:        []senseView{},
	}
	for _, a := range e.Aliases {
		view.Aliases = append(view.Aliases, a.Text)
	}
	for _, sense := range e.Senses {
		view.Senses = append(view.Senses, senseView{ID: sense.ID, Definition: sense.Definition, PartOfSpeech: sense.PartOfSpeech, Context: sense.Context, Examples: sense.Examples, Deprecated: sense.Deprecated})
	}
	for _, edge := range l.Outgoing {
		view.Outgoing = append(view.Outgoing, edgePayload(edge.Target, edge))
	}
	for _, edge := range l.Incoming {
		view.Incoming = append(view.Incoming, edgePayload(edge.From, edge))
	}
	return view
}

func edgePayload(ref vocabulary.Reference, edge store.Edge) edgeView {
	return edgeView{
		Source:   ref.Source,
		Entry:    ref.Entry,
		Sense:    ref.Sense,
		Type:     edge.Type,
		Term:     edge.Term,
		Resolved: edge.Resolved,
	}
}

func (s *Server) getEntry(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	in, err := decode[entryInput](request)
	if err != nil {
		return nil, err
	}
	if in.Source == "" || in.Entry == "" {
		return fail("source and entry are required"), nil
	}
	l, ok, err := s.store.LookupEntry(ctx, in.Source, in.Entry)
	if err != nil {
		return nil, err
	}
	if !ok {
		return fail("entry " + in.Entry + " not found in source " + in.Source), nil
	}
	view := entryFromLookup(in.Source, l)
	view.Outgoing, view.Incoming = nil, nil
	return respond(view)
}

func (s *Server) getSense(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	in, err := decode[entryInput](request)
	if err != nil {
		return nil, err
	}
	if in.Source == "" || in.Entry == "" || in.Sense == "" {
		return fail("source, entry, and sense are required"), nil
	}
	l, ok, err := s.store.LookupEntry(ctx, in.Source, in.Entry, in.Sense)
	if err != nil {
		return nil, err
	}
	if !ok {
		return fail("sense " + in.Sense + " not found in entry " + in.Entry + " of source " + in.Source), nil
	}
	view := entryFromLookup(in.Source, l)
	out := struct {
		Source string    `json:"source"`
		Entry  string    `json:"entry"`
		Term   string    `json:"term"`
		Sense  senseView `json:"sense"`
	}{Source: view.Source, Entry: view.ID, Term: view.Term, Sense: view.Senses[0]}
	return respond(out)
}

type relationshipInput struct {
	Source    string `json:"source"`
	Entry     string `json:"entry"`
	Sense     string `json:"sense"`
	Direction string `json:"direction"`
}

func (s *Server) getRelationships(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	in, err := decode[relationshipInput](request)
	if err != nil {
		return nil, err
	}
	if in.Source == "" || in.Entry == "" {
		return fail("source and entry are required"), nil
	}
	direction := strings.ToLower(strings.TrimSpace(in.Direction))
	if direction != "" && direction != "outgoing" && direction != "incoming" && direction != "both" {
		return fail("direction must be outgoing, incoming, or both"), nil
	}
	args := []string{}
	if in.Sense != "" {
		args = append(args, in.Sense)
	}
	l, ok, err := s.store.LookupEntry(ctx, in.Source, in.Entry, args...)
	if err != nil {
		return nil, err
	}
	if !ok {
		return fail("entry " + in.Entry + " not found in source " + in.Source), nil
	}
	out := struct {
		Source   string     `json:"source"`
		Entry    string     `json:"entry"`
		Sense    string     `json:"sense,omitempty"`
		Outgoing []edgeView `json:"outgoing"`
		Incoming []edgeView `json:"incoming"`
	}{Source: in.Source, Entry: in.Entry, Sense: in.Sense, Outgoing: []edgeView{}, Incoming: []edgeView{}}
	if direction == "" || direction == "both" || direction == "outgoing" {
		for _, edge := range l.Outgoing {
			out.Outgoing = append(out.Outgoing, edgePayload(edge.Target, edge))
		}
	}
	if direction == "" || direction == "both" || direction == "incoming" {
		for _, edge := range l.Incoming {
			out.Incoming = append(out.Incoming, edgePayload(edge.From, edge))
		}
	}
	return respond(out)
}

func (s *Server) browse(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	in, err := decode[searchInput](request)
	if err != nil {
		return nil, err
	}
	if in.Source == "" {
		return fail("source is required"), nil
	}
	entries, err := s.store.Browse(ctx, in.Source)
	if err != nil {
		return nil, err
	}
	limit := in.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if in.Offset < 0 {
		in.Offset = 0
	}
	type browseItem struct {
		ID         string `json:"id"`
		Term       string `json:"term"`
		Senses     int    `json:"senses"`
		Definition string `json:"definition,omitempty"`
	}
	out := struct {
		Items []browseItem `json:"items"`
		Total int          `json:"total"`
		More  bool         `json:"more"`
	}{Items: []browseItem{}, Total: len(entries)}
	if in.Offset >= len(entries) {
		return respond(out)
	}
	end := min(len(entries), in.Offset+limit)
	for _, e := range entries[in.Offset:end] {
		item := browseItem{ID: e.Entry.ID, Term: e.Entry.Term, Senses: len(e.Entry.Senses)}
		if len(e.Entry.Senses) > 0 {
			item.Definition = e.Entry.Senses[0].Definition
		}
		out.Items = append(out.Items, item)
	}
	out.More = end < len(entries)
	return respond(out)
}

type termInput struct {
	Term   string `json:"term"`
	Source string `json:"source"`
}

func (s *Server) listMeanings(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	in, err := decode[termInput](request)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Term) == "" {
		return fail("term is required"), nil
	}
	// Search matches the term, its aliases, and definitions; group the
	// resulting senses by entry so each entry lists all of its meanings.
	hits, err := s.store.Search(ctx, in.Term, in.Source, 100, 0)
	if err != nil {
		return nil, err
	}
	type meaningView struct {
		Source string      `json:"source"`
		Entry  string      `json:"entry"`
		Term   string      `json:"term"`
		Senses []senseView `json:"senses"`
	}
	out := []meaningView{}
	index := map[string]int{}
	for _, h := range hits {
		key := h.Source + "/" + h.Entry
		i, ok := index[key]
		if !ok {
			out = append(out, meaningView{Source: h.Source, Entry: h.Entry, Term: h.Term, Senses: []senseView{}})
			i = len(out) - 1
			index[key] = i
		}
		out[i].Senses = append(out[i].Senses, senseView{ID: h.Sense, Definition: h.Definition})
	}
	return respond(out)
}

func (s *Server) resolveTerm(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	in, err := decode[entryInput](request)
	if err != nil {
		return nil, err
	}
	if in.Source == "" || in.Entry == "" {
		return fail("source and entry are required"), nil
	}
	args := []string{}
	if in.Sense != "" {
		args = append(args, in.Sense)
	}
	l, ok, err := s.store.LookupEntry(ctx, in.Source, in.Entry, args...)
	if err != nil {
		return nil, err
	}
	if !ok {
		return fail("entry " + in.Entry + " not found in source " + in.Source), nil
	}
	return respond(entryFromLookup(in.Source, l))
}
