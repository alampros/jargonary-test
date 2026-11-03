# Project plan

Status: initial draft for review. The product name is undecided; the repository name is provisional.

Project terminology is defined in [GLOSSARY.md](GLOSSARY.md), with a machine-readable draft in [glossary/project.json](glossary/project.json). Use those definitions when interpreting this plan; unresolved schema and implementation choices remain proposals.

## Implementation progress

The initial headless core is implemented: Go/chi/Huma APIs, optional operator API on a separate listener, SQLite/FTS5, strict JSON/YAML/CSV inputs, persistent scraping, filtered authenticated exports, deprecation and relationship traversal, the dogfood endpoint, and plain definition HTML with text preview metadata. Docker/Compose and GitHub Actions workflows are prepared; local container execution still needs a running Docker daemon. Nothing has been published to GHCR.

The structural schema and initial source/export contract are in [schema/dictionary.schema.json](schema/dictionary.schema.json) and [docs/dictionary-format.md](docs/dictionary-format.md). Implemented defaults are in [README.md](README.md). Canonical entry/sense IDs are explicit. Source documents declare forward relationships; API inverses are generated. Exports use deterministic source-qualified hashed entry IDs with audience-safe origin metadata and rewritten references. Exports selecting imported export content are rejected in v1. Configuration, discovery, and secrets resolve at startup; restart applies changes.

The selected editorial public design is now implemented in React/Vite with a responsive alphabet rail/strip, real search and source filters, entry/sense pages, and copyable links. The separate read-only operator UI exposes source histories, resolved configuration, and diagnostics. Go embeds the frontend production build and keeps server-rendered metadata/readable definition content. Docker and CI now build the frontend first. The design brief and evidence are in [docs/design/brief.md](docs/design/brief.md) and [design-qa.md](design-qa.md). Preview images and MCP/skills remain future phases; further UI iteration can follow the initial browser review.

## Product

Build an MIT-licensed, self-hosted company dictionary. The distributed product is a Docker container image published to GitHub Container Registry (GHCR). There is no hosted service, billing, or SaaS tenancy.

Anyone who can reach an installation can browse and search its published vocabulary without an account. Companies can place the installation on their internal network or VPN. Internet-facing installations expose their published vocabulary publicly.

Applications and departments own their vocabulary sources. Jargonary (working name) periodically scrapes structured dictionaries and aggregates them into a searchable, shareable web app. Editing permissions and approval processes belong to the source systems.

## Agreed direction

- Go backend using chi and Huma v2, with an OpenAPI-documented REST API.
- SQLite for v1. Storage is a persistent, rebuildable cache of authoritative sources.
- Sources can be local JSON, YAML, or CSV files, or configured HTTP(S) URLs.
- Each remote source returns a dictionary directly. There is no discovery manifest or list of dictionary URLs inside the document.
- Instances can publish multiple named, filtered exports at `/.well-known/glossary/{exportId}`. Other instances consume these as ordinary HTTP sources. Filtering happens on the publishing instance.
- Protected exports initially use a simple pre-shared secret/password. Other authentication mechanisms, including OAuth2 client credentials, are future work.
- Proposed conventional publishing endpoint: `/.well-known/glossary`, returning JSON directly. Its specification and registration require follow-up; this is not yet an established standard.
- Multiple senses per entry, aliases that may target individual senses, and explicit relationships between entries or senses.
- Deprecation preserves discoverable vocabulary and may recommend replacements through `deprecatedFor`; deletion and export withdrawal remove content from the relevant published view.
- API relationships appear as navigable HATEOAS links, including derived reverse relationships.
- React/TypeScript/Vite is the proposed frontend, with server-generated definition metadata for link previews.
- Two separate UIs and HTTP listeners: the public UI defaults to port 8080; an optional admin/debug UI defaults to port 9090 and is disabled unless explicitly enabled in configuration.
- A dedicated product design stage precedes final UI implementation.
- MCP and skills are a later phase. Preserve the architectural intent without specifying the protocol surface now.

