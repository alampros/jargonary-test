---
title: Source settings
editUrl: https://github.com/alampros/jargonary/edit/main/docs/sources.md
---

See the [annotated config](../examples/config.yaml) for all settings.

## Polling and limits

| Setting | Default | Range |
| --- | --- | --- |
| `scrape_interval` | `5m` | `1m`–`24h`; overridable per source |
| `timeout` | `15s` | `1s`–`1m` per attempt |
| `max_bytes` | `8388608` (8 MiB) | 1 byte–64 MiB |

Format comes from the file/URL extension, defaulting to JSON; `format` overrides it. HTTP redirects are rejected.

Up to four sources fetch concurrently. Failed fetches retry with backoff and retain the last accepted snapshot; a valid empty glossary clears it. Removing a configured source retires its cache on restart. Keep source IDs stable; identical entry IDs across sources remain distinct.

## Discover local files

Omitting `sources` loads `.json`, `.yaml`, and `.yml` files directly in `terms_dir` (default `/terms`). Filenames without extensions become source IDs and must be unique. Subdirectories and symlinks are ignored.

Discovery runs at startup. Deleting a file while running retains its cache. Explicit `sources` replaces discovery; `sources: []` disables it.

## Protected sources

```yaml
sources:
  - id: payments
    url: https://payments.internal/.well-known/glossary
    auth:
      shared_secret:
        file: /run/secrets/payments
        # Or: env: PAYMENTS_SECRET
```

Authenticated URLs require HTTPS; requests send `Authorization: Bearer <secret>`. Use random secrets of at least 32 non-whitespace characters. Secret files are trimmed; rotation requires a restart. Only operators should configure URLs, which can reach internal networks.

## Bundled project glossary

Source ID `project` is reserved. `hide_jargonary_glossary: true` hides it when company sources exist; with none configured or discovered, it remains the fallback. Empty or failed scrapes do not activate the fallback.

The root `/.well-known/glossary` serves this glossary, or 404 when disabled. Company content uses [named exports](publishing.md). The path is a proposed convention, not a registered standard or automatic network discovery.
