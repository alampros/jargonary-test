# Company dictionary (working name: Jargonary)

An MIT-licensed, self-hosted dictionary that scrapes company vocabulary from files and HTTP endpoints. Sources own the content; this application validates, indexes, and publishes it.

The public dictionary and optional read-only operator UI are implemented with React/Vite and served by Go. Definition pages include readable initial HTML and server-generated preview metadata. The scraper, REST API, filtered exports, and SQLite search remain available independently. Preview images and MCP are deferred.

## Run locally

Requires Go 1.27 and Node 26 for frontend builds. No system SQLite installation or C compiler is needed for normal builds. The container needs neither Go nor Node at runtime.

```sh
go tool task run
```

The example discovers `examples/terms/company.yaml` at startup. Initial scrapes run asynchronously; accepted data survives restarts.

- Dictionary UI: `http://localhost:8080/`
- Public API: `http://localhost:8080/api/v1/search?q=PI`
- Multiple meanings: `http://localhost:8080/api/v1/sources/company/entries/component`
- Plain definition page: `http://localhost:8080/definitions/company/component`
- Project's own dictionary: `http://localhost:8080/.well-known/glossary`
- OpenAPI: `http://localhost:8080/openapi.json` (also YAML)
- Interactive API reference: `http://localhost:8080/docs` (Huma's renderer uses external CDN assets)
- Liveness/readiness: `/healthz` and `/readyz`

The bundled Jargonary project glossary has reserved source ID `project` and is enabled by default. “All sources” includes every enabled source in browsing and search, including this glossary. Exports using `include: all` also include it unless they select specific sources.

Set `hide_jargonary_glossary: true` in YAML configuration to disable the bundled source when company sources are configured or discovered. If there are zero company sources after startup discovery, the project glossary remains enabled as a fallback. Failed or empty company scrapes do not trigger this fallback. Restart to apply this setting; disabling removes cached project terms and makes `/.well-known/glossary` return 404. Named company exports continue to use their configured source selectors.

`go tool task run` installs only project-local frontend dependencies, builds the UI, and starts Go. Once the UI is built, `go run ./cmd/jargonary -config examples/config.yaml` also works. Rebuild the frontend and restart Go after UI changes; assets are embedded in the binary. API-only Go builds compile without Node, with plain definition HTML and an explicit build-required page at `/` until `go tool task ui` is run.

## Container

```sh
docker compose up --build
```

The Compose example builds the image locally, mounts sample vocabulary read-only, and persists SQLite in a named volume. Only port 8080 is published. The image runs as UID/GID 65532 and includes trusted CA roots for HTTPS scrapes. It requires no runtime network access except configured sources.

To use your own sources, replace the `/terms` mount. Without a config file, immediate `.json`, `.yaml`, `.yml`, and `.csv` files in `/terms` are discovered at startup. Each filename without its extension becomes its source ID; IDs must be unique across formats. Symlinks and subdirectories are not discovered. Existing files are polled; restart to discover new files or apply configuration changes. Deleting a file preserves its cached snapshot until its source is deliberately retired through configuration/startup discovery or a valid empty dictionary is read.

For explicit configuration, mount a YAML config and run the image with `-config /config/config.yaml`. Use absolute container paths such as `/data/glossary.db` and `/terms`; relative paths resolve against the config file's directory. A host bind mount for `/data` must be writable by UID/GID 65532. Do not run multiple application processes against the same database file.

Tagged `v*` pushes and manual release workflows build and publish `ghcr.io/<owner>/<repository>` for Linux amd64/arm64, with provenance and an SBOM. A GitHub remote, workflow permissions, and package visibility must be configured before release. No image has been published by this implementation.

## Configuration

See [examples/config.yaml](examples/config.yaml). Defaults:

| Setting | Default |
| --- | --- |
| `site_name` | `Company` (the masthead label) |
| `server.listen` | `0.0.0.0:8080` |
| `admin.enabled` | `false` |
| `admin.listen` | `127.0.0.1:9090` |
| `database` | `data/glossary.db` (under `/data` in the image) |
| `terms_dir` | `/terms` |
| `scrape_interval` | `5m` |
| `timeout` | `15s` per attempt |
| `max_bytes` | 8 MiB per dictionary |

Explicit `sources` replace directory discovery; `sources: []` disables it. Each source needs an ID and exactly one `file` or HTTP(S) `url`. Format is inferred from the extension, defaulting to JSON, and can be overridden. Source-specific `scrape_interval` overrides the global interval (1 second to 24 hours). Four sources may scrape concurrently; each source has only one active attempt. Failures use bounded exponential backoff with jitter. All redirects are rejected, including same-origin redirects.

Configure `public_url` for absolute links and canonical/OG URLs. For subpath hosting, include the prefix there and have the reverse proxy strip it before forwarding requests. The conventional `/.well-known/glossary` remains at the origin root; map it separately or use an explicit scrape URL.

Source and export authentication uses `auth.shared_secret.env` or `auth.shared_secret.file`, resolved at startup. Use a random secret of at least 32 non-whitespace characters; file contents are trimmed. Authenticated source URLs must use HTTPS. The consumer sends `Authorization: Bearer <secret>`. Restart to rotate credentials or change configuration. An export must explicitly choose a shared secret or `auth.public: true`.

The operator controls outbound targets. HTTP sources may reach internal networks; do not accept target configuration from public readers. Source credentials are not forwarded through redirects. Public responses contain no source fetch URLs or resolved configuration.

## Operator diagnostics

Set `admin.enabled: true` to start the separate listener. `/` opens the read-only operator UI with expandable source histories, resolved configuration, and vocabulary diagnostics. Its backing APIs are:

- `/api/v1/config`: resolved settings, setting origins, and secret references, with tokens and URL queries omitted.
- `/api/v1/scrapes`: schedules, timestamps, running/backoff status, counts, and the last 20 outcomes per source.
- `/api/v1/diagnostics`: unresolved relationships, self-replacements, and replacement cycles.
- `/openapi.json` and `/docs`: separate operator API documentation.

These routes do not exist on the public listener, and no operator listener starts by default. There are no user accounts: restrict operator access through networking or an authenticated proxy. To publish it to the Docker host, explicitly bind `admin.listen` to `0.0.0.0:9090` inside the container and publish **`127.0.0.1:9090:9090`** on the host. Container-loopback binding is not reachable through ordinary Docker port publishing. Do not publish it publicly.

The public UI uses an alphabet rail on desktop and an independently scrollable strip on mobile, with 44px mobile targets and arrow/Home/End keyboard navigation. Search and source filters use ordinary GET navigation, so URLs are shareable and browser back/forward work. `/` focuses search when pressed outside a form control. Definitions retain all of an entry's meanings and show replacement links for deprecated terminology. The operator UI refreshes source/relationship diagnostics every 15 seconds without exposing them on the public listener.

## Content and publication

The [project glossary](GLOSSARY.md), [format contract](docs/dictionary-format.md), and [example dictionary](examples/terms/company.yaml) describe entries, senses, aliases, and relationships.

Each configured source is a complete dictionary, not a list of dictionary URLs. Successful snapshots replace all of that source's content atomically. Failed reads—including 401, 404, and 410—preserve the last successful snapshot indefinitely. A valid empty dictionary clears a source. Removing a source from the resolved startup configuration retires its cached content. A stale source and a deprecated term are different conditions.

Optional named exports at `/.well-known/glossary/{exportId}` filter senses **on the publisher**, before the consumer receives anything. Use source allowlists plus publication tags when not every source owner should be able to publish to the audience. Entry terms and unrestricted aliases are shared metadata and are included when any sense is selected. Relationships to excluded targets are removed; effective deprecation is preserved. See [examples/config.yaml](examples/config.yaml).

Filtering guarantees structural exclusion. Publishers remain responsible for the text, URLs, examples, source IDs, tags, and attribution they choose to publish; the application does not classify or redact confidential prose. A private publisher's other public routes are still anonymous, so exposing an authenticated export must not expose the entire instance. See the [export-only proxy example](examples/export-only-proxy.conf).

Export IDs preserve source-qualified identities and relationships without merging colliding term text or entry IDs. Imported exports cannot themselves be re-exported in v1. Revoking access stops future scrapes; it does not recall accepted snapshots or downloaded copies.

## Development and operations

Run `go tool task dev` to start two processes: [Air](https://github.com/air-verse/air) rebuilds and restarts Go after backend/configuration changes, and Vite serves frontend modules with React HMR. Open http://localhost:8080 (or the configured Go listener), not Vite's port. Go still renders definitions, social metadata, and initial page data; development HTML loads Vite's React Refresh preamble, client, and entry module from `http://127.0.0.1:5173`. UI edits update without rebuilding Go. Refresh the browser after a Go restart or HTML template change. The optional operator UI uses the same Vite server on its own Go listener.

Both processes stop with Ctrl-C; Task stops the other process if one fails. Air and Task are pinned project-local Go tools. Pass application flags after `--`, for example `go tool task dev -- -config /absolute/path/config.yaml`; configuration outside this repository is not watched. You can run `go tool task dev:server` and `go tool task dev:ui` in separate terminals instead (run `go tool task ui:deps` first). Vite binds only to loopback on a fixed port and allows loopback browser origins. Development mode is enabled explicitly by the `-dev-ui` application flag, requires the repository as the working directory, and permits Vite scripts, injected styles, and HMR connections in the HTML content security policy. Normal builds use embedded assets and the production policy.

```sh
go tool task check
go tool task test
go tool task build
go run ./cmd/jargonary -validate glossary/project.json
go run ./cmd/jargonary -validate examples/terms/company.yaml -format yaml
go tool task schema
```

`go tool task schema` generates the structural JSON Schema from Go types. Application validation also checks uniqueness, alias targets, supported relationships, and export provenance. Search ranks exact terms, exact scoped aliases, term prefixes, then full-text matches. Definition matches use SQLite FTS5 rank; short one-character tokens use exact/prefix lookup. Queries are literal text rather than FTS query syntax. Use `limit`/`offset` and returned navigation links to browse results.

`go tool task ui` performs a reproducible npm install and production build. The compiled UI lives under `internal/webui/dist` and is deliberately not committed; Docker and GitHub Actions build it before Go compilation. Fonts and icons are bundled, so dictionary pages need no external CDN. Their redistribution licenses are included at `/ui/assets/licenses/` (request an individual file such as `source-serif-4.txt`). The generated API reference at `/docs` still uses Huma's CDN-backed renderer.

Task is pinned as a project-local Go tool in `go.mod`; no global Task installation is needed. Run `go tool task --list` for available tasks. The required completion gate is `go tool task check`: Go formatting (failing if `go fmt` changes packages), `go vet`, Go race tests, Biome formatting/lint/import checks, TypeScript, and Vitest component tests. `go tool task fmt` applies formatting and safe fixes. The root Biome config owns shared rules; `web/biome.json` inherits them with `extends: "//"`, so only the root config declares the `root` property. See [AGENTS.md](AGENTS.md).

Back up the complete `/data` directory while the container is stopped; do not copy a running SQLite database file in isolation. Back up source files, configuration, and secret provisioning separately. Restore into a writable volume and restart with the same source IDs. A newer database schema is rejected by older binaries. Keep a stopped backup before upgrades; automatic downgrade is not provided. Rebuilding the cache from reachable sources loses last-good snapshots of sources that are no longer reachable.

The application stops HTTP listeners and in-flight scraping on SIGTERM. Readiness checks database access, not upstream availability: an offline publisher should not take a valid cached dictionary offline.

See [PROJECT_PLAN.md](PROJECT_PLAN.md) for staged work and unresolved design decisions.