## Initial scope and boundaries

The first release provides source configuration, scraping, validation, search, alphabetical browsing, definition pages, source attribution, scrape health, REST access, machine-readable dictionary publishing, and named filtered exports for instance-to-instance sourcing.

It has no user accounts, roles, OIDC setup, internal approval queue, or in-app editor. The optional admin/debug UI initially provides read-only operational insight, not editing or configuration mutation. A later optional editor can manage a writable local source through the same ingestion pipeline. Its access control will be designed in that phase. Machine authentication for export feeds is separate from interactive user authentication and requires no user accounts or external identity provider.

Semantic search, additional database engines, automatic target discovery, arbitrary HTML extraction, comments, reactions, and onboarding quizzes are outside v1.

## Vocabulary model

### Dictionary source

Configuration assigns each source a stable ID. A source publishes a versioned document containing a name and entries. The source ID namespaces entry IDs, so matching text or IDs in two sources never causes an automatic merge.

Configured source labels can add context such as department or application. Source-controlled fields must not override scraper credentials or fetch policy.

### Entry, sense, and alias

- An entry has a stable ID, a preferred term, optional aliases, and one or more senses.
- A sense has an ID stable within its entry, a definition, optional context, examples, tags, links, and relationships.
- An alias has text and may be restricted to a particular sense. An unrestricted alias applies to the entry as a whole.
- Term and alias text need not be unique. Search collects matching senses with context and attribution.
- Matching text across sources does not establish identity or equivalence.

Required content is an entry ID, term, and at least one sense with an ID and definition. Explicit IDs are required for the canonical format so renames do not break references. CSV maps rows to senses and groups them by entry ID.

### Deprecation, deletion, and withdrawal

These are distinct operations:

- **Deprecation:** an entry or sense remains published, searchable, addressable, and available through REST and exports, but is marked as no longer recommended. Optional replacement relationships guide readers to current terminology. Deprecation alone never removes vocabulary.
- **Deletion:** an entry, sense, alias, or relationship is absent from a successfully validated replacement snapshot and is removed from that source's current content. This does not create a deprecated entry or imply a replacement.
- **Export withdrawal:** content no longer matches an export's selector and disappears from that export, while it may remain available on the publishing instance. A consumer applies the removal on its next successful refreshed snapshot.

Proposed schema: entries and senses may carry `deprecated: true` and an optional `deprecationReason`. An entry-level flag applies to every sense; a sense-level flag affects only that meaning. Missing flags mean no explicit deprecation. A `deprecatedFor` relationship also marks its originating entry or sense deprecated, even without an explicit flag. It can point to one or more replacement entries or senses. Deprecation without a replacement is valid. The API exposes effective deprecation status while exports preserve the explicit flags and relationships.

Deprecated content retains its IDs and URLs and appears in ordinary search with a clear deprecation indicator and any resolved replacement links. Direct lookup of an exact deprecated term must remain useful. Do not automatically redirect to replacements, which may have different meanings. Removing a deprecation flag and its outgoing `deprecatedFor` relationships restores active status unless entry-level deprecation still applies.

### Relationships

Relationships can originate at entry or sense level and target an entry or a specific sense. A reference contains an entry ID, an optional sense ID, and an optional source ID; omitted source means the current source.

Initial proposed vocabulary:

| Forward relation | Reverse relation | Meaning |
| --- | --- | --- |
| `abbreviationOf` | `abbreviatedBy` | A shortened form of the target |
| `relatedTo` | `relatedTo` | A symmetric association |
| `deprecatedFor` | `supersedes` | The origin is deprecated in favor of the target; the target supersedes it |

Publishers declare a relationship once. Ingestion derives inverse edges and deduplicates equivalent declarations. Aliases remain lookup names, distinct from edges between independently defined concepts.

Unresolved references produce diagnostics while leaving the definition usable. The API retains reference metadata but emits a navigable target link only when the target resolves. Resolution is reevaluated when any source changes.

