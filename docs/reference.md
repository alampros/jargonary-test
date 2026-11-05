---
title: Configuration and deployment
editUrl: https://github.com/alampros/jargonary/edit/main/docs/reference.md
---

Jargonary is one process: a bundled web UI, configured glossary sources, and a SQLite cache. Run a binary or container, point it at your sources, and open port 8080.

## Configuration

Save a `config.yaml`:

```yaml
# yaml-language-server: $schema=https://alampros.github.io/jargonary/schema/__JARGONARY_VERSION__/config.schema.json
site_name: Company glossary
sources:
  - id: platform
    file: terms/platform.yaml
  - id: payments
    url: https://payments.internal/.well-known/glossary
```

Each source needs a unique ID and either a local file or an HTTP(S) URL. JSON, YAML, and CSV are supported. Relative file paths resolve from the config file's directory.

Sources refresh every five minutes by default. Failed fetches keep the last successful content. Restart after changing configuration; editing glossary content needs no restart.

The [annotated example](../examples/config.yaml) lists the available settings. The [configuration JSON Schema](schemas.md#configuration-schema) provides editor completion and structural validation. See [source settings](sources.md) for polling, file discovery, and protected endpoints.

## Run a binary

Download and extract a binary from [Releases](https://github.com/alampros/jargonary/releases), then run:

```sh
./jargonary -config config.yaml
```

Open [localhost:8080](http://localhost:8080/).

## Container

With `config.yaml` and a `terms/` directory in your working directory:

```sh
docker run --rm -p 8080:8080 \
  -v "$PWD/config.yaml:/config/config.yaml:ro" \
  -v "$PWD/terms:/config/terms:ro" \
  ghcr.io/alampros/jargonary:dev -config /config/config.yaml
```

This uses the development image; pin a release tag or digest for deployment. The container runs as UID/GID **65532:65532**; mounted files must be readable by that user.

To try the repository's sample data instead, run `docker compose up --build`.

## Keep the cache between restarts

By default, the cache is in memory and refills from sources after a restart. To persist it, add:

```yaml
database: /data/glossary.db
```

In a container, mount a writable persistent volume at `/data`. Its directory must be writable by UID/GID 65532. Run one instance per database file.

## Put it behind your proxy

Use your reverse proxy or hosting platform for HTTPS and user login. Set `public_url` to the external URL, such as `https://glossary.example.com`.

Shared secrets protect individual source fetches and exports; they do not add login protection to the public UI. Optional operator and MCP listeners have no built-in authentication and should stay private.

## Check that it is running

`/healthz` checks the HTTP process; `/readyz` checks database access. Sources load asynchronously, so readiness does not mean every glossary has loaded. Logs go to stdout.

For more specific tasks:

- [Publish selected content](publishing.md) to another team or instance.
- [Check source status and APIs](operations.md) when troubleshooting.
- [Change branding and themes](customization.md).
- [Build or contribute](development.md).
