---
title: Development
editUrl: https://github.com/alampros/jargonary/edit/main/docs/development.md
---

Requires Go 1.27 and Node 26. Run from the repository root:

```sh
go tool task dev
```

Use `http://localhost:8080/`, not Vite's port. Refresh after Go restarts or HTML template changes. Dev mode can briefly show unstyled content; use `go tool task run` to check production rendering. Bundled UI assets are uncommitted, so rebuild with `go tool task ui:build` before testing them directly through Go.

Run `go tool task check` before finishing changes. Browser tests need Playwright's Chromium on a fresh checkout:

```sh
npm --prefix ui exec -- playwright install chromium
```

See [Taskfile.yml](../Taskfile.yml) for other commands and [AGENTS.md](../AGENTS.md) for contribution rules.

## Schemas and docs

When changing glossary or configuration types, regenerate and commit the schemas with `go tool task schema VERSION=__JARGONARY_VERSION__`. Preserve older version directories.

The docs build substitutes the manifest version, including the `v` prefix, into version placeholders even inside code blocks. README and runnable examples need real version URLs; release-please updates those in its PR.

Preview docs with `go tool task docs:preview` at `http://127.0.0.1:4321/jargonary/`. Set `PAGES_BASE_URL` to test a different deployment base path.