Derived `supersedes` links do not deprecate their origin. Missing, excluded, or unresolved replacement targets must not reactivate deprecated content. When export filtering prunes a `deprecatedFor` target, serialize the originating entry or sense with an explicit deprecation flag so its status survives without disclosing the excluded target. Diagnose self-replacement and replacement cycles; traversal must be bounded and no replacement chain is followed as an automatic redirect.

## Example canonical document

This example is a proposed schema, not a frozen contract. It exercises multiple meanings, a sense-specific alias, an abbreviation, and a cross-source reference.

```json
{
  "version": "1",
  "name": "Engineering",
  "entries": [
    {
      "id": "component",
      "term": "Component",
      "senses": [
        {
          "id": "asset-part",
          "definition": "A constituent part of an asset system.",
          "context": "Asset management"
        },
        {
          "id": "web",
          "definition": "A reusable web interface element.",
          "context": "Frontend engineering"
        }
      ]
    },
    {
      "id": "asset-component",
      "term": "Asset component",
      "aliases": [{ "text": "Component", "sense": "physical" }],
      "senses": [
        {
          "id": "physical",
          "definition": "A separately tracked physical part of an asset.",
          "relationships": [
            {
              "type": "relatedTo",
              "target": { "entry": "component", "sense": "asset-part" }
            }
          ]
        }
      ]
    },
    {
      "id": "pi",
      "term": "PI",
      "senses": [
        {
          "id": "payments",
          "definition": "Short for Payment Intent in the payments API.",
          "relationships": [
            {
              "type": "abbreviationOf",
              "target": {
                "source": "payments",
                "entry": "payment-intent",
                "sense": "payment-attempt"
              }
            }
          ]
        }
      ]
    }
  ]
}
```

An identical meaning generally belongs in one entry with aliases. The component examples deliberately show that separate publishers or authors can retain distinct entries with overlapping terminology.

## Configuration and scraping

Proposed configuration:

```yaml
server:
  listen: 0.0.0.0:8080
admin:
  enabled: false
  listen: 127.0.0.1:9090
scrape_interval: 5m
sources:
  - id: payments
    url: https://payments.internal/.well-known/glossary
  - id: engineering
    url: https://engineering.internal/glossary.yaml
    format: yaml
    scrape_interval: 1m
  - id: company
    file: /terms/company.csv
```

For the simplest deployment, mounting supported files under `/terms` is sufficient; explicit configuration is optional. Discovery assigns deterministic source IDs from relative paths. Renaming a discovered file changes its identity; explicit IDs are recommended when stable external references matter. Publish exact discovery rules before implementation.

Scraping behavior:

1. Fetch or read with bounded time and size, then parse and validate the complete source.
2. Reject invalid snapshots and retain the last successful version. Unknown relationship targets are warnings rather than snapshot failures.
3. Commit a valid replacement and its search updates atomically. Readers never see partial imports.
4. A successful snapshot removes entries, senses, aliases, and relationships absent from that snapshot. A timeout, authentication failure, HTTP error (including 404/410), invalid document, or missing local file does not delete cached content. An explicitly valid empty dictionary clears a source. Retain the last successful snapshot indefinitely without age-based expiry; scrape freshness is reported separately from vocabulary deprecation.
5. Use conditional HTTP requests where supported. A not-modified response refreshes successful-check time without rebuilding the index.
6. Apply bounded backoff and jitter after failures. A periodic local scan is sufficient initially.
7. Track last attempt, last successful check, last content change, error status, and entry count separately.

Store last successful snapshots persistently so restarts can serve vocabulary during upstream outages. Define explicit source removal semantics before implementation; removing a source from configuration should deliberately retire its cached entries.

Remote sources may use internal addresses. Only operator-controlled configuration selects targets. Define redirect, credential-forwarding, outbound-network, TLS, and size-limit behavior explicitly. Support secret-file or environment references for private-source credentials; never expose credentials or sensitive URL parameters through the public API, diagnostics, or logs.

