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
| `go tool task ui:storybook` | Start Storybook on port 6006 |
| `docker compose up --build` | Try the production container |
| `go tool task --list` | List all tasks |

Production UI assets in `internal/webui/dist` are generated and uncommitted. Rebuild them with `go tool task ui:build` before testing production UI changes.

## Documentation site

The GitHub Pages site uses Astro Starlight in `site/`, with locked dependencies separate from the application UI. Markdown in `docs/` is the canonical content: start each file with a level-one title. The build prepares ignored files in `site/.content/`, adds Starlight titles and edit links, and rewrites Markdown links for the deployed site's base path. Links to repository examples and source files open on GitHub. Edit the originals in `docs/`, not the generated copies.

Build and preview the complete site, including schemas:

```sh
go tool task pages:preview
```

Open `http://127.0.0.1:4321/jargonary/`. `go tool task pages:build` builds without starting a server; output is in `.cache/pages/`. To use a different domain or base path, pass `--base-url`, for example `go tool task pages:build -- --base-url https://docs.example.com/`.

`go tool task pages:check` tests publishing and link rewriting, builds Starlight with its static search index, and checks internal links and heading fragments. It is included in `go tool task check` and CI. For a new guide, add its Markdown file in `docs/` and add it to the sidebar in `site/astro.config.mjs`.

The Pages build combines documentation and versioned schemas in one artifact. Schema URLs under `/schema/` are preserved; stable release schemas come from their Git tags, and the latest alias follows the newest stable release. Local builds without release metadata use the checked-in schema. To reproduce release selection locally, pass `--releases /path/to/releases.json` with GitHub's release list and fetch the corresponding tags first.

## Storybook

Run `go tool task ui:storybook` and open `http://localhost:6006/` to explore the public glossary and operator console in isolation. Pass Storybook flags after `--`, for example `go tool task ui:storybook -- --no-open`. Stories live beside their components in `web/src/*.stories.tsx`. The shared preview loads the application fonts and CSS; MSW supplies the operator API responses without a running Go server.

The `Definition` stories render the definition component on its own, including single meanings, multiple meanings, multiple sources, and stale content. `PublicApp` and `AdminApp` stories render complete pages for integration coverage.

Run `npm --prefix web run build-storybook` to generate `web/storybook-static/`. Run `npm --prefix web test -- --project storybook` for browser story tests; `npm --prefix web test` runs both stories and existing unit tests. The browser tests require Playwright's Chromium binary, installed by the Storybook initializer. On a fresh checkout, install it with `npm --prefix web exec -- playwright install chromium`.

## When changing formats or SQL

- Glossary format: update parsing, validation, examples, and docs; regenerate with `go tool task schema`.
- SQL queries: edit `internal/store/sql/queries.sql`, then run `go tool task go:sql`.
- Validate content: `go run ./cmd/jargonary -validate examples/terms/company.yaml -format yaml`.

## GitHub Actions

The [CI workflow](../.github/workflows/ci.yaml) runs checks on pull requests, skipping branches beginning with `release-please--`. Other branch pushes do not trigger CI. Release-please changes are checked after merging into `main`.

The [build and release workflow](../.github/workflows/release.yaml) runs on pushes to `main`, version tags, and manual release requests. It calls the same CI checks before publishing. Only `main` publishes development container images; release runs publish versioned images and binaries. After checks and artifact publishing succeed, it calls the [Pages workflow](../.github/workflows/pages.yaml) to publish the site and versioned schemas from the checked commit or release tag. Pages has no independent trigger.

See [AGENTS.md](../AGENTS.md) for contribution rules and the [operator reference](reference.md) for deployment.
