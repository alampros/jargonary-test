package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	v "jargonary/internal/vocabulary"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "lookup.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
func lookupDictionary(term string) v.Dictionary {
	return v.Dictionary{Version: "1", Name: term, Entries: []v.Entry{{ID: "same", Term: term, Pronunciation: "spoken " + term, Aliases: []v.Alias{{Text: "alias", Sense: "one"}}, Senses: []v.Sense{{ID: "one", Definition: term, PartOfSpeech: v.PartOfSpeechNoun}, {ID: "two", Definition: "other " + term}}}}}
}
func replaceTest(t *testing.T, s *Store, source string, d v.Dictionary) {
	t.Helper()
	if _, err := s.Replace(context.Background(), source, d, "etag", "modified", "fetch-key"); err != nil {
		t.Fatal(err)
	}
}

func TestLookupSourceIdentityAndLifecycle(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	a, b := lookupDictionary("Alpha"), lookupDictionary("Beta")
	a.Entries[0].Relationships = []v.Relationship{{Type: "relatedTo", Target: v.Reference{Source: "b", Entry: "same", Sense: "one"}}}
	replaceTest(t, s, "a", a)
	check := func(resolved bool) {
		t.Helper()
		got, ok, err := s.LookupEntry(ctx, "a", "same")
		if err != nil || !ok {
			t.Fatalf("lookup: %v %v", ok, err)
		}
		if !reflect.DeepEqual(got.Entry, a.Entries[0]) || len(got.Outgoing) != 1 || got.Outgoing[0].Resolved != resolved {
			t.Fatalf("lookup: %+v", got)
		}
	}
	check(false)
	replaceTest(t, s, "b", b)
	check(true)
	got, ok, err := s.LookupEntry(ctx, "b", "same")
	if err != nil || !ok || got.Entry.Term != "Beta" || len(got.Incoming) != 1 || got.Incoming[0].From.Source != "a" {
		t.Fatalf("collision/incoming: %+v %v", got, err)
	}
	hits, err := s.Search(ctx, "alias", "b", 20, 0)
	if err != nil || len(hits) != 1 || hits[0].Sense != "one" || hits[0].LexicalPronunciation != "spoken Beta" || hits[0].LexicalPartOfSpeech != "noun" {
		t.Fatalf("scoped metadata: %+v %v", hits, err)
	}
	empty := v.Dictionary{Version: "1", Name: "Empty", Entries: []v.Entry{}}
	replaceTest(t, s, "b", empty)
	check(false)
	if _, ok, err = s.LookupEntry(ctx, "b", "same"); err != nil || ok {
		t.Fatalf("empty lookup: %v %v", ok, err)
	}
	for _, table := range []string{"nodes", "relationships", "search"} {
		var n int
		if err = s.db.QueryRow(`SELECT count(*) FROM ` + table + ` WHERE source='b'`).Scan(&n); err != nil || n != 0 {
			t.Fatalf("empty %s: %d %v", table, n, err)
		}
	}
	replaceTest(t, s, "b", b)
	check(true)
	if err = s.Keep(ctx, []string{"a"}); err != nil {
		t.Fatal(err)
	}
	check(false)
	for _, table := range []string{"nodes", "relationships", "source_metadata", "snapshots", "search", "status"} {
		var n int
		if err = s.db.QueryRow(`SELECT count(*) FROM ` + table + ` WHERE source='b'`).Scan(&n); err != nil || n != 0 {
			t.Fatalf("retired %s: %d %v", table, n, err)
		}
	}
}

func TestMigrationsInitializeReopenAndRejectFuture(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "store.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	d := lookupDictionary("Cached")
	replaceTest(t, s, "a", d)
	at := time.Date(2025, 1, 2, 3, 4, 5, 6, time.UTC)
	st := Status{Source: "a", Failures: 3, LastSuccess: at, LastAttempt: at, Recent: []Outcome{{Result: "failed", Detail: "kept"}}}
	if err = s.SaveStatus(ctx, st); err != nil {
		t.Fatal(err)
	}
	before, err := s.Snapshots(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, ok, err := s.LookupEntry(ctx, "a", "same")
	if err != nil || !ok || !reflect.DeepEqual(got.Entry, d.Entries[0]) {
		t.Fatalf("reopen: %+v %v", got, err)
	}
	after, err := s.Snapshots(ctx)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("changed cache: %+v %v", after, err)
	}
	states, err := s.Statuses(ctx)
	if err != nil || len(states) != 1 || !reflect.DeepEqual(states[0], st) {
		t.Fatalf("changed status: %+v %v", states, err)
	}
	hits, err := s.Search(ctx, "alias", "a", 10, 0)
	if err != nil || len(hits) != 1 || hits[0].LexicalPartOfSpeech != "noun" {
		t.Fatalf("reopened search: %+v %v", hits, err)
	}
	changed, err := s.Replace(ctx, "a", d, "new-tag", "new-modified", "new-key")
	if err != nil || changed {
		t.Fatalf("unchanged replace: %v %v", changed, err)
	}
	after, err = s.Snapshots(ctx)
	if err != nil || !after[0].AcceptedAt.Equal(before[0].AcceptedAt) || after[0].FetchKey != "new-key" || after[0].ETag != "new-tag" {
		t.Fatalf("validator refresh: %+v %v", after, err)
	}
	future := filepath.Join(t.TempDir(), "future.db")
	db, err := sql.Open("sqlite", future)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`PRAGMA user_version=3`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if s, err := Open(future); err == nil {
		s.Close()
		t.Fatal("accepted newer schema")
	}
}

func TestInitialMigrationRollsBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conflict.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// A late conflict must roll back the earlier snapshot and FTS table creation.
	if _, err = db.Exec(`CREATE TABLE nodes(conflict TEXT)`); err != nil {
		t.Fatal(err)
	}
	if s, err := Open(path); err == nil {
		s.Close()
		t.Fatal("conflicting migration succeeded")
	}
	var version, n int
	if err = db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name IN ('snapshots','search','relationships','source_metadata')`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if version != 0 || n != 0 {
		t.Fatalf("partial migration: version=%d tables=%d", version, n)
	}
}

func TestReplacementRollbackAndConcurrentLookups(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	a, b := lookupDictionary("Alpha"), lookupDictionary("Beta")
	for _, d := range []*v.Dictionary{&a, &b} {
		d.Entries[0].Relationships = []v.Relationship{{Type: "relatedTo", Target: v.Reference{Entry: "same", Sense: "one"}}}
	}
	replaceTest(t, s, "a", a)
	before, err := s.Snapshots(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// Fail late, after the snapshot and FTS writes, to prove the whole operation rolls back.
	if _, err = s.db.Exec(`CREATE TRIGGER fail_metadata BEFORE INSERT ON source_metadata BEGIN SELECT RAISE(ABORT,'injected'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Replace(ctx, "a", b, "changed", "changed"); err == nil {
		t.Fatal("injected write did not fail")
	}
	after, err := s.Snapshots(ctx)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("snapshot changed on rollback: %v", err)
	}
	got, ok, err := s.LookupEntry(ctx, "a", "same")
	if err != nil || !ok || got.Entry.Term != "Alpha" || got.Outgoing[0].Term != "Alpha" {
		t.Fatalf("normalized rollback: %+v %v", got, err)
	}
	hits, err := s.Search(ctx, "", "a", 20, 0)
	if err != nil || len(hits) != 2 || hits[0].Term != "Alpha" {
		t.Fatalf("FTS rollback: %+v %v", hits, err)
	}
	if _, err = s.db.Exec(`DROP TRIGGER fail_metadata`); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Go(func() {
		for i := 0; i < 30; i++ {
			d := a
			if i%2 == 0 {
				d = b
			}
			if _, err := s.Replace(ctx, "a", d, "", ""); err != nil {
				t.Error(err)
				return
			}
		}
	})
	for i := 0; i < 50; i++ {
		got, ok, err := s.LookupEntry(ctx, "a", "same")
		if err != nil || !ok {
			t.Fatalf("read: %v %v", ok, err)
		}
		if len(got.Outgoing) != 1 || !got.Outgoing[0].Resolved || got.Entry.Term != got.Entry.Senses[0].Definition || got.Entry.Term != got.Outgoing[0].Term {
			t.Fatalf("mixed versions: %+v", got)
		}
		hits, err := s.Search(ctx, "", "a", 20, 0)
		if err != nil || len(hits) != 2 {
			t.Fatalf("search: %+v %v", hits, err)
		}
		for _, hit := range hits {
			if hit.LexicalPronunciation != "spoken "+hit.Term {
				t.Fatalf("mixed search metadata: %+v", hit)
			}
		}
	}
	wg.Wait()
}

