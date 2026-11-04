# Jargonary

**A glossary defines words in their context.** Each team owns its vocabulary; Jargonary brings those sources together into one searchable view while keeping their meanings distinct.

In one company, a developer might use **API** to mean an application programming interface, while an engineer uses it to mean the American Petroleum Institute. Both meanings belong in the glossary, with their context intact.

## Start here

- [Getting started](getting-started.md): run Jargonary and connect your first glossary.
- [Glossary format](dictionary-format.md): author terms, meanings, aliases, and relationships in JSON, YAML, or CSV.
- [Configuration and deployment](reference.md): configure sources, persistence, protected exports, and operational listeners.
- [Connect an AI agent](mcp.md): enable read-only glossary tools over MCP.
- [Branding and themes](customization.md): choose a theme or use your company's branding.
- [JSON Schema](schemas.md): add editor validation using stable schema URLs.
- [Development](development.md): build, test, and contribute.

## Team-owned vocabulary

Teams maintain glossary files or HTTP endpoints through their own editing and approval workflows. Jargonary polls configured sources, validates their data, and combines them for browsing and search. If a source goes offline, its last successful snapshot stays available.

Identical terms and entry IDs across sources retain separate identities. Named exports let you publish selected source content to another instance.

## Simple to deploy

Jargonary ships as a standalone Go binary or Docker container with a bundled UI and SQLite storage. No separate database service is required. TLS termination and user authentication belong in your proxy or hosting layer; shared secrets can protect source fetching and named exports.

[Browse the source](https://github.com/alampros/jargonary) or [download a release](https://github.com/alampros/jargonary/releases).
