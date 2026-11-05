---
title: Publish selected content
editUrl: https://github.com/alampros/jargonary/edit/main/docs/publishing.md
---

Named exports serve selected company content at `/.well-known/glossary/{exportId}`:

```yaml
exports:
  - id: partners
    name: Partner glossary
    include:
      sources: [payments]
      sense_tags:
        any: [partner]
    auth:
      shared_secret:
        file: /run/secrets/partners
```

This selects `partner`-tagged senses from `payments`. Every export requires a shared secret or `auth.public: true`. Secrets use [file/environment references](sources.md#protected-sources). Consumers configure the full HTTPS URL as a source.

## Selection and availability

- Exports preserve source identities, retain applicable aliases, and prune relationships to excluded targets while preserving effective deprecation.
- The bundled `project` source is excluded. Imported exports cannot be re-exported in v1.
- Exports return **503 until every selected source has an accepted snapshot**. Without a source allowlist, all configured sources must be ready, even with tag selection. Empty snapshots count; subsequent failures retain the last accepted content.
- Use source allowlists as well as tags to limit which owners can publish. Filtering does not redact prose, links, tags, or attribution.

Export authentication does not protect the UI or other routes. For export-only access, use the [proxy example](../examples/export-only-proxy.conf). Revocation cannot recall cached or downloaded content.

## Two-instance example

```sh
docker compose -f compose.federation.yaml up --build
```

Upstream: `http://localhost:8081/`; downstream: `http://localhost:8080/`. See [configs](../examples/federation/).

The root `/.well-known/glossary` serves the bundled project glossary, not company content. These paths are a proposed convention, not a registered standard or discovery mechanism.
