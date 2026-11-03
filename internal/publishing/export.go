// Package publishing builds filtered dictionaries before serialization.
package publishing

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"

	"jargonary/internal/config"
	"jargonary/internal/store"
	"jargonary/internal/vocabulary"
)

// ExportID hashes an unambiguous (source, entry) pair. Display names and consumer source IDs do not affect it.
func ExportID(source, entry string) string {
	digest := sha256.Sum256([]byte(source + "\x00" + entry))
	return "x-" + hex.EncodeToString(digest[:])
}

func Build(e config.Export, snapshots []store.Snapshot) (vocabulary.Dictionary, error) {
	d := vocabulary.Dictionary{Version: "1", Name: e.Name, Exported: true, Entries: []vocabulary.Entry{}}
	selected := map[string]map[string]bool{}
	type origin struct {
		source string
		entry  vocabulary.Entry
	}
	var origins []origin
	for _, sn := range snapshots {
		if sn.Source == "project" && !slices.Contains(e.Include.Sources, "project") {
			continue
		}
		if len(e.Include.Sources) > 0 && !slices.Contains(e.Include.Sources, sn.Source) {
			continue
		}
		for _, entry := range sn.Dictionary.Entries {
			copy := entry
			copy.Senses = nil
			for _, sense := range entry.Senses {
				match := e.Include.All
				for _, tag := range e.Include.SenseTags.Any {
					if slices.Contains(sense.Tags, tag) {
						match = true
					}
				}
				if match {
					copy.Senses = append(copy.Senses, sense)
				}
			}
			if len(copy.Senses) == 0 {
				continue
			}
			if sn.Dictionary.Exported {
				return d, errors.New("re-exporting imported exports is disabled in v1")
			}
			id := ExportID(sn.Source, entry.ID)
			if !vocabulary.ValidID(id) {
				return d, errors.New("exported identity exceeds canonical ID length; use shorter source and entry IDs")
			}
			selected[id] = map[string]bool{}
			for _, sense := range copy.Senses {
				selected[id][sense.ID] = true
			}
			origins = append(origins, origin{sn.Source, copy})
		}
	}
	for _, o := range origins {
		entry := o.entry
		entry.Origin = &vocabulary.Reference{Source: o.source, Entry: entry.ID}
		entry.ID = ExportID(o.source, entry.ID)
		// Materialize status before pruning replacement references so hidden targets cannot reactivate a term.
		entry.Deprecated = vocabulary.Deprecated(entry.Deprecated, entry.Relationships)
		entry.Relationships = prune(entry.Relationships, o.source, selected)
		entry.Aliases = nil
		for _, a := range o.entry.Aliases {
			if a.Sense == "" || selected[entry.ID][a.Sense] {
				entry.Aliases = append(entry.Aliases, a)
			}
		}
		entry.Senses = append([]vocabulary.Sense(nil), entry.Senses...)
		for i := range entry.Senses {
			s := &entry.Senses[i]
			s.Deprecated = vocabulary.Deprecated(s.Deprecated, s.Relationships)
			s.Relationships = prune(s.Relationships, o.source, selected)
		}
		d.Entries = append(d.Entries, entry)
	}
	return d, d.Validate()
}

func prune(rs []vocabulary.Relationship, source string, selected map[string]map[string]bool) []vocabulary.Relationship {
	var out []vocabulary.Relationship
	for _, r := range rs {
		targetSource := r.Target.Source
		if targetSource == "" {
			targetSource = source
		}
		id := ExportID(targetSource, r.Target.Entry)
		senses, ok := selected[id]
		if !ok || (r.Target.Sense != "" && !senses[r.Target.Sense]) {
			continue
		}
		r.Target.Source = ""
		r.Target.Entry = id
		out = append(out, r)
	}
	return out
}
