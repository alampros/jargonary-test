package store

//go:generate go tool sqlc generate -f ../../sqlc.yaml

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"jargonary/internal/store/storedb"
	"jargonary/internal/vocabulary"
)

func replaceNormalized(ctx context.Context, tx *sql.Tx, source string, d vocabulary.Dictionary) error {
	q := storedb.New(tx)
	for _, remove := range []func(context.Context, string) error{q.DeleteNodes, q.DeleteRelationships, q.DeleteSourceMetadata} {
		if err := remove(ctx, source); err != nil {
			return err
		}
	}
	add := func(entry, sense string, ordinal int, document any, term, pronunciation string, pos vocabulary.PartOfSpeech, rs []vocabulary.Relationship) error {
		data, err := json.Marshal(document)
		if err != nil {
			return err
		}
		if err = q.InsertNode(ctx, storedb.InsertNodeParams{Source: source, Entry: entry, Sense: sense, Ordinal: int64(ordinal), Document: data, Term: term, Pronunciation: pronunciation, PartOfSpeech: pos}); err != nil {
			return err
		}
		for i, r := range rs {
			targetSource := r.Target.Source
			if targetSource == "" {
				targetSource = source
			}
			if err = q.InsertRelationship(ctx, storedb.InsertRelationshipParams{Source: source, Entry: entry, Sense: sense, Ordinal: int64(i), Type: r.Type, TargetSource: targetSource, TargetEntry: r.Target.Entry, TargetSense: r.Target.Sense}); err != nil {
				return err
			}
		}
		return nil
	}
	senses := 0
	letters := map[string]bool{}
	for i, e := range d.Entries {
		letters[vocabulary.FirstLetter(e.Term)] = true
		header := e
		header.Senses = nil
		if err := add(e.ID, "", i, header, e.Term, e.Pronunciation, vocabulary.PartOfSpeechUnspecified, e.Relationships); err != nil {
			return err
		}
		for j, sense := range e.Senses {
			if err := add(e.ID, sense.ID, j, sense, e.Term, e.Pronunciation, sense.PartOfSpeech, sense.Relationships); err != nil {
				return err
			}
			senses++
		}
	}
	return q.InsertSourceMetadata(ctx, storedb.InsertSourceMetadataParams{Source: source, Name: d.Name, Entries: int64(len(d.Entries)), Senses: int64(senses), Letters: encodeLetters(letters)})
}

func encodeLetters(set map[string]bool) string {
	letters := make([]string, 0, len(set))
	for letter := range set {
		letters = append(letters, letter)
	}
	sort.Strings(letters)
	data, _ := json.Marshal(letters)
	return string(data)
}

// initializeSourceLetters backfills old persistent caches once after migration.
func (s *Store) initializeSourceLetters(ctx context.Context) error {
	sources, err := s.Sources(ctx)
	if err != nil {
		return err
	}
	q := storedb.New(s.db)
	for _, source := range sources {
		if source.Letters != nil {
			continue
		}
		terms, err := q.ListEntryTerms(ctx, source.ID)
		if err != nil {
			return err
		}
		letters := map[string]bool{}
		for _, term := range terms {
			letters[vocabulary.FirstLetter(term)] = true
		}
		if err := q.UpdateSourceLetters(ctx, storedb.UpdateSourceLettersParams{Source: source.ID, Letters: encodeLetters(letters)}); err != nil {
			return err
		}
	}
	return nil
}

type Source struct {
	Letters    []string
	ID         string
	Name       string
	Entries    int
	Senses     int
	AcceptedAt time.Time
}

// Reader runs content queries against either a connection or a read transaction.
type Reader struct{ db storedb.DBTX }

// ReadTx keeps multi-query pages and their relationship views on one SQLite
// snapshot. Close releases the read transaction without writing anything.
type ReadTx struct {
	*Reader
	tx *sql.Tx
}

func (s *Store) Read(ctx context.Context) (*ReadTx, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	return &ReadTx{Reader: &Reader{db: tx}, tx: tx}, nil
}
func (r *ReadTx) Close() error { return r.tx.Rollback() }

func (s *Store) Sources(ctx context.Context) ([]Source, error) {
	return (&Reader{db: s.db}).Sources(ctx)
}

func (s *Reader) Sources(ctx context.Context) ([]Source, error) {
	rows, err := storedb.New(s.db).ListSources(ctx)
	if err != nil {
		return nil, err
	}
	out := []Source{}
	for _, row := range rows {
		at, err := time.Parse(time.RFC3339Nano, row.AcceptedAt)
		if err != nil {
			return nil, err
		}
		var letters []string
		if row.Letters != "" {
			if err := json.Unmarshal([]byte(row.Letters), &letters); err != nil {
				return nil, err
			}
		}
		out = append(out, Source{Letters: letters, ID: row.Source, Name: row.Name, Entries: int(row.Entries), Senses: int(row.Senses), AcceptedAt: at})
	}
	return out, nil
}