## Application architecture

Use a modular Go application with separate packages for configuration, source adapters, canonical validation, synchronization, storage, search, and application services. REST, HTML metadata routes, and future MCP call shared application services.

### Public and admin/debug interfaces

Run two independent HTTP listeners in the same application process, with separate routers and route sets:

- **Public UI:** enabled by default on `0.0.0.0:8080`. Serves dictionary browsing, search, definition pages, public REST, previews, and publishing/export endpoints with their configured access policies.
- **Admin/debug UI:** enabled only by an explicit configuration flag, defaulting to `127.0.0.1:9090` when enabled. Serves operational diagnostics and its own backing API. When disabled, no admin listener starts and no admin routes are registered on the public listener. Ports and bind addresses are configurable; conflicting listener configurations fail clearly.

Initial admin/debug capabilities are read-only:

- Inspect resolved configuration after defaults and overrides, including where settings originated. Show secret references or redacted placeholders, never resolved secrets, authorization headers, or credential-bearing URLs.
- Inspect discovered/configured sources, schedules, running scrapes, retry/backoff state, last attempt, last successful check, last content change, accepted snapshot counts, and bounded recent scrape outcomes.
- Explain fetch, parse, validation, reference-resolution, and snapshot-commit failures with sanitized diagnostics. Do not retain raw response bodies by default.
- Inspect export selector/access-policy summaries without exposing credentials. Detailed source locations and operational diagnostics belong here, not in public dictionary responses.

The public UI may show audience-safe source attribution and freshness. Full resolved configuration and detailed scrape diagnostics are admin-only. No public route or frontend asset bundle should expose admin configuration payloads or diagnostics as a fallback.

A separate port is a deployment separation, not authentication. With no built-in user accounts in v1, the admin listener is intended for trusted operator access via loopback, a private network, or an authenticated reverse proxy. Default deployment examples do not publish port 9090. For Docker access from the host, operators explicitly enable admin, bind it to a container-reachable address, and publish the host port on loopback (for example `127.0.0.1:9090:9090`); document that container-loopback binding alone is not reachable through ordinary port publishing.

Reserve this interface for future administrative/configuration tasks, such as a local-source editor or configuration management. Design access control and mutation protections before adding write operations; enabling diagnostics does not enable those future capabilities.

### Storage and search

SQLite holds source snapshots, entries, senses, aliases, relationship edges, synchronization state, and a full-text index. Use structured indexes for exact term/alias lookup and SQLite FTS5 for textual search. Select and validate a Go SQLite driver with FTS5 support early, including implications for container builds and supported architectures.

Search prioritizes exact terms and aliases, followed by prefixes and relevant definition matches. Preserve punctuation-sensitive names and short initialisms in exact matching. Define Unicode normalization and case matching consistently. Context/source filters and relationship traversal complement text search.

Vectors are a possible later search enhancement. Source data and explicit relationships remain authoritative; embeddings are a rebuildable index. Do not require an embedding provider, model download, or separate vector service in v1.

## REST, publishing, and previews

Use Huma v2 with chi to expose versioned public REST endpoints for search, entries, senses, and audience-safe source attribution/freshness, with generated OpenAPI documentation. Configuration and detailed scraping APIs belong exclusively to the optional admin listener and its separate API documentation.

Represent explicit relationships as typed data plus `_links` carrying navigable URLs. Include `self`, parent entry links, forward relationships, and derived inverse relationships. Clients must not need to construct relationship target URLs. Specify the exact link representation and custom relation semantics during the API milestone.

Use stable IDs for canonical links; display-name changes do not change identity. Configure the public base URL when generating absolute links and preview metadata, including deployments behind a reverse proxy or under a path prefix.

Publish the application's own glossary at `/.well-known/glossary`, directly in the canonical JSON format. Maintain that glossary as a repository source and validate it through the normal pipeline. Keep that endpoint separate from optional named exports of installation vocabulary at `/.well-known/glossary/{exportId}`. Every endpoint returns a canonical JSON dictionary directly; the base endpoint is not an export directory or an implicit aggregate of company vocabulary. Document the additional path components in the proposed well-known specification.

