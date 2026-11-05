---
title: Glossary Format v1
editUrl: https://github.com/alampros/jargonary/edit/main/docs/dictionary-format.md
---

A glossary defines words in context. In the same company, **API** can mean an application programming interface to a developer and the American Petroleum Institute to an engineer. The format preserves those meanings and the context in which each applies.

Each source supplies one complete glossary in **JSON, YAML, or CSV**. Teams can maintain separate files or HTTP endpoints; Jargonary validates and combines their data without merging entries just because their names match.

## Create a Glossary

This is a complete YAML glossary:

```yaml
# yaml-language-server: $schema=https://alampros.github.io/jargonary/schema/__JARGONARY_VERSION__/glossary.schema.json
version: "1"
name: Web Development
entries:
  - id: api
    term: API
    aliases:
      - text: Application Programming Interface
        sense: programming-interface
    senses:
      - id: programming-interface
        definition: An interface through which software exposes operations and data to other software.
        context: Web development
        partOfSpeech: noun
        examples:
          - The checkout page calls the payments API.
```

The [Web Development](../examples/terms/web-development.yaml) and [Engineering](../examples/terms/engineering.yaml) examples define API, Component, Pipeline, and Port in their respective contexts. Load both to see how overlapping vocabulary stays distinct.

JSON uses the same field names and structure as YAML. CSV uses one row per meaning, as described in [Use CSV](#use-csv).

## Understand Entries, Senses, and Context

| Concept | Meaning |
| --- | --- |
| Glossary | A named collection of entries supplied by one source |
| Entry | A term with a stable ID and one or more meanings |
| Sense | One meaning, with its own definition and optional context |
| Context | Where that meaning applies: a team, discipline, application, or process |
| Alias | Another lookup name for an entry or a particular sense |

Use separate sources when teams own their glossaries independently. Each source can contain an `api` entry; Jargonary keeps their identities separate. A single owner can also put several senses in one entry when a term has multiple meanings within that glossary.

`context` is optional, but supplying it helps readers distinguish meanings. It is descriptive text; use `tags` for labels that participate in export selection.

## Define JSON and YAML Fields

### Glossary Fields

| Field | Required | Purpose |
| --- | --- | --- |
| `version` | Yes | Format version, currently the string `"1"` |
| `name` | Yes | Nonblank display name for the glossary |
| `entries` | Yes | Array of entries; `[]` deliberately clears the source |

A missing or `null` `entries` value is invalid. `exported` and entry-level `origin` are reserved for [export metadata](#consume-exported-glossaries); omit them from ordinary authored glossaries.

### Entry Fields

| Field | Required | Purpose |
| --- | --- | --- |
| `id` | Yes | Stable ID, unique within this source |
| `term` | Yes | Nonblank preferred display name |
| `senses` | Yes | Nonempty array of meanings |
| `pronunciation` | No | Authored pronunciation, such as IPA or a plain-English respelling |
| `aliases` | No | Array of objects with `text` and optional `sense` |
| `relationships` | No | Links to other entries or senses |
| `deprecated` | No | Marks the whole entry as no longer recommended |
| `deprecationReason` | No | Explanation for readers |

An alias requires nonblank `text`. If `sense` is set, it must identify a sense in this entry. Otherwise, the alias applies to all of the entry's senses. An alias has no separate definition; create another entry when it needs one.

### Sense Fields

| Field | Required | Purpose |
| --- | --- | --- |
| `id` | Yes | Stable ID, unique within its entry |
| `definition` | Yes | Nonblank explanation of this meaning |
| `context` | No | Where the meaning applies |
| `partOfSpeech` | No | Authored text such as `noun`, `verb`, or `adjective` |
| `examples` | No | Array of usage examples as strings |
| `tags` | No | Array of case-sensitive labels, such as `public` |
| `links` | No | Array of objects with `label` and `url` |
| `relationships` | No | Links from this meaning to other entries or senses |
| `deprecated` | No | Marks this meaning as no longer recommended |
| `deprecationReason` | No | Explanation for readers |

Link URLs must use HTTP or HTTPS and contain no embedded credentials. Authored text is displayed as text, not HTML. Pronunciation and part of speech are supplied by authors; Jargonary does not infer them or generate audio.

### Stable IDs

IDs contain 1–200 ASCII letters, digits, dots, underscores, or hyphens, starting with a letter or digit. They are case-sensitive. This rule applies to entry IDs, sense IDs, and source IDs in references.

Keep IDs unchanged when correcting a definition or renaming a displayed term. The source ID comes from application configuration, not the glossary document; coordinate it with the operator when linking across sources. Reusing an entry ID in a different source is valid and creates a distinct identity.

Terms and aliases can contain Unicode and do not need to be unique. Search normalizes text using NFC and Unicode case folding; it does not change IDs.

## Connect Related Terms

Relationships can originate on an entry or a sense. Each has a `type` and a `target`:

| Target field | Required | Meaning |
| --- | --- | --- |
| `entry` | Yes | Target entry ID |
| `sense` | No | Target sense ID; omit to target the whole entry |
| `source` | No | Configured source ID; omit to target the current source |

For example, an abbreviation can have its own definition and link to its expanded term:

```yaml
# yaml-language-server: $schema=https://alampros.github.io/jargonary/schema/__JARGONARY_VERSION__/glossary.schema.json
version: "1"
name: Web Development
entries:
  - id: api
    term: API
    senses:
      - id: abbreviation
        definition: Short for application programming interface.
        context: Web development
        relationships:
          - type: abbreviationOf
            target:
              entry: application-programming-interface
              sense: software
  - id: application-programming-interface
    term: Application Programming Interface
    senses:
      - id: software
        definition: An interface through which software exposes operations and data to other software.
        context: Web development
```

Supported relationships are:

| Authored type | Meaning | Derived API inverse |
| --- | --- | --- |
| `abbreviationOf` | A shortened form of the target | `abbreviatedBy` |
| `relatedTo` | Related concepts, without asserting identical meanings | `relatedTo` |
| `deprecatedFor` | An old term or meaning with a replacement | `supersedes` |

Author only the forward types. Jargonary generates and deduplicates reverse relationships and provides API links to resolved targets. Cross-source targets use the same structure with `source` added. Unresolved targets generate operator diagnostics while the originating definitions remain available.

### Deprecation

`deprecated: true` keeps a term discoverable while marking it as no longer recommended. Entry-level deprecation applies to all its senses. A `deprecatedFor` relationship also deprecates its origin, even if the replacement is unresolved; its inverse does not deprecate the replacement.

To reactivate a meaning, remove its explicit deprecation flag and outgoing replacement relationships, and ensure its parent entry is not deprecated. Cycles and self-replacements generate diagnostics rather than redirects.

## Use CSV

CSV uses UTF-8, a header, and one row per sense. Rows with the same `entry_id` are grouped into one entry:

```csv
entry_id,term,sense_id,definition,context
api,API,software,An application programming interface.,Web development
api,API,institute,The American Petroleum Institute which publishes industry standards.,Mechanical and petroleum engineering
```

The required columns are `entry_id`, `term`, `sense_id`, and `definition`. Repeated rows must agree on the entry's `term` and `pronunciation`, and each sense ID must be unique within its entry.

| Optional column | Cell format |
| --- | --- |
| `pronunciation`, `part_of_speech`, `context`, `deprecation_reason` | Plain text |
| `deprecated` | Empty, `false`, or `true`; applies to the row's sense |
| `aliases` | JSON array of strings; each alias is scoped to the row's sense |
| `tags`, `examples` | JSON array of strings |
| `links`, `relationships` | JSON array of objects using the JSON/YAML field names |

Quote complex cells using normal CSV escaping. For example, the cell containing `["public","engineering"]` is written as `"[""public"",""engineering""]"`.

Unknown or duplicate headers are rejected. A header without data rows represents an empty glossary. The parser assigns CSV input the display name `CSV dictionary`; CSV has no glossary-name column. Use JSON or YAML for a custom glossary name, unrestricted aliases, entry-level deprecation/relationships, or export provenance. There is no automatic ID generation, delimiter guessing, or spreadsheet formula evaluation.

## Publish Files or HTTP Endpoints

A source contains a complete glossary, not a manifest of other source URLs. Mount files locally or serve them at explicitly configured HTTP(S) URLs. `/.well-known/glossary` is the proposed conventional publishing path; any configured URL works. The path is not a registered standard, and it does not enable automatic network discovery.

In Jargonary, the root `/.well-known/glossary` serves the bundled project glossary when enabled. Named exports at `/.well-known/glossary/{exportId}` publish selected company content. The bundled project source is excluded from named exports.

Every accepted snapshot replaces all of that source's previous content atomically. A valid empty glossary clears it; failed fetches or invalid documents preserve the last successful snapshot. Removing a term from a valid replacement removes it on the next successful scrape. See the [publication guide](publishing.md) for source retirement, authentication, and export selectors.

## Consume Exported Glossaries

Named exports use the same JSON format with generated provenance metadata:

- The glossary has `exported: true`.
- Every entry has `origin` containing original `source` and `entry` IDs, without a `sense` field.
- Entry IDs are `x-` followed by the hexadecimal SHA-256 digest of `sourceID + NUL + entryID`. Sense IDs remain unchanged.

Export references are rewritten to those generated entry IDs, with `source` omitted so references resolve inside the consumer's configured source. Renaming a term, glossary, export, or consumer source does not change the generated entry ID; changing the publisher's source ID does. Independent publishers do not have separate identity namespaces in this format; configure distinct consumer sources to keep their imports separate.

Exports select senses before serialization, retain applicable aliases, and prune relationships to excluded or unresolved targets. Deprecation implied by a removed replacement relationship is preserved as an explicit flag. Consumers rebuild reverse relationships.

In v1, an export selecting any entries from an imported export is unavailable: automatic re-export is not supported. This is an application rule, not cryptographic proof of origin. Revoking fetch access does not erase previously accepted snapshots or downloaded copies.

Filtering excludes unselected records; it does not classify or redact confidential prose, links, labels, or attribution. Publishers are responsible for the content they select.

## Validate a Glossary

Use the [JSON Schema](../docs-site/public/schema/__JARGONARY_VERSION__/glossary.schema.json) for structural checks and editor assistance with JSON or YAML. The published [schema for application __JARGONARY_VERSION__](https://alampros.github.io/jargonary/schema/__JARGONARY_VERSION__/glossary.schema.json) provides a versioned editor URL when Pages is deployed. Format version `1` is separate from the application's release version.

Run the CLI validator before publishing a file:

```sh
go run ./cmd/jargonary -validate examples/terms/web-development.yaml -format yaml
```

An installed binary also supports `jargonary -validate path/to/glossary.json -format json`. Choose `json`, `yaml`, or `csv` to match the input.

Application validation checks rules beyond JSON Schema, including ID uniqueness, alias sense targets, origin metadata, and link URLs. JSON and YAML reject unknown fields and trailing documents; JSON also rejects duplicate keys. Relationship target IDs are validated, but a missing target is a runtime diagnostic rather than a parse failure.

For contributors, `go tool task schema VERSION=__JARGONARY_VERSION__` regenerates the schema from the Go model. Custom relationship types, runtime configuration reload, multi-hop re-export, localization, history/tombstones, and semantic search are not supported in v1.
