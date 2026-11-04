-- name: InsertNode :exec
INSERT INTO nodes(source,entry,sense,ordinal,document,term,pronunciation,part_of_speech)
VALUES(?,?,?,?,?,?,?,?);

-- name: InsertRelationship :exec
INSERT INTO relationships(source,entry,sense,ordinal,type,target_source,target_entry,target_sense)
VALUES(?,?,?,?,?,?,?,?);

-- name: InsertSourceMetadata :exec
INSERT INTO source_metadata(source,name,entries,senses,letters) VALUES(?,?,?,?,?);

-- name: DeleteNodes :exec
DELETE FROM nodes WHERE source=?;

-- name: DeleteRelationships :exec
DELETE FROM relationships WHERE source=?;

-- name: DeleteSourceMetadata :exec
DELETE FROM source_metadata WHERE source=?;

-- name: ListSources :many
SELECT m.source,m.name,m.entries,m.senses,m.letters,s.accepted_at
FROM source_metadata m JOIN snapshots s ON s.source=m.source ORDER BY m.source;

-- name: GetNode :one
SELECT document FROM nodes WHERE source=? AND entry=? AND sense=?;

-- name: ListSenses :many
SELECT document FROM nodes WHERE source=? AND entry=? AND sense<>'' ORDER BY ordinal;

-- name: OutgoingRelationships :many
SELECT r.source,r.entry,r.sense,r.type,r.target_source,r.target_entry,r.target_sense,
       CAST(coalesce(n.term,'') AS TEXT) AS term, CAST(n.source IS NOT NULL AS INTEGER) AS resolved
FROM relationships r
LEFT JOIN nodes n ON n.source=r.target_source AND n.entry=r.target_entry AND n.sense=r.target_sense
WHERE r.source=? AND r.entry=? AND (r.sense=sqlc.arg(selected_sense) OR r.sense='' OR sqlc.arg(selected_sense)='')
ORDER BY r.sense,r.ordinal;

-- name: IncomingRelationships :many
SELECT r.source,r.entry,r.sense,r.type,r.target_source,r.target_entry,r.target_sense,n.term
FROM relationships r
JOIN nodes n ON n.source=r.source AND n.entry=r.entry AND n.sense=r.sense
JOIN nodes target ON target.source=r.target_source AND target.entry=r.target_entry AND target.sense=r.target_sense
WHERE r.target_source=? AND r.target_entry=? AND (r.target_sense=sqlc.arg(selected_sense) OR r.target_sense='' OR sqlc.arg(selected_sense)='')
ORDER BY r.source,r.entry,r.sense,r.ordinal;

-- name: BrowseNodes :many
SELECT source,entry,sense,document FROM nodes
WHERE (source=sqlc.arg(selected_source) OR sqlc.arg(selected_source)='')
ORDER BY source,entry,CASE WHEN sense='' THEN -1 ELSE ordinal END;

-- name: ListEntryTerms :many
SELECT DISTINCT term FROM nodes
WHERE sense='' AND (source=sqlc.arg(selected_source) OR sqlc.arg(selected_source)='');

-- name: UpdateSourceLetters :exec
UPDATE source_metadata SET letters=? WHERE source=?;
