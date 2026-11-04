# Dictionary format v1 (initial implementation)

The application reads one complete JSON, YAML, or CSV dictionary per configured source. The proposed conventional HTTP path is `/.well-known/glossary`; any explicitly configured URL works. The suffix has not been registered as a standard.

## JSON and YAML

[schema/glossary.schema.json](../schema/glossary.schema.json) is generated with `go tool task schema`. After Pages is enabled, the published [format v1 schema](https://alampros.github.io/jargonary/schema/v1/glossary.schema.json) can be used as an editor schema URL. YAML uses exactly the same field names and structure as JSON. Both reject unknown fields and trailing documents. Required top-level fields are `version: "1"`, a nonempty `name`, and an `entries` array. `entries: []` deliberately clears a source; a missing/null array is invalid.

Entries require `id`, `term`, and nonempty `senses`. Senses require `id` and `definition`. Entry IDs are unique per source; sense IDs are unique per entry. IDs contain 1–200 ASCII letters, digits, dots, underscores, or hyphens, starting with a letter/digit. They are case-sensitive. Display names and aliases can contain Unicode and need not be unique. Exact search normalizes NFC and uses Unicode case folding; IDs are never normalized.

Optional entry fields: `aliases`, `deprecated`, `deprecationReason`, and `relationships`. Each alias has `text` and an optional `sense` identifying one of that entry's senses. An unrestricted alias applies to all senses.

Optional sense fields: `context`, `examples` (strings), `tags` (strings), `links` (`label`/`url` objects), `deprecated`, `deprecationReason`, and `relationships`. Links must be HTTP(S) URLs without embedded credentials. Authored text is rendered as text, not HTML. Publication tags are ordinary case-sensitive labels; configuring a selector gives them publication significance.

A relationship is `{ "type": "abbreviationOf", "target": { "source": "payments", "entry": "payment-intent", "sense": "payment-attempt" } }`. Source and sense are optional; omitted source refers to the current configured source. Sources are configured independently, so authors must coordinate source IDs for cross-source references. Unresolved targets generate diagnostics while definitions remain available.

Publish only these forward relationship types:

| Source relation | Derived API inverse |
| --- | --- |
| `abbreviationOf` | `abbreviatedBy` |
| `relatedTo` | `relatedTo` |
| `deprecatedFor` | `supersedes` |

Reverse relations are generated, deduplicated, and exposed as typed relationship metadata plus HATEOAS `_links`. Do not declare inverse-only types in source files. Relationships can connect entries, senses, or one of each. A sense's `entry` link leads to its parent; resolved relation links lead to the exact target. Unresolved metadata is kept in ordinary API views but omitted entirely from filtered exports.

`deprecated: true` marks an entry or sense no longer recommended without deleting it. Entry-level status affects all senses. An outgoing `deprecatedFor` also marks its origin deprecated, with a navigable replacement when resolved. The inverse does not deprecate the new term. Removing the explicit flag and outgoing replacement relation reactivates content unless its parent entry is deprecated. Cycles/self-replacements generate diagnostics, not redirects.

## CSV

CSV is UTF-8 with a header and one row per sense. Required columns:

| Column | Meaning |
| --- | --- |
| `entry_id` | Stable entry ID; repeated rows join this entry |
| `term` | Preferred name; must agree across rows of an entry |
| `sense_id` | Stable ID unique within the entry |
| `definition` | Definition of this sense |

Optional columns: `context`, `aliases`, `tags`, `examples`, `links`, `relationships`, `deprecated`, `deprecation_reason`. Unknown/duplicate headers are rejected. `deprecated` is empty, `false`, or `true`, and applies to the row's sense. JSON/YAML support entry-level fields that CSV does not express directly.

Complex cells use JSON arrays, quoted according to CSV rules:

- `aliases`: array of strings scoped to the row's sense.
- `tags` and `examples`: arrays of strings.
- `links` and `relationships`: arrays of the same objects used in JSON/YAML.

An otherwise valid header without rows represents an empty dictionary named `CSV dictionary`. There is no delimiter guessing, automatic ID generation, or spreadsheet formula evaluation. Use JSON/YAML when you need unrestricted aliases or entry-level metadata.

## Export round trips

The HTTP export format is the same JSON dictionary, with `exported: true` and an `origin` on each entry containing audience-safe original `source` and `entry` IDs. The publisher assigns entry IDs as `x-` plus SHA-256 hex of `sourceID + NUL + entryID`. Sense IDs are unchanged. Original source IDs must be suitable for publication; original target URLs are never copied into origin metadata.

This keeps same-named IDs from different sources distinct. Export references are rewritten to the exported IDs, with omitted source so they resolve within whatever local source ID the consumer chooses. Changing a source ID changes identity; changing a term, dictionary name, export name, or consumer source ID does not change the exported entry ID. Separate publisher identity namespaces and duplicate import-path merging are not supported; use distinct configured sources to retain separate imports.

Selected senses, surviving aliases, and relationships form a complete snapshot. Hidden/unresolved relationship targets are pruned, and deprecation implied by a pruned replacement is materialized as an explicit flag. Reverse edges are rebuilt by the consumer. Structural filtering does not inspect the confidentiality of prose or manually authored links.

An export which selects any entries from a dictionary marked `exported` returns unavailable rather than creating another hop. This is the v1 re-export restriction, not a cryptographic provenance guarantee: trusted source authors can always republish their own content manually.

## Deferred contracts

Custom relation types, runtime configuration reload, multi-hop re-export, content localization, tombstones/history, and semantic search are not implemented. The schema is an initial project contract rather than an independent registered specification. JSON Schema covers structure; the CLI's semantic validator is authoritative for uniqueness, alias resolution, origin rules, and URL restrictions.