func TestLookupIndependentOfSnapshotsAndIndexed(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	replaceTest(t, s, "a", lookupDictionary("Alpha"))
	// If any public query starts decoding snapshots again, these calls fail.
	if _, err := s.db.Exec(`UPDATE snapshots SET document='invalid JSON'`); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := s.LookupEntry(ctx, "a", "same"); err != nil || !ok {
		t.Fatalf("lookup read snapshot: %v", err)
	}
	if _, ok, err := s.LookupEntry(ctx, "a", "missing"); err != nil || ok {
		t.Fatalf("miss read snapshot: %v", err)
	}
	if _, err := s.Search(ctx, "Alpha", "", 10, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Sources(ctx); err != nil {
		t.Fatal(err)
	}
	for query, want := range map[string]string{
		`SELECT document FROM nodes WHERE source='a' AND entry='same' AND sense=''`:                          "PRIMARY KEY",
		`SELECT * FROM relationships WHERE source='a' AND entry='same'`:                                      "PRIMARY KEY",
		`SELECT * FROM relationships WHERE target_source='a' AND target_entry='same' AND target_sense='one'`: "relationship_targets",
	} {
		rows, err := s.db.Query(`EXPLAIN QUERY PLAN ` + query)
		if err != nil {
			t.Fatal(err)
		}
		var plans []string
		for rows.Next() {
			var id, parent, unused int
			var detail string
			if err = rows.Scan(&id, &parent, &unused, &detail); err != nil {
				t.Fatal(err)
			}
			plans = append(plans, detail)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			t.Fatal(err)
		}
		plan := strings.Join(plans, " ")
		if !strings.Contains(plan, "SEARCH") || !strings.Contains(plan, want) || strings.Contains(plan, "SCAN relationships") || strings.Contains(plan, "SCAN nodes") {
			t.Fatalf("unindexed query: %s", plan)
		}
	}
}

func TestReadTransactionKeepsPageContentConsistent(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "shared.db")
	first, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	replaceTest(t, first, "a", lookupDictionary("Before"))
	read, err := first.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer read.Close()
	sources, err := read.Sources(ctx)
	if err != nil || sources[0].Name != "Before" {
		t.Fatalf("initial read: %+v %v", sources, err)
	}
	// A separate connection can commit in WAL mode while this read snapshot is open.
	replaceTest(t, second, "a", lookupDictionary("After"))
	got, ok, err := read.LookupEntry(ctx, "a", "same")
	if err != nil || !ok || got.Entry.Term != "Before" {
		t.Fatalf("lookup changed mid-request: %+v %v", got, err)
	}
	hits, err := read.Search(ctx, "", "a", 20, 0)
	if err != nil || len(hits) != 2 || hits[0].Term != "Before" || hits[0].LexicalPronunciation != "spoken Before" {
		t.Fatalf("search changed mid-request: %+v %v", hits, err)
	}
	entries, err := read.Browse(ctx, "a")
	if err != nil || len(entries) != 1 || entries[0].Entry.Term != "Before" {
		t.Fatalf("browse changed mid-request: %+v %v", entries, err)
	}
	if err = read.Close(); err != nil {
		t.Fatal(err)
	}
	got, ok, err = first.LookupEntry(ctx, "a", "same")
	if err != nil || !ok || got.Entry.Term != "After" {
		t.Fatalf("new request missed replacement: %+v %v", got, err)
	}
}

func TestSenseLookupDoesNotDecodeOtherSenses(t *testing.T) {
	ctx := context.Background()
	s := openTestStore(t)
	replaceTest(t, s, "a", lookupDictionary("Alpha"))
	if _, err := s.db.Exec(`UPDATE nodes SET document='invalid JSON' WHERE sense='two'`); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := s.LookupEntry(ctx, "a", "same", "one"); err != nil || !ok {
		t.Fatalf("decoded another sense: %v", err)
	}
	if _, err := s.db.Exec(`DROP TABLE relationships`); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := s.LookupEntry(ctx, "a", "missing"); err != nil || ok {
		t.Fatalf("entry miss queried edges: %v", err)
	}
	if _, ok, err := s.LookupEntry(ctx, "a", "same", "missing"); err != nil || ok {
		t.Fatalf("sense miss queried edges: %v", err)
	}
}

func TestSourceLetterMetadataLifecycle(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "letters.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	d := v.Dictionary{Version: "1", Name: "Letters", Entries: []v.Entry{
		{ID: "apple", Term: "apple", Aliases: []v.Alias{{Text: "Zebra"}}, Senses: []v.Sense{{ID: "one", Definition: "First."}}},
		{ID: "another", Term: "Another", Senses: []v.Sense{{ID: "one", Definition: "Duplicate initial."}}},
		{ID: "digit", Term: "３D", Senses: []v.Sense{{ID: "one", Definition: "Numeric."}}},
		{ID: "symbol", Term: "Éclair", Senses: []v.Sense{{ID: "one", Definition: "Other initial."}}},
	}}
	if _, err := s.Replace(ctx, "letters", d, "", ""); err != nil {
		t.Fatal(err)
	}
	check := func(want string) {
		t.Helper()
		sources, err := s.Sources(ctx)
		if err != nil || len(sources) != 1 || sources[0].Letters == nil || strings.Join(sources[0].Letters, ",") != want {
			t.Fatalf("letters = %+v, error %v, want %q", sources, err, want)
		}
	}
	check("#,0-9,A")
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	// Recreate the previous metadata schema to exercise the one-time backfill.
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`ALTER TABLE source_metadata DROP COLUMN letters; DELETE FROM goose_db_version WHERE version_id=2; PRAGMA user_version=1;`)
	db.Close()
	if err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	check("#,0-9,A")
	d.Entries = d.Entries[:1]
	d.Entries[0].Term = "Banana"
	if _, err := s.Replace(ctx, "letters", d, "", ""); err != nil {
		t.Fatal(err)
	}
	check("B")
	if _, err := s.Replace(ctx, "letters", v.Dictionary{Version: "1", Name: "Empty", Entries: []v.Entry{}}, "", ""); err != nil {
		t.Fatal(err)
	}
	check("")
}
