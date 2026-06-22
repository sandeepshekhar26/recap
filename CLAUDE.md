# CLAUDE.md — Read this first, every session

This file is the **entry point for any AI coding session** in this repo. The repo is
designed to be **self-sustaining**: all state, decisions, and progress live in tracked
files, not in chat history. A fresh session with zero prior context should be able to
read this file and continue the work correctly. **Do not rely on conversation memory —
rely on the repo.**

## What this project is

**recap** — a local-first memory & project-knowledge layer for coding agents, shipped as
a Claude Code plugin (MCP server + lifecycle hooks) and a standalone MCP server for
Cursor and Codex. The full rationale, competitive analysis, and spec live in
[`decision.md`](decision.md). The wedge: hook-driven auto-capture + per-client isolation
+ structured "why / rejected-approach" capture + cross-tool portability.

## The source-of-truth files (read in this order)

1. [`decision.md`](decision.md) — **why** we're building this and the full spec. Rarely changes.
2. [`ROADMAP.md`](ROADMAP.md) — **what's done and what's next.** Checkboxes are the live status.
3. [`docs/TECH.md`](docs/TECH.md) — **how** it's built: architecture, schema, interfaces, conventions.
4. [`docs/STUDY.md`](docs/STUDY.md) — **reference notes** (MCP, hooks, SQLite, embeddings) with verify-flags.
5. [`docs/JOURNAL.md`](docs/JOURNAL.md) — **what each session did.** Append-only log; read the last entry to orient.

## The working loop (follow this exactly)

Every session, for every unit of work:

1. **Orient.** Read `ROADMAP.md` ("Current focus" + first unchecked task) and the last
   `docs/JOURNAL.md` entry. Run `git log --oneline -10` to see recent commits.
2. **Pick the next unchecked task** from the roadmap (top-down within the current phase).
   Do not skip ahead phases.
3. **Implement** it following the conventions in `docs/TECH.md`.
4. **Verify** it: `go build ./...` and `go test ./...` must pass before you commit.
5. **Update state in the same commit:**
   - Tick the checkbox(es) in `ROADMAP.md` (`[ ]` → `[x]`) and move the "Current focus" pointer.
   - Append a dated entry to `docs/JOURNAL.md` (what changed, why, what's next).
   - If a design decision was made, record it in `docs/TECH.md` (and, if strategic, `decision.md`).
6. **Commit** — one logical change per commit (see style below). Push only when asked.

If you finish a task and the roadmap's "Current focus" still points at it, you didn't
finish step 5. Treat docs+code as a single deliverable.

## Conventions (summary — full version in docs/TECH.md)

- **Language:** Go for the core (module `github.com/sandeepshekhar26/recap`). Embeddings
  are isolated behind an interface; a Rust sidecar is the only non-Go component, and only
  in v1. See `decision.md` §11.
- **Layout:** `cmd/recap` (CLI entrypoint), `internal/*` (implementation). Keep the core
  CGo-free; do not pull ONNX into the Go core.
- **Commits:** imperative subject ≤ ~72 chars; body explains *why*. Co-author trailer:
  `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`. One logical change per commit.
- **Don't break the build.** `go build ./...` and `go test ./...` are the gate.

## Guardrails

- Keep this repo **self-documenting**. If you learn something a future session needs,
  write it down here, in `TECH.md`, or in `STUDY.md` — never leave it only in chat.
- Anything marked **VERIFY** in the docs moves fast (MCP SDK, hook APIs) — re-check
  against upstream before relying on it.
- Push, publish, or release **only when explicitly asked.**
