---
title: Glossary Format v1
editUrl: https://github.com/alampros/jargonary/edit/main/docs/dictionary-format.md
---

Each source supplies a complete glossary in JSON or YAML. Entries with matching names or IDs in different sources remain distinct.

:::tip[Glossary editor]
Try the [Glossary Editor](https://alampros.github.io/jargonary/editor) to create or edit glossary files entirely in your browser. Files stay on your device.
:::

## Create a Glossary

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

JSON uses the same structure as YAML. An entry groups a term's meanings under one owner; use separate sources for independent owners. `context` is descriptive text; use `tags` for export selection.

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

Aliases require nonblank `text`; optional `sense` must identify a sense in the same entry. Without it, the alias applies to all senses. Aliases have no definitions of their own.

### Sense Fields

| Field | Required | Purpose |
| --- | --- | --- |
| `id` | Yes | Stable ID, unique within its entry |
| `definition` | Yes | Nonblank explanation of this meaning |
| `context` | No | Where the meaning applies |
| `partOfSpeech` | No | One of `noun`, `pronoun`, `verb`, `adjective`, `adverb`, `preposition`, `conjunction`, `interjection`, or `determiner` |
| `examples` | No | Array of usage examples as strings |
| `tags` | No | Array of case-sensitive labels, such as `public` |
| `links` | No | Array of objects with `label` and `url` |
| `relationships` | No | Links from this meaning to other entries or senses |
| `deprecated` | No | Marks this meaning as no longer recommended |
| `deprecationReason` | No | Explanation for readers |

Part-of-speech values are case-sensitive. Omit `partOfSpeech` when unspecified; unknown values (including `noun phrase` or `Noun`) are rejected. The editor’s **Not specified** choice omits this field.

Links require HTTP(S) URLs without embedded credentials. Authored text is not HTML; pronunciation and part of speech are not inferred, and audio is not generated.

### Stable IDs

Entry, sense, and referenced source IDs are case-sensitive: 1–200 ASCII letters, digits, dots, underscores, or hyphens, starting with a letter or digit.

Keep IDs stable when renaming terms or correcting definitions. Source IDs come from application configuration, not glossary documents. Identical entry IDs in different sources are separate identities.

Terms and aliases can contain Unicode and do not need to be unique. Search normalizes text using NFC and Unicode case folding; it does not change IDs.

## Connect Related Terms

Relationships on entries or senses contain a `type` and `target`:

| Target field | Required | Meaning |
| --- | --- | --- |
| `entry` | Yes | Target entry ID |
| `sense` | No | Target sense ID; omit to target the whole entry |
| `source` | No | Configured source ID; omit to target the current source |

For an abbreviation linking to another entry's sense:

```yaml
relationships:
  - type: abbreviationOf
    target:
      entry: application-programming-interface
      sense: software
```

Supported relationships are:

| Authored type | Meaning | Derived API inverse |
| --- | --- | --- |
| `abbreviationOf` | A shortened form of the target | `abbreviatedBy` |
| `relatedTo` | Related concepts, without asserting identical meanings | `relatedTo` |
| `deprecatedFor` | An old term or meaning with a replacement | `supersedes` |

Author only forward types; inverses are generated and deduplicated. Unresolved targets produce operator diagnostics without hiding definitions.

### Deprecation

`deprecated: true` preserves discoverability; entry-level flags apply to all senses. `deprecatedFor` also deprecates its origin, even with an unresolved target; its inverse does not deprecate the replacement.

Reactivation requires removing explicit flags and outgoing replacements, including parent-entry deprecation. Cycles and self-replacements produce diagnostics, not redirects.

## Publish Files or HTTP Endpoints

Sources are complete snapshots, not manifests of source URLs. Each accepted snapshot atomically replaces the previous one, including removals; empty glossaries clear the source. Invalid input or failed fetches preserve the last snapshot.

See [source settings](sources.md) for files, HTTP, and discovery, and [publishing](publishing.md) for named exports. The proposed `/.well-known/glossary` convention is not a registered standard or automatic discovery mechanism. Its root serves the bundled project glossary; named exports serve company content and exclude the project source.

## Consume Exported Glossaries

Named exports use this format with generated provenance:

- The glossary has `exported: true`.
- Every entry has `origin` containing original `source` and `entry` IDs, without a `sense` field.
- Entry IDs are `x-` followed by the hexadecimal SHA-256 digest of `sourceID + NUL + entryID`. Sense IDs remain unchanged.

References use generated entry IDs and omit `source`, resolving inside the consumer source. Only original source/entry IDs affect generated identity, not display names, export IDs, or consumer source IDs. Publisher identity is not namespaced; import independent publishers as distinct consumer sources.

Exports retain selected senses and applicable aliases, prune excluded/unresolved targets, and preserve implied deprecation as explicit flags. Consumers rebuild inverses.

Exports selecting imported entries are unavailable in v1. Provenance is application metadata, not cryptographic proof.

## Validate a Glossary

Use [JSON Schema](schemas.md) for editor checks, then validate the complete document:

```sh
jargonary -validate terms/payments.yaml -format yaml
```

Formats: `json`, `yaml`. Application validation adds ID uniqueness, alias targets, provenance, and URL checks. JSON/YAML reject unknown fields and trailing documents; JSON also rejects duplicate keys. Missing relationship targets are runtime diagnostics, not parse failures.

V1 does not support custom relationship types, runtime configuration reload, multi-hop re-export, localization, history/tombstones, or semantic search.
