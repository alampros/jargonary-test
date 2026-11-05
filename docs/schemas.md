---
title: JSON Schema
editUrl: https://github.com/alampros/jargonary/edit/main/docs/schemas.md
---

Use Jargonary's JSON Schemas for structural validation and editor assistance with glossary files and application configuration. Application validation also checks semantic constraints such as relationship targets.

## Published schemas

- [Configuration schema for __JARGONARY_VERSION__](../docs-site/public/schema/__JARGONARY_VERSION__/config.schema.json)
- [Glossary schema for __JARGONARY_VERSION__](../docs-site/public/schema/__JARGONARY_VERSION__/glossary.schema.json)

Schemas are generated from the Go types and checked in under `docs-site/public/schema/__JARGONARY_VERSION__/`. Astro serves these files directly. Their `$id` URLs include the application release version, currently `__JARGONARY_VERSION__`. Keep existing release paths when adding schemas for a new release. The glossary document format remains version `1`; it is separate from the application release version.

`$schema` identifies the JSON Schema specification (currently draft 2020-12), not the Jargonary application version.

## Configuration schema

The [configuration schema](../docs-site/public/schema/__JARGONARY_VERSION__/config.schema.json) describes the YAML startup configuration. It covers optional settings, string durations, sources, authentication references, and export selectors, and rejects unknown fields. Top-level settings are optional because the application supplies defaults.

Add this directive to your config file:

```yaml
# yaml-language-server: $schema=https://alampros.github.io/jargonary/schema/__JARGONARY_VERSION__/config.schema.json
site_name: Company glossary
sources:
  - id: platform
    file: terms/platform.yaml
```

Startup validation also checks duration ranges, trimmed name length, URL and listener validity, duplicate IDs, source references, local files, and secret values. Editor validation does not read files or resolve environment variables. Use `jargonary -config config.yaml` to load and validate configuration at startup.

From a source checkout, `go tool task schema VERSION=__JARGONARY_VERSION__` regenerates both schemas in `docs-site/public/schema/__JARGONARY_VERSION__/`. The task requires `VERSION` and uses it for both the output directory and schema IDs. Both CLI schema flags require `-schema-version`; missing or invalid versions fail explicitly. Commit the generated JSON when the Go types change. Run `go run ./cmd/jargonary -config-schema -schema-version __JARGONARY_VERSION__` to print just the configuration schema.

## Editor integration

For YAML, editors with YAML language-server support can use this directive:

```yaml
# yaml-language-server: $schema=https://alampros.github.io/jargonary/schema/__JARGONARY_VERSION__/glossary.schema.json
version: "1"
name: Payments
entries: []
```

For JSON, configure your editor's schema association for your glossary files using the same URL.

## Validate with Jargonary

The application's validator checks both structure and semantics:

```sh
./jargonary -validate terms/payments.yaml -format yaml
```

From a source checkout:

```sh
go run ./cmd/jargonary -validate examples/terms/web-development.yaml -format yaml
```

See the [format guide](dictionary-format.md) for the complete contract.
