---
title: Status and APIs
editUrl: https://github.com/alampros/jargonary/edit/main/docs/operations.md
---

## Operator listener

A failed fetch keeps old content; `/readyz` does not wait for initial fetches. Enable the operator UI to inspect source status:

```yaml
admin:
  enabled: true
  listen: 0.0.0.0:9090
```

Keep it private or behind an authenticated proxy. In Docker, publish `127.0.0.1:9090:9090`; container-loopback listeners are unreachable through ordinary port publishing.

| Operator path | Content |
| --- | --- |
| `/` | UI; refreshes every 15 seconds |
| `/api/v1/scrapes` | Fetch schedules and outcomes |
| `/api/v1/config` | Resolved settings and origins |
| `/api/v1/diagnostics` | Unresolved relationships and replacement problems |
| `/openapi.json`, `/docs` | API reference |

These routes are absent from the public listener. Diagnostics include secret references but omit resolved tokens and source URL query strings.

## Logs and upgrades

Logs go to stdout: text in a terminal, JSON otherwise. `NO_COLOR` or `TERM=dumb` disables colors. Allow 10 seconds for shutdown on SIGTERM.

SQLite is disposable cache. If an upgrade cannot read it, stop the instance, discard the cache, and restart with sources reachable.

## Public API

Use `/docs` or `/openapi.json` (`.yaml` also available) for the contract. The interactive reference loads CDN assets.

- `/api/v1/search?q=API`: literal search with `limit`/`offset`; exact terms and scoped aliases rank ahead of prefixes and full-text matches.
- `/definitions/API`: matching definitions across sources.
- `/definitions/company/component`: source-qualified entry.

For AI clients, see [MCP](mcp.md).

## Subpath hosting

Include the prefix in `public_url`; the proxy must strip it before forwarding. Map `/.well-known/glossary` separately at the origin root, or give consumers an explicit URL.
