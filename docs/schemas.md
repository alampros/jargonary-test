# JSON Schema

Use Jargonary's JSON Schema for structural validation and editor assistance with JSON and YAML glossaries. Application validation also checks semantic constraints such as relationship targets.

## Published schemas

- [Latest stable schema](../schema/glossary.schema.json): follows the newest stable application release; uses the checked-in schema before the first stable release.
- [Glossary format v1](../schema/v1/glossary.schema.json): the newest published schema for format version `1`.

For a particular stable application release, use `schema/vX.Y.Z/glossary.schema.json` beneath the documentation site's base URL, replacing `vX.Y.Z` with a published release tag. Release-specific schemas are read from that tag's committed schema.

Format versions and application release versions are separate. Use the format URL for editor integration, or pin a release URL when you need its exact schema.

## Editor integration

For YAML, editors with YAML language-server support can use this directive:

```yaml
# yaml-language-server: $schema=https://alampros.github.io/jargonary/schema/v1/glossary.schema.json
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
