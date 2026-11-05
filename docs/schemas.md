---
title: JSON Schema
editUrl: https://github.com/alampros/jargonary/edit/main/docs/schemas.md
---

JSON Schema checks structure; Jargonary also validates semantics, local files, and secrets. Editor validation cannot replace application validation.

## Published schemas

- [Configuration](../docs-site/public/schema/__JARGONARY_VERSION__/config.schema.json)
- [Glossary](../docs-site/public/schema/__JARGONARY_VERSION__/glossary.schema.json)

URLs are versioned by application release (`__JARGONARY_VERSION__`). The glossary's `version: "1"` is the document format; `$schema` identifies the JSON Schema specification (draft 2020-12).

## Configuration schema

```yaml
# yaml-language-server: $schema=https://alampros.github.io/jargonary/schema/__JARGONARY_VERSION__/config.schema.json
site_name: Company glossary
```

Settings are optional because defaults exist; unknown fields are rejected. `jargonary -config config.yaml` performs startup validation, including file and environment references.

## Editor integration

For glossaries, use:

```yaml
# yaml-language-server: $schema=https://alampros.github.io/jargonary/schema/__JARGONARY_VERSION__/glossary.schema.json
version: "1"
name: Payments
entries: []
```

For JSON, associate the same URL in your editor's settings.

## Application validation

```sh
./jargonary -validate terms/payments.yaml -format yaml
```

Formats: `json`, `yaml`. See [the format contract](dictionary-format.md) and [schema regeneration](development.md#schemas-and-docs).
