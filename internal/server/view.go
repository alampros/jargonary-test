package server

import (
	"fmt"
	"net/url"
	"sort"

	"jargonary/internal/store"
	"jargonary/internal/vocabulary"
)

type Hyperlink struct {
	Href  string `json:"href"`
	Title string `json:"title,omitempty"`
}
type Links map[string][]Hyperlink
type Related struct {
	Type     string               `json:"type"`
	Target   vocabulary.Reference `json:"target"`
	Resolved bool                 `json:"resolved"`
	Derived  bool                 `json:"derived"`
}
type SenseView struct {
	Source        string           `json:"source"`
	EntryID       string           `json:"entryId"`
	Term          string           `json:"term"`
	Pronunciation string           `json:"pronunciation,omitempty"`
	Sense         vocabulary.Sense `json:"sense"`
	Deprecated    bool             `json:"deprecated"`
	Relationships []Related        `json:"relationships"`
	Links         Links            `json:"_links"`
}
type EntryView struct {
	Source            string                `json:"source"`
	ID                string                `json:"id"`
	Origin            *vocabulary.Reference `json:"origin,omitempty"`
	Term              string                `json:"term"`
	Pronunciation     string                `json:"pronunciation,omitempty"`
	Aliases           []vocabulary.Alias    `json:"aliases,omitempty"`
	Deprecated        bool                  `json:"deprecated"`
	DeprecationReason string                `json:"deprecationReason,omitempty"`
	Relationships     []Related             `json:"relationships"`
	Senses            []SenseView           `json:"senses"`
	Links             Links                 `json:"_links"`
}
type node struct {
	ref           vocabulary.Reference
	term          string
	explicit      []vocabulary.Relationship
	relationships []Related
	links         Links
}
type graph map[vocabulary.Reference]*node

func entryPath(source, entry string) string {
	return "/api/v1/sources/" + url.PathEscape(source) + "/entries/" + url.PathEscape(entry)
}
func refPath(r vocabulary.Reference) string {
	p := entryPath(r.Source, r.Entry)
	if r.Sense != "" {
		p += "/senses/" + url.PathEscape(r.Sense)
	}
	return p
}

func buildGraph(base string, snapshots []store.Snapshot) graph {
	g := graph{}
	for _, sn := range snapshots {
		for _, e := range sn.Dictionary.Entries {
			r := vocabulary.Reference{Source: sn.Source, Entry: e.ID}
			g[r] = &node{ref: r, term: e.Term, explicit: e.Relationships, relationships: []Related{}, links: Links{"self": {{Href: base + refPath(r), Title: e.Term}}}}
			for _, s := range e.Senses {
				r := vocabulary.Reference{Source: sn.Source, Entry: e.ID, Sense: s.ID}
				g[r] = &node{ref: r, term: e.Term, explicit: s.Relationships, relationships: []Related{}, links: Links{"self": {{Href: base + refPath(r), Title: e.Term}}, "entry": {{Href: base + entryPath(sn.Source, e.ID), Title: e.Term}}}}
			}
		}
	}
	// Enumerate explicit edges deterministically before deriving inverse links.
	var refs []vocabulary.Reference
	for r := range g {
		refs = append(refs, r)
	}
	sort.Slice(refs, func(i, j int) bool { return refPath(refs[i]) < refPath(refs[j]) })
	seen := map[string]bool{}
	add := func(from vocabulary.Reference, typ string, to vocabulary.Reference, derived bool) {
		key := fmt.Sprintf("%s\x00%s\x00%s", refPath(from), typ, refPath(to))
		if seen[key] {
			return
		}
		seen[key] = true
		n := g[from]
		target, ok := g[to]
		n.relationships = append(n.relationships, Related{Type: typ, Target: to, Resolved: ok, Derived: derived})
		if ok {
			n.links[typ] = append(n.links[typ], Hyperlink{Href: base + refPath(to), Title: target.term})
		}
	}
	for _, r := range refs {
		for _, rel := range g[r].explicit {
			target := rel.Target
			if target.Source == "" {
				target.Source = r.Source
			}
			add(r, rel.Type, target, false)
		}
	}
	for _, r := range refs {
		for _, rel := range g[r].explicit {
			target := rel.Target
			if target.Source == "" {
				target.Source = r.Source
			}
			if _, ok := g[target]; ok {
				add(target, vocabulary.Inverse[rel.Type], r, true)
			}
		}
	}
	return g
}

func findEntry(base string, snapshots []store.Snapshot, source, id string) (EntryView, bool) {
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

type Diagnostic struct {
	Source string `json:"source"`
	Entry  string `json:"entry"`
	Sense  string `json:"sense,omitempty"`
	Kind   string `json:"kind"`
}

func diagnostics(snapshots []store.Snapshot) []Diagnostic {
	g := buildGraph("", snapshots)
	out := []Diagnostic{}
	for _, n := range g {
		for _, r := range n.relationships {
			if !r.Resolved {
				out = append(out, Diagnostic{n.ref.Source, n.ref.Entry, n.ref.Sense, "unresolved relationship"})
			}
			if r.Type == "deprecatedFor" && r.Target == n.ref {
				out = append(out, Diagnostic{n.ref.Source, n.ref.Entry, n.ref.Sense, "self replacement"})
			}
		}
	}
	// Detect replacement cycles, without recursive traversal or automatic redirects.
	for start, n := range g {
		visited := map[vocabulary.Reference]bool{}
		queue := []vocabulary.Reference{}
		for _, r := range n.relationships {
			if r.Type == "deprecatedFor" && r.Resolved {
				queue = append(queue, r.Target)
			}
		}
		for len(queue) > 0 {
			r := queue[0]
			queue = queue[1:]
			if r == start {
				out = append(out, Diagnostic{start.Source, start.Entry, start.Sense, "replacement cycle"})
				break
			}
			if visited[r] {
				continue
			}
			visited[r] = true
			for _, rel := range g[r].relationships {
				if rel.Type == "deprecatedFor" && rel.Resolved {
					queue = append(queue, rel.Target)
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return fmt.Sprint(out[i]) < fmt.Sprint(out[j]) })
	return out
}

// Lexical fields come from accepted snapshots, so older indexes need no migration.
type LexicalFields struct {
	Pronunciation string `json:"pronunciation,omitempty"`
	PartOfSpeech  string `json:"partOfSpeech,omitempty"`
}

func lexicalMetadata(snapshots []store.Snapshot) map[vocabulary.Reference]LexicalFields {
	out := map[vocabulary.Reference]LexicalFields{}
	for _, sn := range snapshots {
		for _, entry := range sn.Dictionary.Entries {
			for _, sense := range entry.Senses {
				out[vocabulary.Reference{Source: sn.Source, Entry: entry.ID, Sense: sense.ID}] = LexicalFields{entry.Pronunciation, sense.PartOfSpeech}
			}
		}
	}
	return out
}
