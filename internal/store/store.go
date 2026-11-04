// Package store persists accepted snapshots and their full-text search index in SQLite.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"jargonary/internal/vocabulary"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
	mu sync.RWMutex
}

type Snapshot struct {
	Source       string                `json:"source"`
	Dictionary   vocabulary.Dictionary `json:"dictionary"`
	AcceptedAt   time.Time             `json:"acceptedAt"`
	ETag         string                `json:"-"`
	LastModified string                `json:"-"`
	FetchKey     string                `json:"-"`
}

type Outcome struct {
	At         time.Time `json:"at"`
	Result     string    `json:"result"`
	Detail     string    `json:"detail,omitempty"`
	DurationMS int64     `json:"durationMs"`
}
type Status struct {
	Source      string    `json:"source"`
	Running     bool      `json:"running"`
	LastAttempt time.Time `json:"lastAttempt"`
	LastSuccess time.Time `json:"lastSuccess"`
	LastChange  time.Time `json:"lastChange"`
	NextAttempt time.Time `json:"nextAttempt"`
	Failures    int       `json:"failures"`
	Entries     int       `json:"entries"`
	Recent      []Outcome `json:"recent"`
}

type Hit struct {
	Source     string `json:"source"`
	Entry      string `json:"entry"`
	Term       string `json:"term"`
	Sense      string `json:"sense"`
	Definition string `json:"definition"`
	Context    string `json:"context,omitempty"`
	Deprecated bool   `json:"deprecated"`
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	fail := func(err error) (*Store, error) { _ = db.Close(); return nil, err }
	if _, err = db.Exec(`PRAGMA busy_timeout=5000; PRAGMA journal_mode=WAL; PRAGMA synchronous=FULL;`); err != nil {
		return fail(err)
	}
	var version int
	if err = db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return fail(err)
	}
	if version > 1 {
		return fail(errors.New("database schema is newer than this application; restore a compatible image or backup"))
	}
	if _, err = db.Exec(`
CREATE TABLE IF NOT EXISTS snapshots(source TEXT PRIMARY KEY, document BLOB NOT NULL, accepted_at TEXT NOT NULL, etag TEXT NOT NULL, modified TEXT NOT NULL, fetch_key TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS status(source TEXT PRIMARY KEY, document BLOB NOT NULL);
CREATE VIRTUAL TABLE IF NOT EXISTS search USING fts5(source UNINDEXED, entry UNINDEXED, sense UNINDEXED, term, aliases, definition, context, normalized UNINDEXED, alias_keys UNINDEXED, deprecated UNINDEXED, tokenize='unicode61', prefix='2 3 4');
PRAGMA user_version=1;`); err != nil {
		return fail(fmt.Errorf("initialize SQLite with FTS5: %w", err))
	}
	return s, nil
}

func (s *Store) Close() error                   { return s.db.Close() }
func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

