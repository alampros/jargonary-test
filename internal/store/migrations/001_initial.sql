-- +goose Up
CREATE TABLE snapshots(source TEXT PRIMARY KEY, document BLOB NOT NULL, accepted_at TEXT NOT NULL, etag TEXT NOT NULL, modified TEXT NOT NULL, fetch_key TEXT NOT NULL);
CREATE TABLE status(source TEXT PRIMARY KEY, document BLOB NOT NULL);
CREATE VIRTUAL TABLE search USING fts5(source UNINDEXED, entry UNINDEXED, sense UNINDEXED, term, aliases, definition, context, normalized UNINDEXED, alias_keys UNINDEXED, deprecated UNINDEXED, tokenize='unicode61', prefix='2 3 4');

-- Empty sense identifies an entry header. JSON records preserve portable fields;
-- indexed columns support lookup and search metadata without decoding snapshots.
CREATE TABLE nodes (
    source TEXT NOT NULL,
    entry TEXT NOT NULL,
    sense TEXT NOT NULL,
    ordinal INTEGER NOT NULL,
    document BLOB NOT NULL,
    term TEXT NOT NULL,
    pronunciation TEXT NOT NULL,
    part_of_speech TEXT NOT NULL,
    PRIMARY KEY (source, entry, sense)
) WITHOUT ROWID;

-- Keep authored edges, including unresolved targets and authored duplicates.
-- There is deliberately no foreign key requiring the target to exist.
CREATE TABLE relationships (
    source TEXT NOT NULL,
    entry TEXT NOT NULL,
    sense TEXT NOT NULL,
    ordinal INTEGER NOT NULL,
    type TEXT NOT NULL,
    target_source TEXT NOT NULL,
    target_entry TEXT NOT NULL,
    target_sense TEXT NOT NULL,
    PRIMARY KEY (source, entry, sense, ordinal)
) WITHOUT ROWID;
CREATE INDEX relationship_targets ON relationships (target_source, target_entry, target_sense);

CREATE TABLE source_metadata (
    source TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    entries INTEGER NOT NULL,
    senses INTEGER NOT NULL
);

PRAGMA user_version=1;
