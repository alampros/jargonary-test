# Working on Jargonary

## Product and scope

Jargonary is a distributed glossary: teams own their vocabulary, publish schema-defined data, and combine sources in a searchable UI. A glossary defines words in context; identical terms can have different meanings across teams. Use “glossary” in new user-facing prose. Existing code identifiers and format-guide filenames may still use “dictionary”; do not rename them as incidental cleanup.

Keep deployment simple: a standalone Go binary or Docker container with bundled UI assets and SQLite storage. User authentication and TLS termination belong in a proxy or hosting layer. Shared-secret authentication for sources and exports is part of the application. Source systems own content editing and approval.

Local JSON, YAML, and CSV files and configured HTTP(S) endpoints supply glossaries. `/.well-known/glossary` is a proposed publishing convention, not a registered standard or automatic network discovery mechanism. The root endpoint serves the bundled project glossary; named exports publish selected source content.

## Repository map

- `cmd/jargonary`: application entry point, CLI validation, and schema generation.
- `internal/config`, `internal/scraper`, `internal/vocabulary`: startup configuration, source ingestion, parsing, and validation.
- `internal/store`, `internal/publishing`: SQLite cache/search and filtered exports.
- `internal/server`: HTTP APIs, server-rendered HTML, and public/operator listeners.
- `ui`: React, TypeScript, and Vite frontend; `internal/webui` embeds its production build.
- `internal/themes`: bundled themes; `glossary/project.json`: bundled project glossary.
- `docs-site/public/schema/v0.0.0`, `examples`: generated JSON Schemas and runnable sample data/configuration.
- `docs-site`: Astro Starlight GitHub Pages site serving docs from `docs/` and serving the generated, checked-in schemas from `docs-site/public/schema/v0.0.0/`.

Read [README.md](README.md) for the overview, [docs/dictionary-format.md](docs/dictionary-format.md) for the format contract, [docs/reference.md](docs/reference.md) for operations/development, and [docs/customization.md](docs/customization.md) for branding. Keep the README concise; put detailed guidance in `docs/`.

## Development commands

Run commands from the repository root. The project uses Go 1.27 and Node 26. Task and Air are pinned Go tools; Biome, TypeScript, and Vitest are project-local dependencies in `ui`.

| Command | Purpose |
| --- | --- |
| `go tool task dev` | Run Air for Go rebuilds and Vite for React HMR |
| `go tool task run` | Build the production UI and run the example configuration |
| `go tool task build` | Build the UI and standalone binary |
| `go tool task ui:build` | Install locked frontend dependencies and build embedded assets |
| `go tool task fmt` | Apply Go formatting and safe Biome fixes; review the changes |
| `go tool task check` | Run the required completion gate |
| `go tool task schema VERSION=v0.0.0` | Regenerate the structural JSON Schema from Go types |
| `go tool task docs:build` | Assemble the local Pages preview |
| `go tool task docs:preview` | Build and serve the docs and schemas on loopback port 4321 |
| `go tool task --list` | List available tasks |

Open the Go listener, normally `http://localhost:8080`, during development. Go loads Vite modules only with `-dev-ui`; Air excludes frontend sources and generated assets. Keep Air out of the runtime container. Production assets under `internal/webui/dist` are generated and uncommitted; rebuild them before testing production UI behavior. Do not reintroduce Make or require global Task/Air installations.

## Change and validation practices

Preserve source-qualified identities and distinct senses; matching terms or entry IDs across sources must not merge their ownership. Failed scrapes retain the last successful snapshot; a valid empty glossary clears a source. Keep operator diagnostics on the separate, opt-in listener and credentials out of public responses.

For format changes, update parsing/semantic validation, regenerate the schema, and update affected examples and documentation together. JSON Schema covers structure; application validation also checks semantic constraints. Validate edited glossaries with:

```sh
go run ./cmd/jargonary -validate examples/terms/web-development.yaml -format yaml
```

Use the actual file path and format (`json`, `yaml`, or `csv`). Maintain project vocabulary in `glossary/project.json`, the single source served by Jargonary. Keep Glossary first, distinguish it from the related term Dictionary, and preserve stable IDs and relationship targets.

Add or update focused tests for changed behavior. Follow the existing Go tests and frontend Vitest/Testing Library patterns. The root `biome.json` owns shared rules; `ui/biome.json` inherits them through `extends: "//"` and must not declare another `root` property.

## Required completion gate

Before considering any change completed, run **`go tool task check`** from the repository root and ensure it passes. Run it again after fixing failures or changing code/configuration. Do not skip checks or weaken their configuration to claim success. If the gate cannot run, report the concrete blocker and do not describe the change as complete.

The gate runs Go formatting (`go fmt`, failing if it changes packages), `go vet`, Go unit/integration tests with the race detector, Biome lint/format/import checks, TypeScript checking, and Vitest UI unit tests. Review formatting changes made by the gate before rerunning it. `go tool task go:lint` is the shared Go formatting/static-analysis entry point for local checks and CI.
