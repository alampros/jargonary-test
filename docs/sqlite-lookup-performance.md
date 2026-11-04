# SQLite lookup performance

Entry and sense API requests now query the composite node primary key, load only the requested entry/senses, and resolve incident relationships using the origin primary key and target index. Missing identities return before any relationship query. Search metadata comes from a join to the matching sense node in the same SQL statement. Public pages keep content queries in one SQLite read transaction, including an inline search definition and its relationship links.

`TestLookupIndependentOfSnapshotsAndIndexed` corrupts canonical snapshots and verifies lookups, misses, search, and source lists still work. It checks SQLite query plans for identity, outgoing-edge, and incoming-edge indexes. `TestSenseLookupDoesNotDecodeOtherSenses` corrupts a sibling sense and removes the relationship table to verify selected-sense isolation and the miss path. `TestReadTransactionKeepsPageContentConsistent` replaces content on another WAL connection between page reads and verifies that all page content stays on its original snapshot.

Run the reproducible benchmark from the repository root:

```sh
go test ./internal/server -run '^$' -bench BenchmarkEntryResolution -benchmem -benchtime=150ms -count=1
```

The test-only `snapshotEntry` preserves the previous resolver. Each baseline iteration reads and decodes canonical snapshots and builds the full graph, including for a miss. Indexed iterations exercise the new store queries and view construction. The HTTP benchmark exercises the new public handler, including its error response. Database setup and insertion are outside the measured loops. The fixture has one source, either 1 or 4,000 entries, one sense per entry, and no relationships. The compatibility test separately covers dense authored and inverse relationships.

Representative cumulative allocation measurements on Apple M4, Go 1.27.3:

| Operation | 1 entry: bytes/op | 4,000 entries: bytes/op | 4,000 entries: allocations/op |
| --- | ---: | ---: | ---: |
| Previous resolver, hit | 4,714 | 31,303,901 | 388,771 |
| Indexed resolver, hit | 8,163 | 8,087 | 169 |
| Previous resolver, miss | 4,284 | 31,478,811 | 392,091 |
| Indexed resolver, miss | 1,722 | 1,712 | 38 |
| Indexed resolver, missing sense | 1,728 | 1,728 | 39 |
| Public HTTP 404 | 4,185 | 4,183 | 67 |

These are allocation measurements, not retained memory or evidence of an OOM. The fixture differs from the original review probe; the comparison above runs both resolvers against the same generated data. Fixed SQL/transaction overhead makes successful indexed lookups more expensive for a single-entry installation. Allocation growth from unrelated entries has been removed; no wall-clock threshold is asserted.

Lookup work still scales with the requested entry's senses and incident relationships, and incoming edges need deterministic sorting before inverse deduplication. A highly connected entry can have a large response. Aliases remain in each entry document and in the existing per-sense FTS alias fields, preserving scoped matching.

The full browsing view still loads all selected entries before grouping, sorting, and paginating. HTML search still groups all matching senses before entry pagination. FTS exact/alias/prefix ranking retains its existing query shape; this change removes snapshot decoding for metadata rather than redesigning search ranking. Exports still decode complete canonical snapshots. The opt-in operator diagnostics API and page still decode snapshots and build the installation-wide graph; replacement-cycle checks traverse that graph from each node. Those diagnostic costs remain and are kept off the public listener.
