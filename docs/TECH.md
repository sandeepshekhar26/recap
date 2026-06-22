# TECH.md — Architecture & Conventions

How recap is built. Pairs with [`../decision.md`](../decision.md) (the *why* + §11 stack
decision) and [`STUDY.md`](STUDY.md) (external reference notes). When you make a design
decision, record it here under "Decision log."

---

## 1. Component overview

```
                ┌──────────────────────────────────────────────┐
  Claude Code ──┤ hooks: session-start / session-end / stop /   │
  Cursor      ──┤        user-prompt-submit  (recap hook ...)   │
  Codex       ──┤ MCP stdio: memory_* tools (recap serve)       │
                └───────────────┬──────────────────────────────┘
                                │
                        ┌───────▼────────┐      ┌──────────────────┐
                        │  recap core    │      │ Embedder (iface) │
                        │  (Go)          ├─────▶│  - sidecar (Rust)│
                        │  retrieval +   │      │  - ollama (http) │
                        │  capture +     │      │  - fts5-only nop │
                        │  storage       │      └──────────────────┘
                        └───────┬────────┘
                                │
                    ┌───────────▼────────────┐
                    │ SQLite per client_id    │
                    │ FTS5 + embeddings (RRF) │
                    └─────────────────────────┘
```

Everything except the embedding inference is Go. The embedder is the only place another
language (Rust sidecar, v1) is allowed. **Keep ONNX/CGo out of the Go core** (see
`decision.md` §11).

## 2. Directory layout

```
recap/
├── cmd/recap/          # CLI entrypoint: serve | hook | version (thin main)
├── internal/
│   ├── store/          # SQLite: schema, migrations, repositories, per-client resolution
│   ├── retrieval/      # FTS5 + vector + reciprocal-rank fusion + token-budget selection
│   ├── embed/          # Embedder interface + backends (nop/ollama/sidecar)
│   ├── capture/        # observation queue + async compression worker
│   ├── mcp/            # MCP server + tool handlers (memory_*)
│   ├── hook/           # hook event handlers (session-start/end, stop, user-prompt-submit)
│   └── viewer/         # local web viewer (HTTP API + embedded SPA)
├── docs/               # TECH.md, STUDY.md, JOURNAL.md
└── (packaging)         # .claude-plugin/, .mcp.json, hooks/, server.json — added in §8
```

`cmd/recap` stays thin; logic lives in `internal/*`. Public API surface is intentionally
small — this is an application, not a library.

## 3. Data model (SQLite)

From `decision.md` §6. Final DDL lands with Phase v0 §2; this is the intended shape.

```sql
-- one SQLite file PER client_id (physical isolation), under e.g. ~/.recap/<client_id>.db
CREATE TABLE memories (
  id          INTEGER PRIMARY KEY,
  client_id   TEXT NOT NULL,
  project_id  TEXT NOT NULL,
  type        TEXT NOT NULL,           -- decision | convention | session_summary
  content     TEXT NOT NULL,
  rationale   TEXT,                    -- the "why"
  created_at  INTEGER NOT NULL,        -- unix seconds
  embedding   BLOB                     -- float32[] (nullable when no embedder)
);

CREATE TABLE rejected_approaches (     -- THE differentiator
  id              INTEGER PRIMARY KEY,
  client_id       TEXT NOT NULL,
  project_id      TEXT NOT NULL,
  approach        TEXT NOT NULL,
  reason_rejected TEXT NOT NULL,
  created_at      INTEGER NOT NULL,
  embedding       BLOB
);

CREATE TABLE sessions (
  id          TEXT PRIMARY KEY,        -- session_id from the hook
  client_id   TEXT NOT NULL,
  project_id  TEXT NOT NULL,
  summary     TEXT,
  started_at  INTEGER,
  ended_at    INTEGER
);

-- keyword search over memories.content + rationale (and a parallel one for rejections)
CREATE VIRTUAL TABLE memories_fts USING fts5(content, rationale, content='memories', content_rowid='id');
```

- `client_id` = **hard** boundary (separate DB file). `project_id` = **soft** filter within a client.
- Embeddings stored as a `float32` BLOB; cosine computed in-process unless we adopt `sqlite-vec`.

