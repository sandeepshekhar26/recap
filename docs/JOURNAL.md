# JOURNAL — work log

Append-only. **Newest entry at the top.** Each entry: what changed, why, and what's next.
A fresh session should read the top entry first to orient. Keep entries short and factual.

---

## 2026-06-23 (cont.) — Phase v0 §8: packaging artifacts — PHASE v0 COMPLETE 🎉

**Done**
- `.claude-plugin/plugin.json` — Claude Code plugin with mcpServers (`recap serve`) and the
  three hooks inline (authoritative, avoids strict-mode ambiguity).
- `.claude-plugin/marketplace.json` — single-plugin marketplace (`source: "./"`), so
  `/plugin marketplace add sandeepshekhar26/recap` works.
- `server.json` — official MCP Registry manifest (schema 2025-12-11, name
  `io.github.sandeepshekhar26/recap`, mcpb package).
- `.goreleaser.yaml` — release matrix with **CGO_ENABLED=0**.
- **Verified CGO-free cross-compile** for darwin/linux/windows × amd64/arm64 (5 targets) —
  this is the payoff of the pure-Go SQLite decision. All three JSON manifests validated.
- README updated: status → "v0 functional, pre-release", quickstart (Claude Code plugin,
  Cursor/Codex `.mcp.json`, viewer, per-client `config.json`), license Apache-2.0.

**Phase v0 status:** §1–§8 all done. recap is a working local-first memory layer end-to-end:
MCP server (5 tools) + hooks (session-start injection) + hybrid retrieval + per-client
isolation + web viewer + packaging. Test suite green across mcp/store/retrieval/hook/viewer.

**Deferred to v1 (need a model or a release):** semantic recall (Ollama / `fastembed-rs`
sidecar), async LLM observation compression, Cursor/Codex adapters, `.mcpb` bundle build,
`LICENSE` file, and the actual registry/marketplace submission.

**Next (Phase v1):** start with the Ollama embedder backend (`internal/embed`) so recall
goes semantic with zero call-site changes, then the Rust sidecar and cross-tool adapters.

---

## 2026-06-23 (cont.) — Phase v0 §7: local web viewer

**Done**
- `internal/viewer`: HTTP `Handler` with JSON API — `GET /api/info`, `GET /api/memories`,
  `GET /api/rejections`, `DELETE /api/memories/{id}`, `DELETE /api/rejections/{id}` (Go 1.22
  method+wildcard routes) — plus a single embedded `index.html` (`embed.FS`) SPA that lists
  rejections + memories with delete buttons (light/dark, zero external deps).
- Store: `AllMemories`, `AllRejections`, `DeleteMemory`, `DeleteRejection` (FTS stays in sync
  via the delete trigger).
- `recap viewer [--addr]` (default `127.0.0.1:37788`) opens the per-client DB and serves with
  graceful shutdown.
- Tests: httptest list/delete round-trip + index content-type; **live** smoke (`/api/info`
  JSON + `/` 200, 3654 bytes).

**Why**
- claude-mem proved users want a local browse/edit surface; delete also implements the §10
  "easy edit/delete" mitigation against context poisoning. In-place edit deferred (delete +
  re-save via tools covers it).

**Next**
- **Phase v0 §8 — packaging artifacts:** Claude Code plugin manifest + `.mcp.json` + hooks
  config, `marketplace.json`, `server.json` (MCP registry), GoReleaser. Actual publishing is
  a manual release step.

---

## 2026-06-23 (cont.) — Phase v0 §6: lifecycle hooks

**Done**
- `internal/hook`: `ParseInput` (stdin payload), `SessionStartContext` / `PromptContext`
  builders that emit `{"hookSpecificOutput":{"hookEventName","additionalContext"}}` (silent
  injection, CC 2.1+). Moved the recall formatters down into `internal/retrieval`
  (`FormatRecall`/`FormatMemories`/`FormatRejections` + `Result.HasContent`) so both `mcp`
  and `hook` share them without an import cycle.
- `recap hook <event>` dispatch in cmd (best-effort: errors → stderr, **exit 0**, never
  breaks the session):
  - `session-start` → recall (rejections + relevant memories) → additionalContext; nothing
    when empty.
  - `user-prompt-submit` → small prompt-relevant injection (budget 500, memories only — no
    repeated rejections, to avoid context poisoning per decision.md §10).
  - `session-end` → `store.UpsertSession` bookkeeping. `stop` → no-op.
