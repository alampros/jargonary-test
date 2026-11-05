---
title: Getting started
editUrl: https://github.com/alampros/jargonary/edit/main/docs/getting-started.md
---

Run the sample configuration first, then replace its sources with your teams' glossaries.

## Try the sample with Docker

Clone the repository and start Docker Compose:

```sh
git clone https://github.com/alampros/jargonary.git
cd jargonary
docker compose up --build
```

Open [localhost:8080](http://localhost:8080/). The sample loads company, Web Development, and Engineering glossaries. Search for **API** to see how different teams' meanings remain distinct.

The default SQLite cache is in memory. See [configuration and deployment](reference.md#container) for persistent storage.

## Create your first glossary

Save this as `terms/payments.yaml`:

```yaml
# yaml-language-server: $schema=https://alampros.github.io/jargonary/schema/v1/glossary.schema.json
version: "1"
name: Payments
entries:
  - id: payment-intent
    term: Payment Intent
    senses:
      - id: payment-attempt
        definition: A record tracking a customer's attempt to pay.
        context: Payments
```

Save this as `config.yaml` next to the `terms` directory:

```yaml
site_name: Company glossary
sources:
  - id: payments
    file: terms/payments.yaml
```

Download a binary for your platform from [Releases](https://github.com/alampros/jargonary/releases), extract it, and run:

```sh
./jargonary -config config.yaml
```

Open [localhost:8080](http://localhost:8080/) and search for **Payment Intent**. Stop the earlier Compose instance with `docker compose down` before starting the binary on the same port.

Relative source paths resolve against the configuration file's directory. The [glossary format guide](dictionary-format.md) covers multiple meanings, aliases, relationships, and CSV input.

## Connect another team

Add a local file or an explicitly configured HTTP(S) endpoint:

```yaml
sources:
  - id: payments
    file: terms/payments.yaml
  - id: platform
    url: https://platform.example.com/.well-known/glossary
```

Restart after changing configuration. Source content is polled automatically, so editing a configured glossary needs no restart.

`/.well-known/glossary` is a proposed project convention, not a registered standard or an automatic discovery mechanism. Any configured HTTP(S) URL can supply a glossary.

## Next steps

Use [JSON Schema](schemas.md) for editor assistance, [customize the branding](customization.md), or read the [operator reference](reference.md) for protected sources, named exports, health checks, and MCP access.
