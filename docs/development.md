# Development

Requires Go 1.27 and Node 26. Run commands from the repository root; Task and Air are pinned Go tools, and frontend dependencies are project-local.

```sh
go tool task dev
```

Open `http://localhost:8080/`. Air rebuilds Go; Vite updates React as you edit. Refresh after Go restarts or HTML template changes. Ctrl-C stops both processes.

In dev mode, navigation reloads the page, and Vite injects CSS through JavaScript after the initial HTML arrives. This can briefly show unstyled content and make the layout jitter as styles, fonts, and React load. Use `go tool task run` to check the production build, which loads bundled CSS through stylesheet links.

Pass app flags after `--`:

```sh
go tool task dev -- -config /absolute/path/config.yaml
```

## Common commands

| Command | Purpose |
| --- | --- |
| `go tool task check` | Required before finishing: formatting, vet, race tests, Biome, TypeScript, Vitest |
| `go tool task fmt` | Apply formatting and safe fixes; review changes |
| `go tool task build` | Build UI and `bin/jargonary` |
| `go tool task run` | Build and run the example configuration |
| `go tool task storybook` | Start Storybook on port 6006 |
| `docker compose up --build` | Try the production container |
| `go tool task --list` | List all tasks |

Production UI assets in `internal/webui/dist` are generated and uncommitted. Rebuild them with `go tool task ui` before testing production UI changes.

## Storybook

Run `go tool task storybook` and open `http://localhost:6006/` to explore the public glossary and operator console in isolation. Pass Storybook flags after `--`, for example `go tool task storybook -- --no-open`. Stories live beside their components in `web/src/*.stories.tsx`. The shared preview loads the application fonts and CSS; MSW supplies the operator API responses without a running Go server.

The `Definition` stories render the definition component on its own, including single meanings, multiple meanings, multiple sources, and stale content. `PublicApp` and `AdminApp` stories render complete pages for integration coverage.

Run `npm --prefix web run build-storybook` to generate `web/storybook-static/`. Run `npm --prefix web test -- --project storybook` for browser story tests; `npm --prefix web test` runs both stories and existing unit tests. The browser tests require Playwright's Chromium binary, installed by the Storybook initializer. On a fresh checkout, install it with `npm --prefix web exec -- playwright install chromium`.

## When changing formats or SQL

- Glossary format: update parsing, validation, examples, and docs; regenerate with `go tool task schema`.
- SQL queries: edit `internal/store/sql/queries.sql`, then run `go tool task sql`.
- Validate content: `go run ./cmd/jargonary -validate examples/terms/company.yaml -format yaml`.

See [AGENTS.md](../AGENTS.md) for contribution rules, the [operator reference](reference.md) for deployment, and the [release](../.github/workflows/release.yaml) and [Pages](../.github/workflows/pages.yaml) workflows for publishing.
