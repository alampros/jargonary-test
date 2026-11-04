# Operator reference

Jargonary runs as one process with a bundled web UI and a disposable SQLite cache. Configured glossary sources are the source of truth. Configure local or remote glossary sources, and choose persistent or in-memory SQLite storage. Put TLS and user authentication in your reverse proxy or hosting layer.

This guide covers deployment and day-to-day operation. For building or changing the application, see the [development guide](development.md).

## Container

Use a published image from `ghcr.io/alampros/jargonary`, pinned to a release tag or digest. Stable releases have patch, minor, and major tags; `dev` follows branch builds and prereleases have their full prerelease tag. For a local trial from the repository, run `docker compose up --build` and open `http://localhost:8080/`.

Plan the deployment around these settings:

| Resource | Deployment options |
| --- | --- |
| Process | One process per database file; prevent overlapping instances when using persistent storage |
| Storage | In-memory SQLite by default; optionally configure a database file on a writable persistent volume |
| Configuration | Mount YAML read-only and pass `-config /config/config.yaml`; omit it to use defaults and directory discovery |
| Sources | Configure HTTP(S) URLs, local files, or both. Remote-only deployments need no local glossary mount |
| Network | Inbound HTTP on port 8080; outbound access to configured HTTP(S) sources |
| Permissions | The image runs as UID/GID **65532:65532**; any database directory must be writable and mounted input files readable by that user |

The image includes trusted CA roots and needs no Go, Node, external database, or runtime asset CDN for glossary pages. It is built from `scratch`, so it has no shell or command-line HTTP client; use platform HTTP probes for health checks. The interactive API reference at `/docs` uses external CDN assets.

Do not scale replicas against a shared database file. Each independently deployed instance has its own database and polls its own sources.

The default is `database: ":memory:"`, so SQLite needs no writable database directory or persistent volume. Accepted snapshots, search indexes, and scrape history are lost when the process stops; sources must be fetched again after every restart. To retain last-good snapshots across restarts, set `database: /data/glossary.db` and mount persistent storage at `/data`. A running instance retains its cached snapshots during upstream outages in either mode.

A standalone release binary uses the same configuration:

```sh
./jargonary -config /path/to/config.yaml
```

## Configuration

Start with a small config using paths as seen by the application:

```yaml
site_name: Company glossary
server:
  listen: 0.0.0.0:8080
public_url: https://glossary.example.com
sources:
  - id: platform
    file: /terms/platform.yaml
  - id: payments
    url: https://payments.internal/.well-known/glossary
```

Mount that file and start the image with `-config /config/config.yaml`. Relative file paths resolve against the config file's directory; absolute paths avoid ambiguity in containers. See the [annotated configuration](../examples/config.yaml) for more options.

Configuration and secret references resolve at startup. **Restart after changing configuration or rotating credentials.** Existing source files are polled, so editing their content needs no restart.

The database setting accepts SQLite’s special `:memory:` value; other relative database paths resolve against the config file’s directory.

### Defaults and limits

| Setting | Default | Purpose |
| --- | --- | --- |
| `server.listen` | `0.0.0.0:8080` | Public UI and API |
| `database` | `:memory:` | In-memory SQLite; set a file path to persist cached data |
| `terms_dir` | `/terms` | Source discovery when `sources` is omitted |
| `scrape_interval` | `5m` | Poll interval; allowed range `1m`–`24h` |
| `timeout` | `15s` | Per fetch attempt; allowed range `1s`–`1m` |
| `max_bytes` | `8388608` (8 MiB) | Per glossary; allowed range 1 byte–64 MiB |
| `admin.enabled` / `admin.listen` | `false` / `127.0.0.1:9090` | Optional operator listener |
| `mcp.enabled` / `mcp.listen` | `false` / `127.0.0.1:9091` | Optional read-only AI tool listener |

Each source needs a unique ID and exactly one `file` or HTTP(S) `url`. JSON, YAML, and CSV are supported. Format is inferred from the extension, defaulting to JSON; set `format` to override it. A source can override `scrape_interval`. Up to four sources scrape concurrently, with one active attempt per source. Failures retry with bounded exponential backoff and jitter. **All HTTP redirects are rejected**, including redirects on the same origin; configure the final URL.

### Discover files instead

Omit `sources` to discover immediate `.json`, `.yaml`, `.yml`, and `.csv` files in `terms_dir` at startup. Each filename without its extension becomes a source ID; these must be unique across formats. Subdirectories and symlinks are ignored.

Restart to discover added files. Deleting a file while running leaves its last accepted content cached; restarting without that discovered source retires it. Explicit `sources` replaces discovery, and `sources: []` disables it.

### Branding and the bundled glossary

Set `site_name` (default `Jargonary`) and `theme`, or mount custom CSS and JavaScript. See [branding and themes](customization.md) for appearance settings and their environment overrides.

The bundled “Jargonary project terms” glossary has reserved source ID `project` and appears in browsing and search by default. Set `hide_jargonary_glossary: true` to hide it when company sources exist. With zero company sources after startup discovery, it stays enabled as a fallback; failed or empty scrapes do not trigger that fallback.

