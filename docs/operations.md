# Status and APIs

## Check a stale source

Check the source's latest fetch outcome first: a failed fetch deliberately keeps its previous content. Initial fetches run asynchronously; `/readyz` only checks database access.

For source status and configuration diagnostics, enable the read-only operator UI:

```yaml
admin:
  enabled: true
  listen: 0.0.0.0:9090
```

Keep this listener private or behind an authenticated proxy. For Docker, publish it with `127.0.0.1:9090:9090`; the default container-loopback listener cannot be reached through ordinary port publishing.

| Operator path | Provides |
| --- | --- |
| `/` | Operator UI, refreshing every 15 seconds |
| `/api/v1/scrapes` | Schedules, timestamps, counts, and recent outcomes |
| `/api/v1/config` | Resolved settings and their origins |
| `/api/v1/diagnostics` | Unresolved relationships and replacement problems |
| `/openapi.json`, `/docs` | Operator API documentation |

These routes are separate from the public listener. Diagnostics show secret references, but omit resolved tokens and source URL query strings.

## Logs and restarts

Logs go to stdout: readable text in a terminal, JSON when piped or redirected. Startup logs show resolved configuration and listener addresses. Set `NO_COLOR` or `TERM=dumb` to disable terminal colors.

To upgrade, replace the binary or image and restart with the same configuration. SQLite is a cache; sources remain the source of truth. If a cache file is incompatible, stop the instance, discard it, and restart with sources reachable. Allow 10 seconds for HTTP shutdown on SIGTERM.

## Public API

The public listener serves:

- `/api/v1/search?q=API` for JSON search, with `limit`/`offset` pagination.
- `/definitions/API` for matching definitions across sources.
- `/definitions/company/component` for a source-qualified entry.
- `/openapi.json` or `/openapi.yaml` for the API contract, and `/docs` for the interactive reference.

Search takes literal text. Exact terms and scoped aliases rank ahead of prefixes and full-text matches. The interactive API reference loads external CDN assets.

## AI access

See [Connect an AI agent](mcp.md) for MCP setup, client connection details, and available tools.

## Subpath hosting

Set `public_url` to include the prefix and have your proxy strip it before forwarding. `/.well-known/glossary` belongs at the origin root; map it separately or configure consumers with an explicit URL.
