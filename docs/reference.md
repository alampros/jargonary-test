---
title: Configuration and deployment
editUrl: https://github.com/alampros/jargonary/edit/main/docs/reference.md
---

## Configuration

```yaml
# yaml-language-server: $schema=https://alampros.github.io/jargonary/schema/__JARGONARY_VERSION__/config.schema.json
site_name: Company glossary
sources:
  - id: platform
    file: terms/platform.yaml
  - id: payments
    url: https://payments.internal/.well-known/glossary
```

Each source needs a unique ID and a file or HTTP(S) URL. Relative paths resolve from the config file. Restart after configuration changes; glossary content is polled automatically, every five minutes by default.

See the [annotated config](../examples/config.yaml) for settings and [source rules](sources.md) for discovery, authentication, and failures.

## Binary

```sh
./jargonary -config config.yaml
```

The default public listener is `:8080`.

## Container

```sh
docker run --rm -p 8080:8080 \
  -v "$PWD/config.yaml:/config/config.yaml:ro" \
  -v "$PWD/terms:/config/terms:ro" \
  ghcr.io/alampros/jargonary:0.1.0 -config /config/config.yaml
```

Use a release version without the leading `v`, or an image digest. The container runs as **65532:65532**; mounts must be readable by that user.

## Persistence

SQLite defaults to memory. To retain the cache across restarts:

```yaml
database: /data/glossary.db
```

Mount persistent storage at `/data`, writable by UID/GID 65532. Run one instance per database file.

## Proxy and health

Terminate TLS and authenticate users at your proxy. Set `public_url` to the external URL. Source/export secrets do not protect the public UI; operator and MCP listeners have no built-in authentication.

`/healthz` checks the process; `/readyz` checks database access, not completion of source fetches. See [operations](operations.md) for diagnostics and subpath hosting.
