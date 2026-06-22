# JOURNAL — work log

Append-only. **Newest entry at the top.** Each entry: what changed, why, and what's next.
A fresh session should read the top entry first to orient. Keep entries short and factual.

---

## 2026-06-23 — Foundation: repo made self-sustaining + Go scaffold

**Done**
- Established the continuity system so sessions don't depend on chat history:
  `CLAUDE.md` (entry point + working loop), `README.md`, `ROADMAP.md` (live checkboxes),
  `docs/TECH.md` (architecture/conventions/decision log), `docs/STUDY.md` (reference notes),
  this journal.
- Verified two fast-moving facts before writing code:
  - Go MCP SDK import path: `github.com/modelcontextprotocol/go-sdk/mcp`.
  - SessionStart injects silently via `hookSpecificOutput.additionalContext` (CC 2.1.0+).
- Scaffolded the Go module (`go.mod`, `cmd/recap` CLI skeleton with `serve` / `hook` /
  `version` stubs, `.gitignore`). Builds clean with stdlib only (no deps yet).

**Why**
- The repo must carry its own state so any new session resumes from files, not memory —
  this is also dogfooding the product's own thesis (memory for coding agents).

**Next**
- Phase 0: add CI (`go build` + `go test` on push).
- Phase v0 §1: add the Go MCP SDK and serve an empty stdio server; then register the five
  `memory_*` tools as no-ops so a client can list them.

**Open decisions** (see ROADMAP "Decisions still open"): SQLite driver (CGo vs pure-Go);
v0 embedding default (Ollama-or-FTS5 now, Rust sidecar in v1); Cursor Memories status.