The conventional well-known path is at the origin root. Subpath deployments must support explicit source URLs and document reverse-proxy mapping if operators want the conventional endpoint.

Go serves embedded frontend assets and generates entry/sense-specific HTML metadata before React loads. Include canonical URL, title, description, and preview image metadata. Verify previews from the initial HTML without executing JavaScript. Crawlers still need network access to the installation.

## Named exports and instance-to-instance sourcing

A company can maintain its complete vocabulary in a private instance and publish an approved subset through a public instance, without maintaining duplicate source definitions. The private instance constructs a filtered export; the public instance fetches only that export. Private definitions must never be fetched into the public instance and then filtered there.

An instance may define multiple exports for different audiences, such as public and partner vocabulary. Each has a stable ID used as one URL path segment, a display name, an explicit inclusion selector, and an independent access policy. Changing the display name does not change the endpoint. No exports exist unless configured, and missing selectors are invalid; publishing everything requires an explicit `include: all`.

Proposed configuration on the private publishing instance:

```yaml
exports:
  - id: public
    name: Public company glossary
    include:
      sense_tags:
        any: [public]
    auth:
      shared_secret:
        env: PUBLIC_EXPORT_SECRET
  - id: partners
    name: Partner glossary
    include:
      sense_tags:
        any: [public, partner]
    auth:
      shared_secret:
        env: PARTNER_EXPORT_SECRET
```

The public consuming instance uses the existing HTTP source configuration; no special `upstream` source type is needed:

```yaml
sources:
  - id: company
    url: https://glossary.internal/.well-known/glossary/public
    auth:
      shared_secret:
        env: PRIVATE_GLOSSARY_EXPORT_SECRET
```

The publisher's `PUBLIC_EXPORT_SECRET` and consumer's `PRIVATE_GLOSSARY_EXPORT_SECRET` contain the same operator-provisioned secret. Environment variable names are local to each deployment. Secret-file references are an alternative to environment references, not a required deployment pattern. The examples are proposed configuration syntax, not a frozen contract.

### Filtering and publication boundary

- Initially support simple sense-tag inclusion: `any` matches at least one listed tag. An optional `sources` allowlist combines with tag matching using AND. Omitting the allowlist trusts all configured source owners to publish through matching tags; configure it when that delegation is inappropriate. Arbitrary query expressions are outside the initial scope.
- Export only matching senses and omit entries with no remaining senses. Remove aliases restricted to excluded senses. Entry terms and unrestricted aliases are shared entry metadata; authors must consider them published when any sense is included. Finalize handling of all other shared fields before implementation.
- Remove relationships to excluded or unresolved targets, including target reference metadata and derived inverse links. Recompute navigable relationships within the exported view. This is stricter than the ordinary REST behavior that retains unresolved references.
- Export names, attribution, links, and provenance must be suitable for the audience. Do not expose internal hostnames, source fetch URLs, credentials, private diagnostics, or counts of excluded vocabulary.
- Structural filtering cannot determine whether publisher-authored definitions, examples, or links contain confidential information. Content approval remains the source owner's responsibility; use audience-safe source IDs because origin attribution includes them.
- Filtering must precede serialization and construction of export-specific metadata. Conditional responses and cache validators must describe the filtered representation, and authenticated responses must not enter shared caches that bypass export access checks.
- Adding a publication tag is a publishing action governed by the source owner's review process. Export access authorizes fetching that subset; it does not grant access to the private instance's other routes or bypass its network boundary.

### Initial authentication and future mechanisms

For v1, a protected export uses a simple pre-shared secret/password supplied by the operator on both instances. The consumer sends it in an `Authorization: Bearer` header over HTTPS; this is a static shared credential, with no OAuth token endpoint, token exchange, or user login. Each export checks its own configured secret. Never put secrets in URLs, export documents, public API responses, or logs, and never forward them to another origin on redirects. Define rotation and configuration reload behavior during the configuration milestone.

