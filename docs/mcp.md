# Connect an AI agent

Jargonary's [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server lets an AI agent look up your teams' vocabulary. Tools are read-only and use the same cached glossaries as the web UI.

## Enable MCP

Add this to `config.yaml` and restart Jargonary:

```yaml
mcp:
  enabled: true
  listen: 127.0.0.1:9091
```

MCP is disabled by default. It runs on a separate listener from the public UI and operator console.

## Connect your client

In an MCP client that supports **Streamable HTTP**, add a server with:

| Setting | Value |
| --- | --- |
| Name | `jargonary` |
| Transport | Streamable HTTP |
| URL | `http://127.0.0.1:9091/` |

Use the listener's root URL, including the trailing slash. The server is stateless and does not require persistent sessions.

Try asking: **“What does API mean in each team's glossary?”** The agent can list available glossaries, look up the term, and follow related definitions. For a remote client, use the HTTPS URL of your authenticated proxy instead of localhost.

## Available tools

| Tool | Purpose |
| --- | --- |
| `list_glossaries` | List sources and their entry and sense counts |
| `search` | Search terms, aliases, and definition text |
| `list_meanings` | Look up meanings across sources, grouped by entry |
| `browse_glossary` | Page through a source's entries |
| `get_entry` | Read an entry and its senses |
| `get_sense` | Read one meaning |
| `get_relationships` | List incoming and outgoing cross-references |
| `resolve_term` | Read an entry with related terms resolved |

Use the source, entry, and sense IDs returned by tools in subsequent calls. The same entry ID can belong to different sources; keep the source ID with it.

## Access and containers

The MCP listener has **no built-in authentication**. Keep it private or put it behind an authenticated proxy. Source and export shared secrets do not protect MCP access.

For Docker access from the host, set `mcp.listen` to `0.0.0.0:9091` inside the container and publish the port with `127.0.0.1:9091:9091`. A loopback listener inside the container cannot be reached through ordinary port publishing.