## 4. Key interfaces

```go
// internal/embed
type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    Dims() int
    Name() string
}
// backends: NopEmbedder (FTS5-only), OllamaEmbedder (http), SidecarEmbedder (Rust, v1)

// internal/store
type Repository interface {
    SaveMemory(ctx context.Context, m Memory) (int64, error)
    SaveRejection(ctx context.Context, r Rejection) (int64, error)
    ListRejections(ctx context.Context, clientID, projectID string) ([]Rejection, error)
    Search(ctx context.Context, q Query) ([]Hit, error)   // used by retrieval layer
}

// internal/retrieval
type Retriever interface {
    // FTS5 + vector, fused by RRF, trimmed to a token budget.
    Recall(ctx context.Context, q Query, tokenBudget int) ([]Hit, error)
}
```

## 5. Retrieval (RRF + token budget)

1. Run FTS5 keyword query and vector cosine query in parallel.
2. Fuse with **reciprocal-rank fusion**: `score(d) = Σ 1/(k + rank_i(d))`, `k≈60`.
3. **Always prepend** active `rejected_approaches` for the current project (high-signal —
   `decision.md` §10 "prioritize rejections").
4. Trim to a hard token budget (≈800–1500 tokens) using the index + small-files pattern.

## 6. Hook flow (auto-capture / inject)

- **SessionStart** → `recap hook session-start` prints JSON
  `{"hookSpecificOutput":{"additionalContext":"..."}}` (silently injected since CC 2.1.0).
  Budget-trimmed recall + active rejections. Must be fast.
- **SessionEnd / Stop** → enqueue an observation in **<10ms**, return immediately; a detached
  background worker does slow AI compression (5–30s) and writes to the store. Never block the hook.
- **UserPromptSubmit** → lightweight relevance injection (optional in v0).

See [`STUDY.md`](STUDY.md) for verified hook field names and the `<1s` constraint.

## 7. Conventions

- **Module:** `github.com/sandeepshekhar26/recap`. Go (see `go.mod` for version).
- **Errors:** wrap with `%w`; no panics in library code; CLI maps errors to non-zero exit.
- **Context:** every I/O-bound function takes `context.Context` first.
- **Tests:** table-driven; `go test ./...` is the gate. Retrieval has **golden tests**
  (fixed corpus → expected top-K) and an **isolation test** (Client A query never returns
  Client B rows).
- **No CGo in the core** unless the SQLite-driver decision (below) requires it; if so,
  isolate it so cross-compilation stays sane.
- **Commits:** one logical change; imperative subject; body says *why*; Claude co-author trailer.

## 8. Decision log

Record every non-obvious technical decision here (newest first).

- **2026-06-23 — SQLite driver: `modernc.org/sqlite` (pure Go).** No CGo → trivial
  cross-compile for the GoReleaser matrix and a true single binary. **FTS5 verified working**
  under modernc (TestFTS5Search). Vectors stored as little-endian `float32` BLOBs with
  in-process cosine (§3); `sqlite-vec` deferred — unnecessary at v0 scale and it would
  reintroduce CGo. One DB connection (`SetMaxOpenConns(1)`) + WAL + busy_timeout for safe
  low-concurrency local writes.
- **2026-06-23 — Stack:** Go core + pluggable `Embedder` (Rust `fastembed-rs` sidecar
  default / Ollama / FTS5-only). Rationale in `decision.md` §11.
- **2026-06-23 — Continuity model:** repo is self-sustaining; `CLAUDE.md` working loop +
  `ROADMAP.md` checkboxes + `JOURNAL.md` log are the cross-session memory. Chat history is
  not authoritative.

## 9. Open technical questions (VERIFY before the dependent task)

- ~~**SQLite driver**~~ — RESOLVED 2026-06-23: `modernc.org/sqlite` (pure Go), in-process
  cosine. See decision log.
- ~~**Go MCP SDK** version~~ — RESOLVED: pinned at v1.6.1 (past 1.0, stable).
- **Embedding default for v0:** Ollama-or-FTS5-fallback now, Rust sidecar in v1 — confirm
  before §3 vector work and §6 injection.