An export can explicitly opt into unauthenticated access when public access or network controls are sufficient; omitting an access policy must not silently make it public. The authentication configuration should accommodate additional mechanisms later, including OAuth2 client credentials with an existing authorization server. OAuth integration, token issuance, and interactive identity management are outside v1.

### Identity, propagation, and withdrawal

Exports must preserve stable entry/sense identity, source attribution, and relationships across the export boundary. Finalize the canonical representation for multiple originating sources and reference rewriting before implementation; do not flatten colliding entry IDs or merge terms by text. Track provenance without publishing sensitive source configuration. Arbitrary chains, duplicate import paths, and cyclic instance sourcing require explicit rules; do not enable unrestricted re-export chains before those rules exist.

A successful refreshed snapshot removes vocabulary withdrawn from an export. Authentication failure, upstream outage, a missing/deleted export endpoint, or invalid content retains the last successful snapshot indefinitely, including across restarts. There is no age-based expiry in v1. Removing a publication tag propagates on the next successful refreshed snapshot; revoking a credential prevents further fetches but does not delete previously imported content. To clear a consumer through synchronization, publish a valid empty dictionary while it can still fetch the export. Explicit operator removal of a configured source is a separate local removal action.

Apply successful removals consistently to search, REST, definition pages, generated previews, and downstream exports. Deprecation is an ordinary content update and remains visible through all those surfaces, subject to export selectors. Neither deletion, withdrawal, nor credential revocation recalls external copies or search-engine caches.

## Product design stage

Design around fast lookup and readable definitions: prominent search, keyboard navigation, alphabetical browsing, numbered senses, context labels, source attribution, aliases, navigable relationships, and stable share links. Include empty, loading, ambiguous-result, stale-source, unresolved-reference, and deprecated-entry/sense states. Clearly distinguish stale source data from deprecated terminology and show replacement guidance without automatic redirects. Validate responsive behavior and accessibility before final implementation.

Design a separate operator experience for the admin/debug listener: resolved configuration, per-source scrape status, recent outcomes, and actionable validation diagnostics. Keep detailed operational information out of the public UI and cover both interfaces in design review.

## Container and delivery

- Multi-stage build: compile frontend assets, embed them, and build the Go application.
- One application container, public HTTP on port 8080 by default, optional admin/debug HTTP on port 9090, SQLite under a mounted `/data` directory, optional sources under `/terms`, and no required external services. Both listeners share the same process and storage; admin is disabled and its port unpublished in the default deployment.
- Run without root privileges. Provide health/readiness endpoints, graceful shutdown, structured logs, and documented volume permissions.
- GitHub Actions runs Go formatting checks, `go vet`, `go test`, frontend type/build checks, and relevant integration tests.
- Pull requests validate builds; authorized release workflows publish versioned GHCR images with commit/release provenance. Decide supported architectures after validating the SQLite driver.
- Include an MIT license, quick start, configuration reference, example dictionaries, backup/restore instructions, upgrade behavior, and a recovery/rebuild procedure.
- Do not assume an arbitrary live copy of a SQLite file is a consistent backup. Document a supported backup method or stopped-container copy.

## Phases and acceptance criteria

### 1. Contracts and technical validation

Finalize canonical JSON schema, YAML mapping, CSV conventions, identity/reference rules, validation policy, and configuration defaults. Specify named export selectors, shared-secret authentication, multi-source export identity, relationship pruning, and cycle/withdrawal policies. Validate SQLite/FTS5 and Go-served preview metadata with small technical probes. Document the well-known convention and registration follow-up. Resolve the open decisions below.

Acceptance: fixtures represent ambiguous terms, sense-specific aliases, punctuation, initialisms, inverse relationships, cross-source references, and entry/sense deprecation with and without replacements; the chosen SQLite build supports the required search behavior.

### 2. Headless dictionary core