// Edge is an authored outgoing edge or an incoming edge whose inverse is
// derived by the caller. Incoming edges always originate at existing nodes.
type Edge struct {
	From     vocabulary.Reference
	Type     string
	Target   vocabulary.Reference
	Term     string
	Resolved bool
}
type Lookup struct {
	Entry    vocabulary.Entry
	Outgoing []Edge
	Incoming []Edge
}

// LookupEntry reads the requested entry and its incident edges in one SQLite
// read transaction. No snapshot documents or unrelated nodes are decoded.
func (s *Store) LookupEntry(ctx context.Context, source, entry string, selectedSenses ...string) (Lookup, bool, error) {
	read, err := s.Read(ctx)
	if err != nil {
		return Lookup{}, false, err
	}
	defer read.Close()
	return read.LookupEntry(ctx, source, entry, selectedSenses...)
}

func (s *Reader) LookupEntry(ctx context.Context, source, entry string, selectedSenses ...string) (Lookup, bool, error) {
	selected := ""
	if len(selectedSenses) > 0 {
		selected = selectedSenses[0]
	}

	q := storedb.New(s.db)
	var result Lookup
	// Test the exact requested identity first, so sense misses are also indexed.
	data, err := q.GetNode(ctx, storedb.GetNodeParams{Source: source, Entry: entry, Sense: selected})
	if errors.Is(err, sql.ErrNoRows) {
		return result, false, nil
	}
	if err != nil {
		return result, false, err
	}
	var senses [][]byte
	if selected != "" {
		senses = [][]byte{data}
		data, err = q.GetNode(ctx, storedb.GetNodeParams{Source: source, Entry: entry})
		if err != nil {
			return result, false, err
		}
	} else {
		senses, err = q.ListSenses(ctx, storedb.ListSensesParams{Source: source, Entry: entry})
		if err != nil {
			return result, false, err
		}
	}
	if err = json.Unmarshal(data, &result.Entry); err != nil {
		return result, false, err
	}
	for _, data := range senses {
		var sense vocabulary.Sense
		if err = json.Unmarshal(data, &sense); err != nil {
			return result, false, err
		}
		result.Entry.Senses = append(result.Entry.Senses, sense)
	}
	outgoing, err := q.OutgoingRelationships(ctx, storedb.OutgoingRelationshipsParams{Source: source, Entry: entry, SelectedSense: selected})
	if err != nil {
		return result, false, err
	}
	for _, r := range outgoing {
		result.Outgoing = append(result.Outgoing, Edge{From: vocabulary.Reference{Source: r.Source, Entry: r.Entry, Sense: r.Sense}, Type: r.Type, Target: vocabulary.Reference{Source: r.TargetSource, Entry: r.TargetEntry, Sense: r.TargetSense}, Term: r.Term, Resolved: r.Resolved != 0})
	}
	incoming, err := q.IncomingRelationships(ctx, storedb.IncomingRelationshipsParams{TargetSource: source, TargetEntry: entry, SelectedSense: selected})
	if err != nil {
		return result, false, err
	}
	for _, r := range incoming {
		result.Incoming = append(result.Incoming, Edge{From: vocabulary.Reference{Source: r.Source, Entry: r.Entry, Sense: r.Sense}, Type: r.Type, Target: vocabulary.Reference{Source: r.TargetSource, Entry: r.TargetEntry, Sense: r.TargetSense}, Term: r.Term, Resolved: true})
	}
	return result, true, nil
}

// BrowseEntry contains one source-qualified entry for the browsing view.
type BrowseEntry struct {
	Source string
	Entry  vocabulary.Entry
}

// Browse reads the selected sources' entry records without canonical snapshots
// or relationship resolution. Search and definitions do not use this operation.
func (s *Store) Browse(ctx context.Context, source string) ([]BrowseEntry, error) {
	return (&Reader{db: s.db}).Browse(ctx, source)
}

func (s *Reader) Browse(ctx context.Context, source string) ([]BrowseEntry, error) {
	rows, err := storedb.New(s.db).BrowseNodes(ctx, source)
	if err != nil {
		return nil, err
	}
	out := []BrowseEntry{}
	for _, row := range rows {
		if row.Sense == "" {
			var e vocabulary.Entry
			if err = json.Unmarshal(row.Document, &e); err != nil {
				return nil, err
			}
			out = append(out, BrowseEntry{Source: row.Source, Entry: e})
		} else {
			var sense vocabulary.Sense
			if err = json.Unmarshal(row.Document, &sense); err != nil {
				return nil, err
			}
			e := &out[len(out)-1].Entry
			e.Senses = append(e.Senses, sense)
		}
	}
	return out, nil
}