The root `/.well-known/glossary` serves only this bundled glossary, and returns 404 when it is disabled. Publish company content through [named exports](#content-and-publication).

## Network and access

Only the public listener starts by default. There are no built-in user accounts. Use your proxy for reader authentication and TLS, and keep the optional operator and MCP listeners on restricted networks or behind an authenticated proxy.

Set `public_url` to the externally visible URL for absolute links and canonical/social metadata. For subpath hosting, include the prefix in `public_url` and have the proxy strip it before forwarding. `/.well-known/glossary` still belongs at the origin root; map it separately or configure consumers with an explicit URL. This path is a proposed publishing convention, not a registered standard or automatic network discovery mechanism.

Source URLs are operator-controlled and can reach internal networks. Do not accept them from public readers. Public responses omit fetch URLs and resolved configuration.

### Source and export secrets

Use either an environment reference or a mounted secret file:

```yaml
sources:
  - id: payments
    url: https://payments.internal/.well-known/glossary
    auth:
      shared_secret:
        file: /run/secrets/payments
        # Or use: env: PAYMENTS_SECRET
```

Use a random secret of at least 32 non-whitespace characters. Secret file contents are trimmed. Authenticated source URLs require HTTPS; the consumer sends `Authorization: Bearer <secret>`. Restart to rotate the value.

### Optional AI access

`mcp.enabled: true` starts a separate, stateless, read-only MCP listener at `mcp.listen`. It has no built-in authentication. Treat access to it as access to the instance's glossaries. See the [MCP overview](../README.md#expose-a-glossary-to-ai-agents).

## Health and troubleshooting

Probe these paths on the public listener:

| Endpoint | Meaning |
| --- | --- |
| `/healthz` | The HTTP process is responding |
| `/readyz` | The database is accessible |

Initial source fetches run asynchronously. Readiness does **not** mean all sources have loaded, and upstream outages do not make a cached glossary unready. Monitor source freshness separately.

## Operator diagnostics

Enable the read-only operator UI when you need source status or configuration diagnostics:

```yaml
admin:
  enabled: true
  listen: 0.0.0.0:9090
```

Binding to all container interfaces makes it reachable through container networking; restrict access with your platform. For local Docker port publishing, bind the host side to `127.0.0.1:9090:9090`. The default container-loopback address cannot be reached through ordinary Docker port publishing.

The operator listener provides:

| Path | What to check |
| --- | --- |
| `/` | Operator UI; diagnostics refresh every 15 seconds |
| `/api/v1/scrapes` | Schedules, backoff, timestamps, counts, and the last 20 outcomes per source |
| `/api/v1/config` | Resolved settings and origins; secret references are shown, tokens and URL queries omitted |
| `/api/v1/diagnostics` | Unresolved relationships, self-replacements, and replacement cycles |
| `/openapi.json`, `/docs` | Operator API documentation |

These operator routes are separate from the public listener. For stale terms, check the source's latest scrape outcome before restarting: a failed fetch deliberately keeps the previous snapshot.

## Content and publication

Source systems own content editing and approval. Jargonary reads each source as a complete glossary and replaces its accepted snapshot atomically.

| Event | Effect on cached content |
| --- | --- |
| Successful valid fetch | Replaces all content for that source |
| Valid empty glossary | Clears that source |
| Failed read, invalid data, HTTP 401/404/410 | Keeps the last successful snapshot indefinitely |
| Source removed from startup configuration | Retires its cached content on restart |

A stale source and a deprecated term are separate conditions. Keep source IDs stable across configuration changes: matching terms or entry IDs in different sources retain their separate identities.

### Publish a named export

Exports at `/.well-known/glossary/{exportId}` select company content by source and publication tags. Each export must explicitly choose a shared secret or `auth.public: true`. For example:

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

Filtering happens on the publisher. Use a source allowlist plus tags when source owners should not all be able to publish to an audience. The bundled `project` source is always excluded, including from `include: all`, and cannot be selected explicitly.

An export returns **503 until every selected configured source has an accepted snapshot**. Without a source allowlist, every configured source must have a snapshot, even for tag selection, because unavailable sources' tags are unknown. An accepted empty glossary counts as ready. Later failures keep publishing the last successful snapshot; sources outside the allowlist do not block it.

Exports preserve distinct source identities and remove relationships to excluded targets while retaining effective deprecation. Terms and unrestricted aliases are shared entry metadata and accompany any selected sense. Filtering does not redact prose, URLs, examples, tags, source IDs, or attribution: publishers must ensure selected content suits the audience. Imported exports cannot be re-exported in v1. Revoking access stops future fetches, but cannot recall cached snapshots or downloaded copies.

**A protected export does not protect the rest of the instance.** Other public routes remain anonymous. To expose only an authenticated export, restrict proxy routes using the [export-only proxy example](../examples/export-only-proxy.conf).

For content structure and validation, see the [format guide](dictionary-format.md) and [example glossary](../examples/terms/company.yaml).

## Restarts and upgrades

Deploy the new binary or image with your configuration and source access. Jargonary recreates its cache from sources; no database backup is required. If an optional file-backed cache is incompatible with the version you want to run, stop the instance, discard that cache, and restart. Sources must be reachable to repopulate it.

On SIGTERM, Jargonary stops listeners and cancels in-flight scraping. Allow time for graceful shutdown before forcing termination; HTTP shutdown has a 10-second budget.

## Public API

The public listener serves the glossary UI at `/`, JSON search at `/api/v1/search?q=API`, and plain definition pages such as `/definitions/company/component`. Open `/openapi.json` (or `/openapi.yaml`) for the API contract and `/docs` for the interactive reference.

Search accepts literal text rather than FTS query syntax. Results rank exact terms, exact scoped aliases, term prefixes, then full-text matches. Use `limit`/`offset` and returned navigation links for pagination.

## Development and operations

For local development, builds, and checks, see the [development guide](development.md).
