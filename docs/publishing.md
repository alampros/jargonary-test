# Publish selected content

Named exports share selected company content at `/.well-known/glossary/{exportId}`. Teams keep ownership of editing and approval in their source systems.

## Configure an export

Add an export to your config and restart:

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

This publishes senses tagged `partner` from the `payments` source. Each export must explicitly choose a shared secret or `auth.public: true`. Secrets use the same [file or environment references](sources.md#protected-sources) as protected sources.

Configure another instance to fetch the export's full HTTPS URL as a source. For a runnable two-instance example:

```sh
docker compose -f compose.federation.yaml up --build
```

Open the upstream at `http://localhost:8081/` and downstream at `http://localhost:8080/`. See the [example configs](../examples/federation/).

## What gets shared

Exports preserve source identities and remove relationships to excluded targets while retaining effective deprecation. Terms and unrestricted aliases accompany any selected sense. The bundled `project` glossary is always excluded. Imported exports cannot be re-exported in v1.

Use both a source allowlist and tags when only certain source owners should publish to an audience. Filtering selects content; it does not redact definitions, examples, URLs, tags, or attribution. Review selected content for its audience.

## Availability and access

An export returns **503 until every selected source has an accepted snapshot**. Without a source allowlist, every configured source must have a snapshot, even when selecting by tag. An accepted empty glossary counts as ready; later fetch failures keep publishing the last successful content.

A protected export does not protect the public UI or other routes. To expose only an authenticated export, use the [export-only proxy example](../examples/export-only-proxy.conf). Revoking access prevents future fetches but cannot recall downloaded or cached copies.

The root `/.well-known/glossary` serves the bundled project glossary. Named exports serve your teams' content. This publishing path is a proposed project convention, not a registered standard or automatic discovery mechanism.
