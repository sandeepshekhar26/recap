# STUDY.md — Reference notes

External facts the build depends on, with sources and **VERIFY** flags for anything that
moves fast. This is research memory for future sessions — keep it accurate, date the
findings, and re-verify flagged items before relying on them. Last reviewed: **2026-06-23**.

---

## 1. MCP — Go SDK

- **Official Go SDK** exists, maintained with Google: module
  `github.com/modelcontextprotocol/go-sdk`. Primary import:
  `github.com/modelcontextprotocol/go-sdk/mcp`. (`oauthex` subpackage for OAuth extensions.)
- Build a server with the `mcp` package and run it over **stdio** transport — that is what
  Claude Code / Cursor / Codex / Claude Desktop register for local servers. Use Streamable
  HTTP only for a future hosted/team mode.
- **VERIFY:** exact version + API surface before writing server code — the SDK is young and
  pre-1.0; pin a version in `go.mod` and re-check the `mcp` package docs.
- Sources: https://github.com/modelcontextprotocol/go-sdk ·
  https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp

## 2. Claude Code hooks

- **SessionStart** fires when a session begins *or resumes*; matchers/sources include
  `startup`, `resume`, `clear`, `compact`. It can **inject context** — as of Claude Code
  **2.1.0** the injection is *silent* via `hookSpecificOutput.additionalContext` (no
  user-visible message). Resume fires again with `source: resume`, so you can refresh.
  **This is recap's auto-injection mechanism.**
- **SessionEnd** fires on termination with a `reason` (`clear` / `logout` /
  `prompt_input_exit` / other); **cannot block**; receives `session_id`. Good for
  end-of-session capture.
- **Stop** fires after every Claude response — useful for incremental capture, noisier than
  SessionEnd.
- **Output caps:** hook output strings (`additionalContext`, `systemMessage`, stdout) are
  capped around **10,000 characters** — keep injection well under the token budget anyway.
- **Latency:** hooks should be fast (<1s ideal). Proven pattern (claude-mem): enqueue in
  <10ms, do slow compression in a **detached** background worker (`( … ) &>/dev/null & exit 0`).
- **VERIFY:** the full event list moves fast (reported as 12 → ~17 → "30 hook events" across
  2025–2026) and exact JSON field names — re-check the hooks reference at build time.
- Sources: https://code.claude.com/docs/en/hooks ·
  https://claudefa.st/blog/tools/hooks/session-lifecycle-hooks ·
  https://docs.claude-mem.ai/hooks-architecture

## 3. SQLite — FTS5 & vectors

- **FTS5** is SQLite's full-text search (BM25 ranking) via a virtual table; use an external-
  content table mirroring `memories` to avoid duplicate storage.
- **Vectors:** either (a) `sqlite-vec` (Alex Garcia) loadable extension for in-DB ANN, or
  (b) store `float32` BLOBs and compute cosine in-process. (b) is simplest and CGo-free.
- **Driver trade-off (the open decision):**
  - `mattn/go-sqlite3` — CGo; full FTS5; can load extensions like `sqlite-vec`; complicates
    cross-compilation.
  - `modernc.org/sqlite` — pure Go (no CGo); cross-compiles trivially; loadable-extension
    story is weaker → pairs naturally with in-process cosine.
- Hybrid retrieval = FTS5 + vector fused via **reciprocal-rank fusion** (validated by
  Zep/Hindsight as the accuracy unlock). `score = Σ 1/(k + rank)`, `k≈60`.

## 4. Embeddings (local, zero-config)

- Target models: **`all-MiniLM-L6-v2`** (~30MB, 384-dim, English) or
  **`embeddinggemma-300m`** (~300MB, multilingual). Both proven in this space (MemPalace).
- Runtimes by language: Python `fastembed` (easiest), Rust **`fastembed-rs`** / `ort` (best
  for a bundled binary), Node `transformers.js`. Go's only path is CGo `onnxruntime_go` /
  `hugot` — painful packaging, which is why embeddings are a **sidecar**, not in-core.
- **recap plan:** `Embedder` interface, three backends — Rust `fastembed-rs` sidecar
  (default, v1), Ollama HTTP (`all-minilm` / `nomic-embed-text`, if present), FTS5-only nop
  fallback (v0). See `decision.md` §11.
- **VERIFY:** `onnxruntime_go` CGo-vs-cross-compile reality before attempting any in-Go embedder.

## 5. claude-mem (the reference architecture to study)

- Market leader, Apache-2.0 (so studyable). Claude Code plugin: **5 hooks** (SessionStart,
  PostToolUse, Stop, UserPromptSubmit, SessionEnd), SQLite + AI-compressed *observations*,
  local web viewer (port `:37777`), data in `~/.claude-mem/`.
- **What to copy:** the fast-enqueue + async-compress hook pattern; the local viewer; per-
  project filtering; cross-tool reach.
- **What it lacks (our wedge):** no per-*client* (cross-repo, walled) isolation; no
  structured "why"/rejected-approach type — it's an observation log, not a decision model.
- Sources: https://docs.claude-mem.ai/hooks-architecture

## 6. Distribution & registries (for Phase v0 §8)

- **Claude Code marketplace:** repo with `.claude-plugin/marketplace.json`; users
  `/plugin marketplace add <owner>/<repo>` then `/plugin install`. Plugin dir:
  `.claude-plugin/plugin.json`, `.mcp.json`, `hooks/`, `skills/`. Submit to
  `anthropics/claude-plugins-community` for reach.
- **Claude Desktop:** `.mcpb` bundle via `npx @anthropic-ai/mcpb` (`mcpb init` / `mcpb pack`).
- **Official MCP Registry:** publish `server.json` (reverse-DNS namespace, GitHub OIDC name
  proof, CI-automated) → picked up by Smithery / Glama / mcp.so / PulseMCP downstream.
- **Go builds:** GoReleaser for the OS/arch matrix (darwin/linux/windows × amd64/arm64).

## 7. Things to double-check that the spec already flagged (`decision.md` §10)

- Cursor Memories may have moved to Rules in v2.1.x — verify before the Cursor adapter.
- Two products named "OpenMemory" (Mem0's deprecated one vs CaviraOSS's active one).
