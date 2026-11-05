---
title: Getting started
editUrl: https://github.com/alampros/jargonary/edit/main/docs/getting-started.md
---

## Try the sample

```sh
git clone https://github.com/alampros/jargonary.git
cd jargonary
docker compose up --build
```

Open [localhost:8080](http://localhost:8080/) and search for **API** to see distinct meanings from the sample teams.

## Use your own glossary

Save `terms/payments.yaml`:

```yaml
# yaml-language-server: $schema=https://alampros.github.io/jargonary/schema/__JARGONARY_VERSION__/glossary.schema.json
version: "1"
name: Payments
entries:
  - id: payment-intent
    term: Payment Intent
    senses:
      - id: payment-attempt
        definition: A record tracking a customer's attempt to pay.
```

Save `config.yaml` beside the `terms` directory:

```yaml
sources:
  - id: payments
    file: terms/payments.yaml
```

Stop the sample with `docker compose down`, then run a [release binary](https://github.com/alampros/jargonary/releases):

```sh
./jargonary -config config.yaml
```

Source paths resolve relative to the config file. Restart after changing configuration; glossary content refreshes automatically.

See [source settings](sources.md) to add files or HTTP endpoints, [the format guide](dictionary-format.md) for richer definitions, and [deployment](reference.md) for containers and persistence.
