# recap

**Local-first memory & project-knowledge layer for coding agents.**

recap gives AI coding agents a memory that survives the blank-context-window problem —
auto-captured, private, and portable across tools. It ships as a Claude Code plugin
(MCP server + lifecycle hooks) and as a standalone local MCP server for Cursor and Codex.

> Status: **v0 functional, pre-release.** The core works locally — MCP server (`memory_*`
> tools), lifecycle hooks (auto-recall at session start), and a web viewer. Cross-tool
> adapters and packaged installers are next. See [`ROADMAP.md`](ROADMAP.md).

## Why recap is different

The "agent memory" space is crowded, so recap competes on a sharp wedge (full analysis in
[`decision.md`](decision.md)):

- **"Why" / rejected-approach capture** — a first-class record type so the agent stops
  re-pitching approaches you already ruled out.
- **Per-client isolation** — a `client` boundary above the repo. Client A's context can
  *physically* never surface in Client B's session (separate DB per client).
- **Cross-tool "Switzerland"** — one local store, thin adapters for Claude Code, Cursor,
  and Codex. Native vendor memory becomes a feed, not a competitor.
- **Zero-config & local** — SQLite + a small local embedding model. No API key, no cloud,
  private by default.

## Architecture at a glance

- **Go core** — MCP server (stdio), lifecycle hooks, storage, retrieval, web viewer.
- **SQLite** — `decision` / `rejected_approach` / `convention` / `session_summary`
  records, FTS5 keyword index + vector embeddings, fused via reciprocal-rank fusion.
- **Pluggable embeddings** — `Embedder` interface with a bundled Rust `fastembed-rs`
  sidecar (default), Ollama (if present), or FTS5-only fallback.

See [`docs/TECH.md`](docs/TECH.md) for the full design.

## Quickstart (local, from source)

Requires Go 1.26+.

```bash
go install github.com/sandeepshekhar26/recap/cmd/recap@latest   # or: go build -o recap ./cmd/recap
recap version
```

**Claude Code** — install as a plugin from this repo (registers the MCP tools + hooks; needs `recap` on `PATH`):

```
/plugin marketplace add sandeepshekhar26/recap
/plugin install recap@recap
```

**Cursor / Codex** — register the MCP server manually (e.g. `~/.cursor/mcp.json`):

```json
{ "mcpServers": { "recap": { "command": "recap", "args": ["serve"] } } }
```

**Browse / prune your memory:**

```bash
recap viewer        # http://127.0.0.1:37788
```

**Per-client isolation** — map directories to clients in `~/.recap/config.json`:

```json
{ "clients": [ { "path_prefix": "/Users/you/work/acme", "client_id": "acme" } ] }
```

Each client gets its own SQLite file, so Client A's memory can never surface in Client B's sessions.

## Repo map

| File | Purpose |
|---|---|
| [`decision.md`](decision.md) | Why we're building this; competitive analysis + spec |
| [`ROADMAP.md`](ROADMAP.md) | Phased plan with live checkboxes |
| [`docs/TECH.md`](docs/TECH.md) | Architecture, schema, interfaces, conventions |
| [`docs/STUDY.md`](docs/STUDY.md) | Reference notes (MCP, hooks, SQLite, embeddings) |
| [`docs/JOURNAL.md`](docs/JOURNAL.md) | Dated log of what each work session did |
| [`CLAUDE.md`](CLAUDE.md) | Entry point + working loop for AI sessions |

## License

Apache-2.0 (a `LICENSE` file is added at the first tagged release).
