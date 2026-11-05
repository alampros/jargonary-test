---
title: Connect an AI agent
editUrl: https://github.com/alampros/jargonary/edit/main/docs/mcp.md
---

Enable the read-only [MCP](https://modelcontextprotocol.io) listener:

```yaml
mcp:
  enabled: true
  listen: 127.0.0.1:9091
```

Restart, then connect a **Streamable HTTP** client to `http://127.0.0.1:9091/`. Use the root URL with its trailing slash. The server is stateless; persistent sessions are unnecessary.

## Tools

| Tool | Purpose |
| --- | --- |
| `list_glossaries` | Sources and entry/sense counts |
| `search` | Terms, aliases, and definitions |
| `list_meanings` | Meanings across sources, grouped by entry |
| `browse_glossary` | Paginated source entries |
| `get_entry` | Entry and its senses |
| `get_sense` | One meaning |
| `get_relationships` | Incoming/outgoing references |
| `resolve_term` | Entry with related terms resolved |

Tools read the same cache as the UI. Reuse returned IDs and retain source IDs: entry IDs alone are not globally unique.

## Access

MCP is disabled by default and has **no built-in authentication**. Keep it private or use an authenticated HTTPS proxy. Source/export secrets do not protect MCP.

In Docker, listen on `0.0.0.0:9091` and publish `127.0.0.1:9091:9091` for host access. Container-loopback listeners cannot be reached through ordinary port publishing.
