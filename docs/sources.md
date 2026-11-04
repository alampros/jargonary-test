# Source settings

Configure sources in `config.yaml`; the [deployment guide](reference.md) shows the basic setup. See the [annotated example](../examples/config.yaml) for all options.

## Polling and limits

| Setting | Default | Allowed values |
| --- | --- | --- |
| `scrape_interval` | `5m` | `1m`–`24h`; can also be set per source |
| `timeout` | `15s` | `1s`–`1m` per fetch attempt |
| `max_bytes` | `8388608` (8 MiB) | 1 byte–64 MiB per glossary |

Format is inferred from the file or URL extension, defaulting to JSON. Set a source's `format` to override it. HTTP redirects are rejected; configure the final URL.

Sources scrape independently, with up to four concurrent fetches. Failures retry with backoff and keep the last successful snapshot. A valid empty glossary clears that source. Removing a source from configuration retires its cached content on restart. Keep source IDs stable: matching terms or entry IDs across sources remain distinct.

## Discover local files

Omit `sources` to load files directly inside `terms_dir` (default `/terms`). Supported extensions are `.json`, `.yaml`, `.yml`, and `.csv`; subdirectories and symlinks are ignored. Each filename without its extension becomes a source ID and must be unique.

Restart to discover added or removed files. Deleting a file while running keeps its cached content. Explicit `sources` replaces discovery; `sources: []` disables it.

## Protected sources

Reference a secret file or environment variable:

```yaml
sources:
  - id: payments
    url: https://payments.internal/.well-known/glossary
    auth:
      shared_secret:
        file: /run/secrets/payments
        # Or: env: PAYMENTS_SECRET
```

Authenticated source URLs require HTTPS. Jargonary sends `Authorization: Bearer <secret>`. Use a random secret of at least 32 non-whitespace characters. Secret files are trimmed; restart after rotating a secret.

Only operators should configure source URLs, since they can reach internal networks.

## Bundled project glossary

The built-in Jargonary glossary uses reserved source ID `project`. Set `hide_jargonary_glossary: true` to hide it when company sources are configured. With no company sources, it remains as a fallback; empty or failed scrapes do not trigger that fallback.

The root `/.well-known/glossary` serves this bundled glossary, returning 404 when it is disabled. To share company content, configure a [named export](publishing.md).
