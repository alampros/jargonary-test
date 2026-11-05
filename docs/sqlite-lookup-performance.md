---
title: SQLite lookup performance
editUrl: https://github.com/alampros/jargonary/edit/main/docs/sqlite-lookup-performance.md
---

Entry/sense lookups use composite identity indexes and incident-edge indexes rather than decoding every source snapshot. Public page reads share one SQLite transaction so concurrent scrapes cannot mix content versions.

## Reproduce

```sh
go test ./internal/server -run '^$' -bench BenchmarkEntryResolution -benchmem -benchtime=150ms -count=1
```

The [benchmark](../internal/server/lookup_test.go) compares the previous snapshot resolver with indexed lookup. Setup is excluded; fixtures contain one source, 1 or 4,000 entries, one sense each, and no relationships.

Representative allocations on Apple M4, Go 1.27.3:

| Operation | 1 entry: bytes/op | 4,000 entries: bytes/op |
| --- | ---: | ---: |
| Previous resolver, hit | 4,714 | 31,303,901 |
| Indexed resolver, hit | 8,163 | 8,087 |
| Previous resolver, miss | 4,284 | 31,478,811 |
| Indexed resolver, miss | 1,722 | 1,712 |
| Indexed resolver, missing sense | 1,728 | 1,728 |
| Public HTTP 404 | 4,185 | 4,183 |

These measure cumulative allocation, not retained memory or timing. Fixed SQL overhead costs more for tiny glossaries; unrelated entries no longer increase lookup allocations.

## Remaining costs

Lookup work scales with the requested senses and incident relationships; incoming edges also need sorting. Browsing and HTML search still group all matching entries/senses before pagination. Exports decode full snapshots. Operator diagnostics build the installation-wide graph, including replacement-cycle checks, and stay off the public listener.