// Keep removes cached sources deliberately removed from the startup configuration.
func (s *Store) Keep(ctx context.Context, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	keep := map[string]bool{}
	for _, id := range ids {
		keep[id] = true
	}
	rows, err := s.db.QueryContext(ctx, `SELECT source FROM snapshots UNION SELECT source FROM status`)
	if err != nil {
		return err
	}
	var remove []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		if !keep[id] {
			remove = append(remove, id)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, id := range remove {
		for _, table := range []string{"snapshots", "status", "search"} {
			if _, err = tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE source=?`, id); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (s *Store) Replace(ctx context.Context, source string, d vocabulary.Dictionary, etag, modified string, fetchKeys ...string) (bool, error) {
	fetchKey := ""
	if len(fetchKeys) > 0 {
		fetchKey = fetchKeys[0]
	}
	if err := d.Validate(); err != nil {
		return false, err
	}
	data, err := json.Marshal(d)
	if err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var previous []byte
	err = s.db.QueryRowContext(ctx, `SELECT document FROM snapshots WHERE source=?`, source).Scan(&previous)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	if string(previous) == string(data) {
		_, err = s.db.ExecContext(ctx, `UPDATE snapshots SET etag=?,modified=?,fetch_key=? WHERE source=?`, etag, modified, fetchKey, source)
		return false, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO snapshots VALUES(?,?,?,?,?,?) ON CONFLICT(source) DO UPDATE SET document=excluded.document,accepted_at=excluded.accepted_at,etag=excluded.etag,modified=excluded.modified,fetch_key=excluded.fetch_key`, source, data, time.Now().UTC().Format(time.RFC3339Nano), etag, modified, fetchKey)
	if err != nil {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM search WHERE source=?`, source); err != nil {
		return false, err
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO search(source,entry,sense,term,aliases,definition,context,normalized,alias_keys,deprecated) VALUES(?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return false, err
	}
	defer stmt.Close()
	for _, e := range d.Entries {
		for _, sense := range e.Senses {
			var aliases, keys []string
			for _, a := range e.Aliases {
				if a.Sense == "" || a.Sense == sense.ID {
					aliases = append(aliases, a.Text)
					keys = append(keys, vocabulary.Normalize(a.Text))
				}
			}
			keyJSON, _ := json.Marshal(keys)
			dep := vocabulary.Deprecated(e.Deprecated, e.Relationships) || vocabulary.Deprecated(sense.Deprecated, sense.Relationships)
			if _, err = stmt.ExecContext(ctx, source, e.ID, sense.ID, e.Term, strings.Join(aliases, " "), sense.Definition, sense.Context, vocabulary.Normalize(e.Term), string(keyJSON), dep); err != nil {
				return false, err
			}
		}
	}
	return true, tx.Commit()
}

func (s *Store) Snapshots(ctx context.Context) ([]Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.db.QueryContext(ctx, `SELECT source,document,accepted_at,etag,modified,fetch_key FROM snapshots ORDER BY source`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Snapshot{}
	for rows.Next() {
		var sn Snapshot
		var b []byte
		var at string
		if err = rows.Scan(&sn.Source, &b, &at, &sn.ETag, &sn.LastModified, &sn.FetchKey); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(b, &sn.Dictionary); err != nil {
			return nil, err
		}
		sn.AcceptedAt, err = time.Parse(time.RFC3339Nano, at)
		if err != nil {
			return nil, err
		}
		out = append(out, sn)
	}
	return out, rows.Err()
}

func (s *Store) SaveStatus(ctx context.Context, st Status) error {
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO status VALUES(?,?) ON CONFLICT(source) DO UPDATE SET document=excluded.document`, st.Source, b)
	return err
}

func (s *Store) Statuses(ctx context.Context) ([]Status, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT document FROM status ORDER BY source`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Status{}
	for rows.Next() {
		var b []byte
		var st Status
		if err = rows.Scan(&b); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(b, &st); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

func (s *Store) Search(ctx context.Context, q, source string, limit, offset int) ([]Hit, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	norm := vocabulary.Normalize(q)
	base := `SELECT source,entry,term,sense,definition,context,deprecated FROM search WHERE (?='' OR source=?)`
	args := []any{source, source}
	if norm != "" {
		// Treat input as literal text: callers cannot inject FTS operators or SQL wildcards.
		parts := strings.FieldsFunc(norm, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
		var terms []string
		for _, p := range parts {
			if utf8.RuneCountInString(p) >= 2 {
				terms = append(terms, `"`+p+`"*`)
			}
		}
		match := strings.Join(terms, " AND ")
		cte := `WITH matches AS (SELECT rowid,0 AS rank FROM search WHERE 0) `
		if match != "" {
			cte = `WITH matches AS (SELECT rowid,rank FROM search WHERE search MATCH ?) `
			args = append([]any{match}, args...)
		}
		base = cte + `SELECT search.source,search.entry,search.term,search.sense,search.definition,search.context,search.deprecated FROM search LEFT JOIN matches ON matches.rowid=search.rowid WHERE (?='' OR source=?)`
		base += ` AND (normalized=? OR EXISTS(SELECT 1 FROM json_each(alias_keys) WHERE value=?) OR normalized LIKE ? ESCAPE '!' OR matches.rowid IS NOT NULL) ORDER BY CASE WHEN normalized=? THEN 0 WHEN EXISTS(SELECT 1 FROM json_each(alias_keys) WHERE value=?) THEN 1 WHEN normalized LIKE ? ESCAPE '!' THEN 2 ELSE 3 END, coalesce(matches.rank,0),lower(term),source,entry,sense`
		prefix := strings.NewReplacer(`!`, `!!`, `%`, `!%`, `_`, `!_`).Replace(norm) + "%"
		args = append(args, norm, norm, prefix, norm, norm, prefix)
	} else {
		base += ` ORDER BY lower(term),source,entry,sense`
	}
	base += ` LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx, base, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Hit{}
	for rows.Next() {
		var h Hit
		if err = rows.Scan(&h.Source, &h.Entry, &h.Term, &h.Sense, &h.Definition, &h.Context, &h.Deprecated); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