- Added `store.UpsertSession`. Hook package unit tests (parse, JSON output shape, empty
  cases) + **live smoke test**: seeded a rejection+memory via serve, then `hook session-start`
  emitted the correct injection JSON; empty project emitted nothing; session-end exit 0.

**Why / honest scope**
- SessionStart injection is the highest-value, fully-deterministic hook and it works. The
  "compress observations with an LLM" half of auto-capture is **deferred to v1**: it needs a
  model, which conflicts with v0's zero-config-local promise. v0's capture path is explicit
  (the agent calls `memory_save`/`memory_save_rejection`); automatic transcript compression
  arrives with the Ollama/sidecar work.

**Next**
- **Phase v0 §7 — local web viewer:** HTTP JSON API (list/delete) + embedded SPA.

---

## 2026-06-23 (cont.) — Phase v0 §4 + §5: tools wired to storage

**Done**
- `internal/config`: loads optional `$RECAP_HOME/config.json` (directory→client_id rules,
  base dir) into a `store.Config`; missing file = defaults, not an error.
- `internal/mcp` refactored to carry `Deps{Store, Retriever, ClientID, ProjectID}`; the five
  tools are now **real**:
  - `memory_save` (validates type), `memory_recall` (rejections + fused memories),
    `memory_search` (memories only), `memory_save_rejection`, `memory_list_rejections`.
  - Bad input returns `IsError` results, not protocol errors.
  - `format.go` renders recall as "Already ruled out … / Relevant memories …" (reused by §6).
- `recap serve` resolves client_id (config rules) + project_id (nearest `.git`) from cwd,
  opens that client's DB, builds the retriever (Nop embedder for now), and serves.
- Tests: end-to-end tool calls over the in-memory transport (save → recall surfaces the
  rejection first + keyword-matched memory), invalid-input → IsError, plus a **live stdio
  smoke test** of the built binary with an isolated `RECAP_HOME` (persisted to `default.db`,
  exit 0).

**Why**
- The server is now genuinely usable by any MCP client; the differentiator
  (`memory_save_rejection` → always surfaced on recall) works end-to-end.

**Note (honest scope):** recall is keyword-only until an embedder is wired (v1) — a query
with no keyword overlap returns only the always-on rejections. Semantic recall comes free by
swapping `embed.Nop` for Ollama/sidecar.

**Next**
- **Phase v0 §6 — hooks:** `recap hook session-start` (inject recall as
  `additionalContext`), `session-end` (session bookkeeping), `user-prompt-submit` (light
  injection). LLM-based observation compression is deferred to v1 (needs a model; conflicts
  with zero-config-local — lands with Ollama/sidecar).

---

## 2026-06-23 (cont.) — Phase v0 §3: retrieval engine

**Done**
- `internal/embed`: `Embedder` interface (`Embed`/`Dims`/`Name`) + `Nop` backend so
  retrieval degrades to keyword-only until a real model is wired (Ollama/sidecar in v1).
- `internal/retrieval`: hybrid `Retriever.Recall`:
  - keyword via FTS5 (with `sanitizeFTS` → quoted-OR MATCH expr; empty-text falls back to
    recent memories);
  - vector via cosine over candidate embeddings (skipped when embedder Dims()==0);
  - `fuseRRF` (k=60) dedupes + fuses both rankings;
  - `trimToBudget` enforces a hard token cap, charging always-injected rejections first and
    keeping ≥1 memory.
  - Active rejections are returned separately and always (decision.md §10 "prioritize rejections").
- Added `store.ListMemories` (candidate pool for vector scan / empty-query fallback).
- Tests: RRF ordering+dedupe, cosine (identical/orthogonal/mismatched), FTS sanitization,
  and end-to-end Recall over a real DB — keyword path, vector-fusion path (fake embedder),
  and budget trimming.

**Why**
- This is the accuracy core (decision.md §6). Building it behind the `Embedder` interface
  means v0 ships useful keyword recall now and gains semantic recall by swapping the backend
  later — no call-site changes.

**Next**
- **Phase v0 §5 + §4:** wire the five `memory_*` MCP tools to the store + retriever so the
  server actually persists and recalls (currently they are no-op stubs).

---

## 2026-06-23 (cont.) — Phase v0 §2: storage layer

