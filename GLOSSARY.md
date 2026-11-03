# Project glossary

Draft vocabulary for discussing and building the product. These definitions use the current plan; the product name and schema are still provisional.

The machine-readable source is [glossary/project.json](glossary/project.json), using the proposed canonical format. This document is its reading companion; update both together. The JSON is intended to become the application’s own dictionary at `/.well-known/glossary` once the server exists.

Use **dictionary** (or **glossary**) for content, **source** for a configured input, and **target** for its location. Use **entry** for a record, **term** for its preferred name, **sense** for a meaning, and **definition** for the explanatory text.

Implementation proposals include JSON field names, `supersedes` as the reverse of `deprecatedFor`, and portable identity across exports. This glossary does not settle those contracts.

## Core vocabulary

### Dictionary

Also: Glossary.

An identified collection of entries describing vocabulary for a particular context. A source supplies a dictionary; an export produces a dictionary for a selected audience.

**Example:** The Engineering dictionary contains entries maintained by the Engineering team.

Related: [Source](#source), [Entry](#entry), [Export](#export).

### Entry

A record with a stable ID, a preferred term, optional aliases, and one or more senses. Its term text does not have to be unique.

**Example:** One Component entry has asset-management and frontend senses.

Related: [Term](#term), [Sense](#sense), [Alias](#alias).

### Term

A word or phrase used to name a concept. The term field is an entry's preferred display name; the same text may name different entries or meanings.

**Example:** Component is term text, not an identity key.

Related: [Entry](#entry), [Sense](#sense), [Stable ID](#stable-id).

### Sense

Also: Meaning.

One identifiable meaning within an entry, with its own definition and optional context, examples, tags, links, and relationships.

**Example:** The web-interface meaning of Component is one sense.

Related: [Entry](#entry), [Definition](#definition), [Context](#context).

### Definition

The explanatory text of a sense. A definition describes a meaning; it is not the complete entry or sense record.

**Example:** A reusable web interface element is a definition.

Related: [Sense](#sense).

### Alias

An additional lookup name for an existing entry, optionally restricted to one of its senses. It has no independent definition and need not be globally unique.

**Example:** Component can be an alias for a sense of Asset component while also naming another entry.

Related: [Entry](#entry), [Sense](#sense), [Abbreviation](#abbreviation).

### Abbreviation

A shortened form of another term. It may be an alias when it needs no independent definition, or a separate entry or sense connected by abbreviationOf.

**Example:** PI can be a separately defined abbreviation of Payment Intent.

Related: [Alias](#alias), [Relationship](#relationship).

### Context

Information that distinguishes where a sense applies, such as a department, discipline, or application.

**Example:** Frontend engineering distinguishes one meaning of Component.

Related: [Sense](#sense), [Tag](#tag).

### Tag

A label attached to a sense for organization and selection. Tags used by export selectors participate in publication decisions.

**Example:** A public tag can make a sense eligible for an explicitly configured export.

Related: [Sense](#sense), [Selector](#selector).

### Stable ID

Also: Identifier.

A reference key that remains unchanged when display text changes. Entry IDs are scoped to a source and sense IDs to an entry; portable export identity is still being designed.

**Example:** Renaming Payment Intent does not change its entry ID.

Related: [Source](#source), [Reference](#reference).

## Relationships

### Relationship

A typed connection from an entry or sense to another entry or sense. Reverse connections are derived where the relation defines an inverse.

**Example:** PI abbreviationOf Payment Intent is a relationship between independently identifiable records.

Related: [Reference](#reference), [Alias](#alias).

### Reference

Structured data identifying a relationship target by entry ID, optional sense ID, and optional source ID. An omitted source means the current source.

**Example:** A reference can select only the payments sense of PI.

Related: [Relationship](#relationship), [Stable ID](#stable-id).

### Unresolved reference

A reference whose target cannot be found in the current imported content. It produces a diagnostic, not removal of the originating definition.

**Example:** A replacement may remain unresolved while its source has not yet been imported.

Related: [Reference](#reference), [Deprecation](#deprecation).

### abbreviationOf

A relationship stating that the originating entry or sense is a shortened form of the target. Its reverse relation is abbreviatedBy.

**Example:** PI abbreviationOf Payment Intent; Payment Intent abbreviatedBy PI.

Related: [Abbreviation](#abbreviation), [Relationship](#relationship).

### relatedTo

A symmetric relationship associating two entries or senses without asserting that they have identical meanings.

**Example:** Asset component relatedTo Asset system.

Related: [Relationship](#relationship).

### deprecatedFor

A relationship marking the originating entry or sense deprecated in favor of a replacement target. Its derived reverse relation is supersedes; a missing target does not remove deprecation.

**Example:** An old application name can be deprecatedFor its new name.

Related: [Deprecation](#deprecation), [Relationship](#relationship).

## Sources and synchronization

### Instance

Also: Installation.

One deployed application with its own configuration, cached dictionaries, serving endpoints, and optional exports.

**Example:** A private instance and a public instance are separate deployments.

Related: [Source](#source), [Export](#export).

### Source

An operator-configured input identified by a stable local ID, read from a file or fetched from a URL, together with its format, schedule, and access settings. Source configuration is distinct from dictionary content.

**Example:** payments is a configured source that fetches a JSON dictionary from an internal application.

Related: [Dictionary](#dictionary), [Target](#target), [Snapshot](#snapshot).

### Target

Also: Scrape target.

The concrete URL or file path read by a scraper for a source. It is a location, not the source identity or the dictionary itself.

**Example:** https://payments.internal/.well-known/glossary is a target URL.

Related: [Source](#source), [Scrape](#scrape).

### Scrape

One attempt to read a source and check its dictionary. A fetch can succeed while parsing or validation fails; only a valid complete replacement changes published content.

**Example:** A scrape receiving malformed JSON fails and leaves the previous snapshot available.

Related: [Source](#source), [Snapshot](#snapshot), [Synchronization](#synchronization).

### Snapshot

A complete validated version of a source dictionary accepted for serving. A valid empty dictionary is a snapshot; a failed read is not.

**Example:** A new snapshot replaces the source's previous entries atomically.

Related: [Dictionary](#dictionary), [Last successful snapshot](#last-successful-snapshot).

### Last successful snapshot

The most recently accepted complete dictionary for a source, retained indefinitely through failed scrapes and restarts until replaced or explicitly removed locally.

**Example:** Revoking an export credential does not erase a consumer's last successful snapshot.

Related: [Snapshot](#snapshot), [Deletion](#deletion), [Stale source](#stale-source).

### Synchronization

Also: Sync.

The process of scraping, validating, and atomically replacing a source snapshot and its derived search and relationship data.

**Example:** A successful sync applies additions, changes, and removals together.

Related: [Scrape](#scrape), [Snapshot](#snapshot).

### Stale source

A source whose accepted content has not been successfully revalidated recently. Freshness describes synchronization health and does not mark its vocabulary deprecated or remove it.

**Example:** An unreachable source may be stale while its terms remain current and usable.

Related: [Last successful snapshot](#last-successful-snapshot), [Deprecation](#deprecation).

## Publication and lifecycle

### Export

Also: Named export.

A configured, named publication of selected vocabulary as a dictionary at a stable endpoint, with an explicit selector and access policy. A consumer treats that endpoint as an ordinary HTTP source.

**Example:** The public export includes approved senses from a private instance.

Related: [Dictionary](#dictionary), [Selector](#selector), [Publisher](#publisher), [Consumer](#consumer).

### Selector

Also: Inclusion selector.

An operator-defined rule choosing which senses enter an export. Matching senses carry permitted entry metadata; relationships to excluded targets are pruned.

**Example:** sense_tags.any: [public] selects senses carrying the public tag.

Related: [Export](#export), [Tag](#tag), [Withdrawal](#withdrawal).

### Publisher

The application or instance serving a dictionary for another instance to scrape. Filtering an export happens here, before the consumer receives the content.

**Example:** A private instance publishes only its selected public vocabulary to a public consumer.

Related: [Consumer](#consumer), [Export](#export).

### Consumer

An instance fetching a dictionary into one of its configured sources. Permission to fetch an export does not grant access to the publisher's other routes.

**Example:** A public instance consumes an authenticated export from a private instance.

Related: [Publisher](#publisher), [Source](#source).

### Deletion

Removal of content from a source's current dictionary. Consumers apply the removal when a successful replacement snapshot omits it; fetch failures and missing endpoints are not deletion instructions.

**Example:** Removing a sense from a valid dictionary deletes that sense at the next successful sync.

Related: [Withdrawal](#withdrawal), [Deprecation](#deprecation), [Snapshot](#snapshot).

### Withdrawal

Also: Export withdrawal.

Removal of previously included content from a particular export, without requiring its deletion from the publisher's underlying dictionary. Consumers apply it on a successful refreshed snapshot.

**Example:** Removing a public tag withdraws a sense from an export selecting that tag.

Related: [Deletion](#deletion), [Deprecation](#deprecation), [Export](#export).

### Deprecation

A published indication that an entry or sense is no longer recommended. It remains searchable and addressable and can name replacements through deprecatedFor. Entry-level deprecation applies to every sense.

**Example:** An old product name stays findable so readers can discover the preferred replacement.

Related: [deprecatedFor](#deprecatedfor), [Deletion](#deletion), [Withdrawal](#withdrawal).

## Interfaces

### Well-known glossary endpoint

The proposed conventional path /.well-known/glossary returning a dictionary directly as JSON. It is not a directory of source URLs. The path requires specification and registration follow-up.

**Example:** This project will serve its own glossary at the base endpoint; configured company exports use named child paths.

Related: [Dictionary](#dictionary), [Export](#export).

### HATEOAS

Hypermedia as the Engine of Application State. In this API, responses include navigable links to entries, senses, and resolved relationships so clients need not construct target URLs.

**Example:** An abbreviation response links directly to the expanded term and the reverse link is available there.

Related: [Relationship](#relationship), [Reference](#reference).

### Listener

Also: HTTP listener.

A configured network bind address and port accepting HTTP requests for a particular interface. Public and admin listeners share one process but use separate routers and route sets.

**Example:** The public listener defaults to 0.0.0.0:8080; the optional admin listener defaults to 127.0.0.1:9090.

Related: [Instance](#instance), [Public UI](#public-ui), [Admin/debug UI](#admindebug-ui).

### Public UI

The dictionary interface available to anyone who can reach the public listener, enabled by default on port 8080. It provides browsing, search, and definition pages with audience-safe attribution and freshness.

**Example:** A reader searches for PI without enabling or accessing the admin interface.

Related: [Listener](#listener), [Admin/debug UI](#admindebug-ui), [Dictionary](#dictionary).

### Admin/debug UI

Also: Admin UI, Debug UI.

An optional operator interface on a separate listener, defaulting to port 9090 and disabled unless explicitly enabled. Initially read-only, it exposes redacted resolved configuration and detailed scrape diagnostics. Separate networking does not itself authenticate operators; future write tasks need an access-control design.

**Example:** An operator enables admin on a trusted interface to investigate why a source failed validation.

Related: [Listener](#listener), [Public UI](#public-ui), [Resolved configuration](#resolved-configuration), [Scrape diagnostics](#scrape-diagnostics).

### Resolved configuration

Also: Effective configuration.

The settings actually used by an instance after defaults and configured overrides are applied. The admin/debug view includes setting origins and redacts secret values and sensitive URL components.

**Example:** An operator sees the effective scrape interval and its origin without seeing the export bearer secret.

Related: [Admin/debug UI](#admindebug-ui), [Source](#source).

### Scrape diagnostics

Operator-facing information about source synchronization, including schedules, running attempts, retry state, timestamps, bounded recent outcomes, and sanitized fetch or validation failures. Detailed diagnostics are served only by the optional admin listener.

**Example:** The admin UI explains a failed JSON parse while the public UI continues serving the last successful snapshot.

Related: [Admin/debug UI](#admindebug-ui), [Scrape](#scrape), [Snapshot](#snapshot), [Stale source](#stale-source).

## Export provenance

### Origin

Audience-safe attribution identifying the original source and entry of an exported record. It preserves provenance independently of the consumer's configured source ID and never includes source fetch credentials or URLs.

**Example:** An imported entry can record engineering/component as its origin while belonging to the consumer's company source.

Related: [Export](#export), [Stable ID](#stable-id), [Source](#source).

### Re-export

Publication through an export of content previously imported from another instance's export. The initial implementation rejects exports selecting such content, limiting automated instance-to-instance publication to one hop.

**Example:** A public consumer cannot automatically export an imported partner glossary to a third instance.

Related: [Export](#export), [Publisher](#publisher), [Consumer](#consumer).
