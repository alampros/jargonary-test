-- +goose Up
-- Empty string marks metadata from before letter sets were cached. Startup
-- fills those once; a valid empty glossary stores the JSON array [].
ALTER TABLE source_metadata ADD COLUMN letters TEXT NOT NULL DEFAULT '';
PRAGMA user_version=2;