**Done**
- **Driver decision: `modernc.org/sqlite` (pure Go).** Resolves the gating open question —
  CGo-free, cross-compiles trivially, **FTS5 verified working** (TestFTS5Search). Recorded
  in TECH.md §8; §9 open-questions trimmed. `sqlite-vec` deferred (would reintroduce CGo).
- `internal/store`: `Open()` with WAL + busy_timeout + `MaxOpenConns(1)` and idempotent
  v1 migrations (memories / rejected_approaches / sessions + `memories_fts` FTS5 external-
  content table kept in sync by insert/update/delete triggers).
- Models (`Memory`, `Rejection`, `Session`, `MemoryType`) + little-endian `float32` BLOB
  embedding encode/decode (ready for §3 vectors; nil today).
- Repository CRUD: `SaveMemory`/`GetMemory`, `SaveRejection`/`ListRejections` (newest-first,
  project-scoped), `SearchMemories` (FTS5 BM25, project-scoped).
- Per-client resolution (`clients.go`): `Config` with longest-prefix `ClientRule`s →
  `client_id`; `DBPath` = one `<client_id>.db` per client under `$RECAP_HOME`/`~/.recap`;
  `ResolveProjectID` walks up to the nearest `.git`.
- Tests: CRUD, validation, FTS5 search, client-id resolution incl. path-segment edge cases,
  project-id git-walk, embedding roundtrip, and the **isolation guarantee** (client B's DB
  can't see client A's rows).

**Why**
- Storage is the contract retrieval (§3) and the tool handlers (§4–§5) build on; the
  per-client *file* boundary is what makes the privacy story (decision.md §4 Axis 2) real
  rather than a soft filter.

**Next**
- **Phase v0 §3 (Retrieval):** `Embedder` interface + FTS5-only no-op embedder; vector
  cosine; reciprocal-rank fusion; token-budget selection.

---

## 2026-06-23 (cont.) — Phase v0 §1: MCP server skeleton

**Done**
- Added the official Go MCP SDK **v1.6.1** (`github.com/modelcontextprotocol/go-sdk/mcp`).
  Confirmed the real API by `go doc` before coding: `NewServer`, generic
  `AddTool[In,Out]` with `ToolHandlerFor`, `StdioTransport`, `Server.Run`,
  `NewInMemoryTransports`.
- `internal/mcp`: `Serve()` runs the stdio server; `newServer()` registers the five
  `memory_*` tools as typed no-op stubs (each carries its input JSON schema via a Go
  struct + `jsonschema` tags, and returns a "not implemented (Phase §x)" message).
- `recap serve` now starts the server with SIGINT/SIGTERM graceful shutdown; client
  disconnect (stdin EOF) and ctx-cancel are treated as **clean** exits (exit 0).
- Tests: in-memory `ListTools` test asserts exactly the five tools with descriptions
  (first real test → CI now has something to run). Also did a manual stdio JSON-RPC smoke
  test (initialize → tools/list → tools/call) — all green, exit 0.

**Why**
- A discoverable tool surface is the contract every client (Claude Code/Cursor/Codex)
  binds to; locking the names + schemas now lets storage/retrieval land behind a stable API.

**Next**
- **Phase v0 §2 (Storage):** decide SQLite driver (CGo `mattn` vs pure-Go `modernc`) —
  this is the gating open decision — then schema + migrations + per-client DB resolution.

---

## 2026-06-23 (cont.) — Phase 0 complete: CI

**Done**
- Added `.github/workflows/ci.yml`: on push to `main` and on PRs, runs gofmt check →
  `go build ./...` → `go vet ./...` → `go test ./...` on `ubuntu-latest`, Go version read
  from `go.mod`. This closes the last Phase 0 box; **Phase 0 is complete.**
- Verified the same suite locally — all green (no test files yet, which is expected).

**Why**
- A build/test gate from day one keeps the commit-by-commit cadence honest before real code lands.

**Next**
- **Phase v0 §1:** add `github.com/modelcontextprotocol/go-sdk/mcp`, make `recap serve` run
  an empty stdio MCP server, then register the five `memory_*` tools as no-ops.

**Note:** CI only *executes* after a `git push` (it triggers on the `push`/`pull_request`
events). No marketplace/registry "publish" is involved — Actions runs automatically once
the workflow file is on GitHub.

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