Implement local and remote sources, persistent snapshots, atomic updates, search, relationship resolution, REST/HATEOAS, the dogfood endpoint, and named filtered exports with shared-secret access. Add Docker packaging and CI early so subsequent work is tested in the deployment environment.

Implement the optional admin listener and its read-only configuration/scrape diagnostics API, with no corresponding routes on the public listener.

Acceptance: failed scrapes preserve content across restarts; valid snapshots add/change/delete entries correctly; unchanged responses avoid unnecessary rebuilds; forward and reverse links resolve; search returns all relevant senses with attribution; no credentials appear in public outputs; one instance can consume another instance's filtered export; export credentials cannot access other exports; excluded senses, aliases, relationship targets, and private source metadata do not leak; successful export refreshes propagate withdrawals.

Additional lifecycle acceptance: long outages, authentication failures, and missing export endpoints never age out cached content; valid empty snapshots clear it; deprecated entries remain searchable at stable URLs; sense-only deprecation does not deprecate unrelated senses; `deprecatedFor` and `supersedes` are navigable when resolved; pruning replacement targets preserves deprecation without leaking target metadata; successful removals reach all serving surfaces.

Listener acceptance: defaults start only the public listener; explicitly enabling admin starts its separate listener; admin UI/API routes remain absent from the public router; resolved configuration and diagnostics redact secrets; the documented Docker access pattern works without exposing admin on all host interfaces.

### 3. Product design

Produce and review the public browse, search, and definition experiences and the separate admin/debug configuration and scraping experience before final UI construction.

Acceptance: a reviewed design covers desktop/mobile use, keyboard access, ambiguous meanings, attribution, and failure states.

### 4. Web app and release

Build both UIs, server-generated public previews, operational documentation, release workflow, and GHCR publication configuration. Serve each UI and its backing API from its own listener.

Acceptance: a fresh installation works with a mounted source directory and data volume; remote source configuration works without accounts or external services; direct definition links render correct initial metadata; the container passes end-to-end smoke checks and can restart with persisted content.

Actual registry publication requires the repository's GitHub Actions permissions and release setup; preparing the workflow is distinct from publishing an image.

### 5. MCP and skills

Add an MCP server for vocabulary search, entry/sense lookup, and relationship traversal using shared application services. Provide skills for explaining company terminology, resolving ambiguity, citing sources, and preparing proposed additions to source files. Select transports, authorization, supported clients, and packaging in this phase. Do not include MCP implementation in the first release milestone.

### Later possibilities

An optional editor for a writable local source; semantic/hybrid search; additional source integrations; additional database engines only if justified. Additional source/export authentication mechanisms, including OAuth2 client credentials, are also planned for later. Each requires a separate scope decision.

## Open decisions for review

- Final product name.
- Canonical schema field names and minimum relationship types; whether unrestricted custom relationship types belong in v1.
- CSV representation of aliases, examples, and relationships without making basic files cumbersome.
- Default scrape interval, failure backoff, stale indication, and source removal/reload behavior.
- Exact REST routes, link relation conventions, and public diagnostic fields.
- Final export configuration syntax, shared metadata publication rules, multi-source identity/provenance representation, duplicate and cycle handling, and credential rotation/reload. Last-success retention without age-based expiry is settled for v1.
- React/Vite with Go-generated metadata is implemented; full React SSR is not required for the initial UI.
- Whether the dogfood endpoint is always enabled or configurable; the default UI should not mix product documentation terms into company vocabulary unexpectedly.
- Well-known suffix registration process and publication of the independent format specification.

## Reference material

- [Huma chi integration](https://huma.rocks/features/bring-your-own-router/)
- [SQLite FTS5](https://www.sqlite.org/fts5.html)
- [RFC 8615: Well-Known URIs](https://www.rfc-editor.org/rfc/rfc8615.html)
- [IANA Well-Known URI registry](https://www.iana.org/assignments/well-known-uris/)
- [Official Go MCP SDK](https://go.sdk.modelcontextprotocol.io/)
