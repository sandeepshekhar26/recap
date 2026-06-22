# ROADMAP — recap

Live status tracker. **Checkboxes are the source of truth for progress.** Work top-down
within the current phase. When you finish a task, tick it, move "Current focus", and log
it in [`docs/JOURNAL.md`](docs/JOURNAL.md) — in the same commit. See [`CLAUDE.md`](CLAUDE.md)
for the working loop.

**Legend:** `[ ]` todo · `[x]` done · `[~]` in progress · `[!]` blocked / needs decision

---

## ▶ Current focus

**Phase v0 §1–§3 done** (MCP skeleton + storage + retrieval engine, all tested). Next:
**Phase v0 §5 + §4 — wire the `memory_*` tools to storage/retrieval** so the server is
functional end-to-end: `memory_save`/`memory_recall`/`memory_search` and the differentiator
`memory_save_rejection`/`memory_list_rejections`. Then §6 hooks for auto-capture/injection.

---

## Phase 0 — Foundation (repo is self-sustaining)

- [x] Write `decision.md` (spec + competitive analysis)
- [x] Tech stack decision (`decision.md` §11)
- [x] `git init`, remote, first push
- [x] `CLAUDE.md` entry point + working loop
- [x] `README.md`
- [x] `ROADMAP.md` (this file)
- [x] `docs/TECH.md` (architecture & conventions)
- [x] `docs/STUDY.md` (reference notes)
- [x] `docs/JOURNAL.md` (session log, first entry)
- [x] Go module scaffold (`go.mod`, `cmd/recap`, `.gitignore`) that builds
- [x] CI: GitHub Actions running `gofmt` + `go build` + `go vet` + `go test` on push/PR

## Phase v0 — Claude Code plugin core (decision.md §7)

Goal: a working, installable-by-hand Claude Code plugin that auto-captures and injects
memory locally. Ship to the Claude Code marketplace + official MCP Registry at the end.

### 1. MCP server skeleton ✅
- [x] Add `github.com/modelcontextprotocol/go-sdk/mcp` (v1.6.1); serve an empty stdio server
- [x] `recap serve` subcommand wires the server to stdio transport (clean EOF/ctx shutdown)
- [x] Register no-op versions of the five tools (`memory_recall`, `memory_search`,
      `memory_save`, `memory_save_rejection`, `memory_list_rejections`) so a client can list them
      — verified by an in-memory `ListTools` test and a real stdio JSON-RPC smoke test

### 2. Storage layer (SQLite) ✅
- [x] Choose SQLite driver — **pure-Go `modernc.org/sqlite`** (CGo-free); recorded in TECH.md §8
- [x] Schema + migrations: `memories`, `rejected_approaches`, `sessions`, FTS5 virtual table (+ sync triggers)
- [x] Per-client DB resolution: directory → `client_id` (longest-prefix rules) → DB file path
- [x] CRUD repository functions with tests (save/get memory, rejections, FTS5 search, isolation, embedding roundtrip)

### 3. Retrieval ✅
- [x] FTS5 keyword query (with query sanitization → safe MATCH expression)
- [x] `Embedder` interface + FTS5-only no-op embedder (`internal/embed`)
- [x] Vector cosine over stored embeddings
- [x] Reciprocal-rank fusion of keyword + vector results
- [x] Token-budget-aware selection (hard cap; rejections charged first)
      — note: SessionStart *formatting* of the index/small-files output lands in §6

### 4. The differentiator: rejected-approach capture
- [ ] `memory_save_rejection` writes `{approach, reason_rejected, scope, date}`
- [ ] `memory_list_rejections` for the active project
- [ ] SessionStart injection always prepends active rejections ("Already ruled out: X because Y")

### 5. Tools wired to storage
- [ ] Implement `memory_save` / `memory_recall` / `memory_search` against the repository

### 6. Hooks (auto-capture & inject)
- [ ] `recap hook session-start` → emits `hookSpecificOutput.additionalContext` under token budget
- [ ] `recap hook session-end` / `stop` → enqueue observation fast (<10ms), compress async
- [ ] Background worker: compress queued observations → store (the claude-mem pattern)
- [ ] `recap hook user-prompt-submit` → lightweight relevance injection

### 7. Local web viewer
- [ ] HTTP server + JSON API (list/edit/delete) embedded via `embed.FS`
- [ ] Minimal SPA for browse/edit/delete

### 8. Package & publish
- [ ] Claude Code plugin manifest (`.claude-plugin/plugin.json`, `.mcp.json`, `hooks/`)
- [ ] Marketplace repo (`.claude-plugin/marketplace.json`)
- [ ] `server.json` → official MCP Registry (GitHub OIDC, CI-automated)
- [ ] GoReleaser cross-platform build matrix

## Phase v1 — Cross-tool & packaging (weeks 5–8)

- [ ] Rust `fastembed-rs` embedding sidecar (the zero-config default embedder)
- [ ] Ollama HTTP embedder backend
- [ ] Cursor adapter (MCP + optional `.cursor/rules` injection block, marker-bounded)
- [ ] Codex adapter (MCP)
- [ ] `.mcpb` bundle for Claude Desktop
- [ ] Import/mirror from native Claude Code Auto Memory
- [ ] Token-budget tuner

## Phase v2 — Paid / team (months 3+)

- [ ] Cloud-sync tier (cross-machine)
- [ ] Team sharing (shared client/project memory, RBAC, audit)
- [ ] Hosted embeddings

---

## Decisions still open (resolve before the dependent task)

- [!] **SQLite driver:** CGo (full FTS5 + `sqlite-vec` extension) vs pure-Go (CGo-free,
      manual cosine). Blocks Phase v0 §2. See `decision.md` §11 + `docs/TECH.md`.
- [!] **Embedding default for v0:** ship Ollama-or-FTS5-fallback now, Rust sidecar in v1
      (current plan) — confirm before §6 injection work.
- [!] **Cursor Memories status** (may have moved to Rules in v2.1.x) — verify before the
      Cursor adapter (`decision.md` §10).
