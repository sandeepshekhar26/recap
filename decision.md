# decision.md — Local-first Memory & Project-Knowledge Layer for Coding Agents (MCP plugin)

> Standalone decision dossier for this project. Combines competitive/landscape analysis and a build specification, in equal depth. Be brutally honest with yourself when reading section 10.

## 1. Executive Summary & Verdict

**Verdict: Build it, but only if you win on a sharply-defined wedge — turnkey, hook-driven auto-capture + per-client isolation + "why/rejected-approach" capture + true cross-tool portability — because the generic "agent memory" category is brutally crowded and partially commoditized.** The pain is real and widely documented (claude-mem alone shows 82.2K GitHub stars on its repo header as of June 2026), but "memory for coding agents" is now table stakes: Anthropic shipped native Auto Memory in Claude Code (v2.1.59+), Cursor shipped per-project Memories in v1.0 (June 2025), and Mem0/Supermemory/claude-mem all ship Claude Code plugins. A solo dev cannot win "best general memory layer." A solo dev *can* win "the memory tool a freelancer/agency dev installs once and never thinks about, that never bleeds Client A's context into Client B, and that stops the agent re-pitching approaches already ruled out — across Claude Code, Cursor, and Codex."

The durable wedge is NOT the storage tech (SQLite + embeddings is commodity). It is (a) the *cross-tool* abstraction that makes any single vendor's native memory a feature you absorb rather than a competitor, and (b) two under-served capabilities — strict per-client isolation and "why" capture — that the incumbents treat as afterthoughts.

## 2. The Problem / Why This Matters

Every LLM coding session starts from a blank context window. Developers re-explain architecture, re-paste decisions, and — most painfully — watch the agent re-suggest an approach they already tried and rejected. Mem0's own Series A announcement (Oct 28, 2025) names this exact pain: "They watch coding assistants suggest the same rejected patterns dozens of times." Evidence of pain:

- **claude-mem** (Alex Newman / @thedotmack), a Claude Code memory plugin, shows 82.2K stars on its GitHub repo header as of June 2026 (it was tracked at 65.8K in April 2026 by Augment Code — the growth rate itself is a signal). The project reached v12.3.8 as of April 20, 2026, with 1,792 commits, 106 contributors, and 244 releases. Its tagline names the exact problem: "sessions that forget everything."
- The Hacker News thread on Letta Code surfaced the *counter*-argument too: "In my experience, 'memory' is really not that helpful in most cases… Maintaining the memory is a considerable burden" and "Context poisoning is a real problem that these memory providers only make worse." This is the central risk: bad memory is worse than no memory.
- Anthropic's own GitHub issues (#36561, #21854) acknowledge the cross-project memory gap; both were closed as duplicates.
- A DEV Community thread documents the "context rot" failure mode: a single large CLAUDE.md grows until "Claude starts skipping sections when loading (the 'laziness' failure mode)." The fix the community converged on: "~20-line category files with clear filenames, keep a 1-line index."
- The freelancer/agency problem is specifically under-served: native Claude Code memory is per-repository by default (`~/.claude/projects/<project>/memory/`), and there is no built-in notion of a *client* boundary that spans repos but is walled off from other clients.

The "why" capture point is the sharpest insight: one DEV commenter noted "'attempted_and_failed' [is] more valuable than the raw strategy itself… memory stops being a log and starts becoming operational guidance." Most tools store facts and preferences; few capture *rejected approaches with rationale*, which is exactly what prevents the agent from re-suggesting dead ends.

## 3. Competitive Landscape

The single most important distinction: **libraries/SDKs (a dependency for people *building apps*) vs. turnkey products for *coding-agent users*.** Mem0's ~41K stars (and 14M downloads, per its Oct 2025 funding announcement) and Letta's ~21.7K stars are *developer-platform* traction — they matter to someone building a customer-support bot, not to a freelancer who wants their Cursor to stop forgetting. Library stars are not a moat against a finished end-user product, the same way Redis having vastly more stars than a polished note-taking app tells you nothing about who wins the note-taking market. The relevant competitors for *this* product are the turnkey coding-agent memory tools, plus the built-in baselines.

### Libraries / SDKs / platforms (target = app developers, NOT the wedge)
| Tool | What it is | License / traction | Auto-capture | Per-project isolation | "Why" capture | Cross-tool |
|---|---|---|---|---|---|---|
| **Mem0** | Managed memory API + OSS self-host; vector+graph+KV, 3-tier (user/session/agent) | Apache-2.0; ~41K stars, 14M downloads; **Oct 28 2025 funding: Seed led by Kindred Ventures + $20M Series A led by Basis Set Ventures** (w/ Peak XV, GitHub Fund, YC); V3 algo claims 91.6 on LoCoMo (Apr 2026) | Via SDK calls, not agent lifecycle | Namespacing by user/session/agent | No | Has a Claude Code plugin (hosted MCP) but is an API-first platform |
| **OpenMemory (Mem0's)** | Former local-first MCP server + React UI | **Deprecated/sunset**, folded into "Mem0 self-hosted server"; note Mem0 *also* now markets a product literally called "OpenMemory" for coding agents — confusing naming | Auto-captures coding prefs (per marketing) | Per-repo matching | No | Claude Code / IDE focus |
| **CaviraOSS/OpenMemory** | Separate OSS local memory store (SQLite/Postgres, Ollama embeddings) | Distinct project from Mem0's; confusingly same name | Partial | Yes (namespaces) | No | Claude Desktop, Copilot, Codex |
| **Zep / Graphiti** | Temporal knowledge-graph memory; bi-temporal edges (t_valid/t_invalid) | Graphiti OSS (Neo4j-backed); Zep is enterprise SaaS; SOTA on DMR (94.8% vs MemGPT 93.4%) | Via ingestion API | Graph partitioning | Temporal invalidation ≈ partial "why" | API/SDK, not coding-agent native |
| **Letta (ex-MemGPT)** | Stateful-agent runtime, tiered memory (core/archival/recall); now "Letta Code" | ~21.7K stars; YC-backed, $10M seed (Felicis/Founders Fund/YC); **Letta Code: 42.5% on Terminal-Bench, "ranking 4th overall and 2nd among agents using Claude 4 Sonnet," #1 model-agnostic open-source agent** | Agent self-edits memory via tools | Per-agent | Via agent reasoning, not structured | Its own runtime; a competitor at the *agent* layer |
| **Supermemory** | All-in-one memory API (memory+RAG+profiles); claims LongMemEval/LoCoMo lead (self-reported) | Closed-source core; ships Claude Code/OpenCode plugins | Signal extraction (keyword-triggered) | `repoContainerTag` per repo; team memory | No | Strongest cross-coding-agent of the API players |
| **cognee** | OSS graph+vector pipeline (SQLite + LanceDB + Kuzu); Python-only | OSS; local-first capable | Pipeline-driven | Yes | No | Library |
| **Memobase / Membase / Memary / txtai / agentmemory** | Various memory libraries/SDKs | OSS, smaller | No (library primitives) | Varies | No | Library |

### Turnkey coding-agent memory tools (the real competitive set)
| Tool | What it is | Auto-capture via hooks | Per-project isolation | "Why" capture | Cross-tool | Gap it leaves |
|---|---|---|---|---|---|---|
| **claude-mem** (thedotmack) | The market leader; Claude Code plugin, SQLite + AI-compressed observations, local; Apache-2.0; 82.2K+ stars; web viewer at :37777; data in `~/.claude-mem/` | **Yes — 5 hooks** (SessionStart, PostToolUse, Stop, UserPromptSubmit, SessionEnd) | Per-project filtering | No — captures *observations*, not rejected-with-rationale | Yes — Claude Code, Gemini, Codex, OpenCode, Copilot, Hermes | No *client* (cross-repo, walled) isolation; no structured "why"; observation-log model, not decision/rejection model |
| **Hindsight** (Vectorize) | Multi-scope memory (user/agent/session/org), multi-strategy retrieval (semantic+entity+temporal+graph) w/ cross-encoder rerank; Claude Code plugin | Auto-recall every prompt, auto-retain after response | Scope choice (tag at agent_id vs user_id) | No explicit rejected-approach type | Plugin + Python/TS/Go SDKs | Heavier; API/infra-leaning; no client-isolation primitive |
| **Mem0 Claude Code plugin** | Hosted-cloud memory via MCP; free tier 10K memories/1K retrievals/mo | Lifecycle hooks + skills | user/session/agent | No | Claude Code CLI + Cowork | Cloud-first; not zero-config-local; no "why" |
| **Supermemory Claude plugin** | `supermemory-search`/`supermemory-save`; team memory | Signal keywords (`remember`, `decision`, `bug`, `fix`) | `repoContainerTag` | No | Yes | Closed-source; keyword-triggered, not lifecycle-complete |
| **itsjwill/claude-memory** | Free OSS; auto-capture + Supabase cloud backup; explicitly markets "client info" capture | Skills (capture/session-start/session-end) | Tags incl. client | Captures "decisions/learnings" | Claude Code | Closest to your vision but Supabase-coupled, no strict isolation guarantees, no cross-tool |
| **MemPalace** | OSS verbatim-store + semantic search; ChromaDB default; embeddinggemma-300m local; hooks for Claude Code/Codex/Cursor | Auto-save hooks + before-compaction snapshot | "wings" per project; namespace isolation on Qdrant/pgvector | No (verbatim, not structured) | Claude Code, Codex, Cursor, MCP | No structured decision/why model |
| **MemNexus / MemoryBank (community)** | Cross-project MCP memory layers | Varies | Cross-project by design | No | MCP-based | Thin, early |

### Built-in baselines (the real long-term threat)
- **Claude Code CLAUDE.md**: static, hand-written, loaded every session; four scopes; survives compaction at project root. Manual.
- **Claude Code Auto Memory** (v2.1.59+, on by default): Claude writes its own notes to `~/.claude/projects/<project>/memory/`, MEMORY.md index loaded at startup, topic files on demand. **Per-repository by default.** Real limits: 200-line/25KB startup cap on MEMORY.md, index-based (not semantic) retrieval, no cross-project/cross-client bridge. Disable via `CLAUDE_CODE_DISABLE_AUTO_MEMORY=1`.
- **Cursor Memories** (shipped v1.0 June 2025): background model proposes, user approves; "stored per project on an individual level." **Note: one source reports the feature was removed in v2.1.x, with users advised to export to Rules — verify current status before building the Cursor adapter.** Also `.cursor/rules/*.md` / AGENTS.md (static; legacy single-file `.cursorrules` deprecated).
- **Codex**: relies on AGENTS.md-style files; weakest native memory.

**The honest threat assessment:** Anthropic and Cursor have *already* shipped native memory. They will keep improving it. But both are (1) single-vendor and (2) per-repo. Neither solves the freelancer's cross-repo-but-client-walled problem, neither captures "why/rejected," and neither is portable across the three tools a polyglot dev actually uses. That is the moat: **you are the Switzerland layer that gets *more* valuable each time a vendor ships native memory, because you normalize across them.**

### The specific gap nobody fills well
A *turnkey* product (not a library) that simultaneously delivers: (1) **hook-driven auto-capture** that needs zero user discipline; (2) **per-client isolation** as a first-class boundary spanning repos but walled off (the agency use-case); (3) **structured "why"/rejected-approach capture** so the agent stops re-pitching dead ends; (4) **token-budget-aware injection** (the index+small-files pattern, done automatically); and (5) **genuine cross-tool portability** (one store, Claude Code + Cursor + Codex). claude-mem nails #1 and partially #4/#5 but misses #2 and #3. Everyone else misses more.

## 4. Why We Can Win — the features-ahead axis & durable wedge

- **Axis 1 — "Why" capture as a data model, not an afterthought.** Ship a first-class `rejected_approach` record type: {approach, reason_rejected, date, scope}. On SessionStart injection, surface "Already ruled out: X because Y" so the agent never re-pitches. No incumbent has this as a typed primitive. This is the single most defensible *feature* differentiator.
- **Axis 2 — Per-client isolation.** A `client`/`workspace` boundary above the repo: memories tagged to Client A are *physically* unable to surface in Client B's sessions (separate SQLite DBs per client, selected by directory mapping). This is the freelancer/agency killer feature and a genuine trust/privacy story.
- **Axis 3 — Cross-tool Switzerland.** One local backend, thin adapters per tool (Claude Code hooks+MCP, Cursor MCP+rules-injection, Codex MCP). When Anthropic improves Auto Memory, you ingest/mirror it rather than compete. This converts the biggest *threat* into a *feed*.
- **Axis 4 — Zero-config local + token discipline.** SQLite + a small local embedding model, the index+small-files injection pattern automated, with a hard token budget. No API key, no cloud, private by default.

The wedge is durable because it is **multi-vendor positioning + two data-model features**, none of which a single vendor is incentivized to copy (Anthropic won't make Claude Code memory work great *in Cursor*).

## 5. What We're Shipping (v0 form factor & feature set)

A **Claude Code plugin** (bundling an MCP server + lifecycle hooks) that is also installable as a **standalone local MCP server** for Cursor and Codex. Local SQLite, small local embedding model, zero-config.

**v0 feature set:**
- MCP tools: `memory_recall`, `memory_search`, `memory_save`, `memory_save_rejection` (the differentiator), `memory_list_rejections`.
- Hooks: SessionStart (inject relevant decisions/conventions/rejections within a token budget), SessionEnd + Stop (auto-capture observations → compress → store), UserPromptSubmit (lightweight relevance injection).
- Record types: `decision`, `rejected_approach`, `convention`, `session_summary`.
- Per-client isolation via directory→client mapping (one SQLite DB per client).
- Hybrid retrieval: SQLite FTS5 keyword + vector similarity.
- Local web viewer for browse/edit/delete (claude-mem proved users want this).

## 6. Build Specification

### Claude Code hooks (verified against current docs)
Hook events fire at three cadences: once per session (**SessionStart**, **SessionEnd**), once per turn (**UserPromptSubmit**, **Stop**, **StopFailure**), and per tool call (**PreToolUse**, **PostToolUse**). Full event list includes Setup, PreCompact, SubagentStart/Stop, Notification, and more (~17 events as of 2026). Configuration lives in `settings.json` under a `hooks` block (or bundled in a plugin). Each hook is a `type: "command"` (shell, reads JSON on stdin) or `type: "mcp_tool"` or `type: "prompt"` handler.

Critical mechanics for this product:
- **SessionStart**: stdout is **injected as context Claude can see** (one of only three events with this property, alongside UserPromptSubmit and UserPromptExpansion). As of Claude Code 2.1.0, SessionStart context is injected silently via `hookSpecificOutput.additionalContext`. `source` field tells you startup/resume/clear/compact — so you can refresh on resume. **This is your auto-injection mechanism.**
- **SessionEnd**: fires once on termination with a `reason` field (clear/logout/prompt_input_exit/other); **cannot block**; ideal for end-of-session capture. Receives `session_id`.
- **Stop**: fires after every Claude response — useful for incremental capture but noisier than SessionEnd.
- **Hooks must be fast (<1s ideal; default command timeout 600s but you should set explicit short timeouts).** The proven pattern (claude-mem) is: hook enqueues an observation in <10ms, a background worker does the slow AI compression (5–30s) asynchronously. Run slow work in a detached subshell (`( … ) &>/dev/null & exit 0`).
- SessionStart can return `{"hookSpecificOutput":{"additionalContext":"…"}}` and even `reloadSkills`/`sessionTitle` (v2.1.152+).

### MCP server implementation
- Expose tools via the official MCP SDK (`@modelcontextprotocol/sdk`, `McpServer` + `StdioServerTransport`). **stdio transport** for local (what Claude Desktop/Cursor/Codex register); Streamable HTTP only if you later offer a hosted/team mode.
- Discovery/loading: Claude Code reads MCP servers from plugin `.mcp.json` or user/project config; Cursor reads `~/.cursor/mcp.json` (global) or `.cursor/mcp.json` (project); Claude Desktop reads `claude_desktop_config.json`.
- A **plugin bundles MCP server + hooks together**: `plugin.json` declares `mcpServers`; `hooks/` dir + `.mcp.json` ship in the same installable unit.

### Storage / retrieval architecture
- **SQLite** with tables: `memories(id, client_id, project_id, type, content, rationale, created_at, embedding BLOB)`, `rejected_approaches(id, client_id, project_id, approach, reason_rejected, created_at, embedding)`, `sessions(id, client_id, project_id, summary, started_at, ended_at)`, plus an **FTS5** virtual table for keyword search.
- **Embeddings**: a small local model so it's zero-config and private. Options: `all-MiniLM-L6-v2` (~30MB, English) or `embeddinggemma-300m` (~300MB, multilingual) — both proven in this space (MemPalace ships exactly these choices). Run via a bundled ONNX runtime or a tiny local server; avoid mandatory API keys.
- **Hybrid retrieval**: run FTS5 keyword + cosine vector in parallel, fuse with reciprocal-rank fusion. (Zep/Hindsight validate multi-signal retrieval as the accuracy unlock.)
- **Per-client/per-project namespacing**: `client_id` is the hard boundary (separate DB file per client = physical isolation); `project_id` is a soft filter within a client.
- **Token-budget-aware injection**: maintain a 1-line index per memory category + small topic files; at SessionStart, select top-K by fused relevance under a fixed token cap (e.g., 800–1500 tokens), always prepending active `rejected_approaches` for the current project. This directly implements the community-validated "index + ~20-line files" anti-context-rot pattern.

### Cross-tool design
One shared SQLite backend + per-tool adapters: Claude Code (hooks + MCP), Cursor (MCP server + optional `.cursor/rules` injection block bounded by markers, à la MemNexus), Codex (MCP). The backend is identical; only the capture/inject front-ends differ.

### Distribution & packaging
- **Claude Code marketplace**: a GitHub repo with `.claude-plugin/marketplace.json` listing your plugin; users run `/plugin marketplace add <owner>/<repo>` then `/plugin install <name>@<marketplace>`. Plugin dir has `.claude-plugin/plugin.json`, `.mcp.json`, `hooks/`, `skills/`. Submit to `anthropics/claude-plugins-community` (SHA-pinned, auto-screened) for reach.
- **Cursor**: ship MCP config + install script.
- **Claude Desktop**: package as **`.mcpb` bundle** (formerly `.dxt`) via `npx @anthropic-ai/mcpb` (`mcpb init`, `mcpb pack`) — a zip of the server + `manifest.json`, one-click install.
- **MCP registries**: publish `server.json` to the **official MCP Registry** (registry.modelcontextprotocol.io, reverse-DNS namespace, prove name ownership via GitHub OIDC; automate via CI). Directories **Smithery** (`smithery mcp publish`), **Glama** (auto-crawls; claim listing), **mcp.so**, **PulseMCP** (hand-reviewed; getting into their newsletter is high-signal) then pick it up downstream.

## 7. Phasing & Effort

- **v0 (weeks 1–4):** Claude Code plugin: SessionStart injection + SessionEnd/Stop capture (async worker), SQLite schema, FTS5+vector hybrid, the `rejected_approach` type, per-client DB isolation, local web viewer. Ship to Claude Code marketplace + official MCP Registry. *Realistic for a strong solo systems dev: ~3–4 weeks given claude-mem is an open reference architecture (Apache-2.0) you can study.*
- **v1 (weeks 5–8):** Cursor + Codex adapters; `.mcpb` bundle for Claude Desktop; token-budget tuner; import/mirror from native Auto Memory.
- **v2 (months 3+):** Paid cloud-sync tier (cross-machine), team sharing, hosted embeddings.

## 8. Distribution & GTM
Lead where the pain is voiced: r/ClaudeAI, r/ClaudeCode, r/cursor, Hacker News (Show HN), dev.to. The hook for posts is the *differentiator demo*: "watch Claude stop re-suggesting the library you already rejected" and "Client A context can never leak into Client B." Get into the Claude Code community marketplace and PulseMCP's directory/newsletter. claude-mem's 82K+ stars prove the audience exists and installs this category readily.

## 9. Monetization
- **Free local core** (the wedge: auto-capture, isolation, why-capture, cross-tool, all local/private).
- **Paid Pro (~$8–12/mo self-serve):** cloud sync across machines, hosted embeddings, web dashboard.
- **Team (~$20–40/user/mo):** shared client/project memory, RBAC, audit. (Mirrors Hindsight/claude-mem's team-server direction.)

## 10. Key Risks & Open Questions
- **Native memory absorbs the category.** Mitigation: cross-tool + the two data-model features no single vendor will replicate. *Benchmark to watch:* if Anthropic ships cross-project AND "why" capture AND a client boundary, the wedge narrows — pivot to pure cross-tool + team.
- **"Context poisoning" / memory-makes-it-worse.** This is the real product risk (HN skeptics). Mitigation: conservative injection, token caps, easy edit/delete, and prioritizing *rejections* (which are high-signal) over raw observations.
- **Cursor Memories status uncertain** (one source says removed in v2.1.x → Rules). **Verify current Cursor memory behavior before building the Cursor adapter.**
- **Two products both named "OpenMemory"** (Mem0's deprecated one vs CaviraOSS's active one) — verify which you're benchmarking against.
- **Embedding model packaging** across OS/arch (ONNX) is the fiddliest engineering bit; budget time.
- Verify all hook event names/fields against current Claude Code docs at build time — the API moves fast (event count went 12→13→~17 across 2025–2026).

## 11. Tech Stack Decision

> Decision owner prefers **Go**. The honest finding below: Go is not just the preference, it is genuinely the right core for *this specific* product — with exactly one subsystem (local embedding inference) where Go is weakest and where Rust/Python win. The correct move is to **stay in Go for the core and isolate the embedding subsystem behind an interface**, not to rewrite everything in Rust.

### Verdict (short)
**Go for the core** (MCP server, hooks, storage, retrieval orchestration, async capture worker, web viewer). **Isolate embeddings behind an `Embedder` interface** with three pluggable backends: a bundled **sidecar** (Rust `fastembed-rs`, the zero-config default), **Ollama** over HTTP (if present), and an **FTS5-only fallback** (no embeddings — still useful). Do not adopt Rust as the primary language; capture its one real advantage (ONNX inference) surgically via the sidecar. Python and TypeScript lose on the thing that *is* the product's wedge: a fast-starting, single-binary, zero-config local tool.

### Why Go is the right core here (not just preference)
1. **Hook cold-start latency is a first-class requirement.** Hooks fire constantly — SessionStart, UserPromptSubmit, Stop, PostToolUse. Section 6 demands `<1s` (ideally `<10ms` to enqueue). A Go binary starts in single-digit milliseconds; a Node hook pays ~100–300ms interpreter+`require` startup *per fire*, and Python ~50–150ms. On a per-tool-call hook this is the difference between invisible and annoying. **This alone is a strong, objective argument for a compiled language over Node/Python.**
2. **The async-capture pattern is literally goroutines + channels.** "Enqueue an observation in <10ms, a background worker does slow AI compression (5–30s) asynchronously" (the claude-mem pattern) is Go's canonical concurrency shape. No `tokio` ceremony, no GIL, no single-threaded event-loop CPU contention.
3. **Single static binary = the zero-config wedge.** "No API key, no cloud, private by default" is far easier to deliver as one cross-compiled binary than as an interpreter + `node_modules`/venv.
4. **Cross-compile to every OS/arch** from one machine (caveat: CGo, see embeddings).

### Honest cross-language comparison (by subsystem)
| Subsystem | **Go** | Rust | Python | TS / Node |
|---|---|---|---|---|
| MCP SDK maturity | Good — official `modelcontextprotocol/go-sdk` (co-built w/ Google), solid; *verify version* | Official `rmcp`, real but less mature | Official, very mature | Official **reference** SDK, most mature |
| Hook cold-start | **Excellent** (~ms) | **Excellent** (~ms) | Poor (~50–150ms) | Mediocre (~100–300ms) |
| Async capture worker | **Excellent** (goroutines/channels) | Good (tokio, more ceremony) | OK (asyncio/threads, GIL) | Good (event loop; single-thread CPU) |
| SQLite + FTS5 | Good — `mattn/go-sqlite3` (CGo, full FTS5 + extension load) or `modernc.org/sqlite` (pure-Go, weaker extension story) | **Excellent** (`rusqlite`, bundled, FTS5) | **Excellent** (stdlib `sqlite3`) | **Excellent** (`better-sqlite3`) |
| **Local ONNX embeddings** | **Weak** — CGo (`onnxruntime_go`, `hugot`); packaging pain | **Excellent** (`ort` + `fastembed-rs`) | **Excellent** (`fastembed`, `sentence-transformers`) | Good (`transformers.js`, `onnxruntime-node`) |
| Single-binary distribution | **Excellent** (CGo-permitting) | **Excellent** | Poor (PyInstaller/uv) | Poor-ish (SEA/`pkg`, or `.mcpb`) |
| `npx`/registry distribution | Binary or npm-wrapper | Binary or npm-wrapper | `uvx` | **Native** `npx` + Smithery (best) |
| Web viewer | Easy (`net/http` + `embed.FS`) | Easy (axum) | Easy | Easy |
| Reference code to copy | port concepts | port concepts | port concepts | **claude-mem is TS** — lowest-friction to study |

**Reading the table:** Go wins or ties on every row that matches *this product's* constraints (latency, concurrency, single-binary, privacy) and loses meaningfully on exactly one: local ONNX embeddings. Rust's only decisive lead is that same embeddings row — which is precisely why a Rust *sidecar*, not a Rust *rewrite*, is the answer. TS's leads (SDK maturity, `npx`, a TS reference impl in claude-mem) are real but are convenience, not architecture; they don't outweigh Node's hook-latency tax in a hook-driven design.

### The one hard part: local embeddings (resolve explicitly)
This is the subsystem section 10 already flags as "the fiddliest engineering bit." Decision: **define `type Embedder interface { Embed(ctx, []string) ([][]float32, error); Dims() int }` and ship multiple backends.**

- **Default (bundled sidecar) — Rust `fastembed-rs` (uses `ort`/ONNX Runtime).** A tiny standalone binary that reads text on stdin / a Unix socket and returns vectors. Ships `all-MiniLM-L6-v2` (~30MB, 384-dim) or `embeddinggemma-300m`. Keeps the Go core CGo-free and cross-compilable; the only platform-specific artifact is the sidecar + the ONNX Runtime shared lib. Two binaries, zero interpreters, fully local. *(A Python `fastembed` sidecar is the easier-to-write alternative but reintroduces an interpreter dependency — avoid for the zero-config default.)*
- **If present — Ollama** (`all-minilm` / `nomic-embed-text`) over its local HTTP API. Pure-Go HTTP client, no packaging at all. Great for users who already run Ollama; not zero-config for those who don't.
- **Always-available fallback — FTS5 keyword-only.** If no embedder is wired up, hybrid retrieval degrades to BM25/FTS5. Still ships value on day one; lets v0 land before the sidecar is polished.

**Phasing tie-in:** v0 can ship with **Ollama-or-FTS5-fallback** (pure Go, fast to build), and the **bundled Rust sidecar** lands in v1 alongside the Cursor/Codex adapters — matching section 7's timeline.

### Recommended component stack (bill of materials)
- **Language/runtime:** Go 1.22+ (core). Rust (single small sidecar crate, v1).
- **MCP:** `github.com/modelcontextprotocol/go-sdk` — `McpServer` + stdio transport. *Verify current version/stability at build time.*
- **SQLite:** start with `mattn/go-sqlite3` (CGo) for **full FTS5 + loadable-extension support** (needed for `sqlite-vec`); evaluate `modernc.org/sqlite` (pure-Go) only if you drop in-DB vectors and do cosine in-process. Decide once — it affects the CGo/cross-compile story.
- **Vector search:** `sqlite-vec` (Alex Garcia) as a loadable extension *or* in-process cosine over a `BLOB` embedding column. Fuse with FTS5 via **reciprocal-rank fusion** (per §6).
- **Embeddings:** `fastembed-rs` (sidecar) / Ollama HTTP / FTS5-fallback, behind the `Embedder` interface.
- **Web viewer:** Go `net/http` (or `chi`) serving a JSON API + a small SPA embedded via `embed.FS` (vanilla JS or a tiny Svelte/Preact build) — claude-mem's `:37777` viewer is the proof users want this.
- **Hooks:** thin shell wrappers (or direct binary invocation) calling Go subcommands (`memory hook session-start`, `… session-end`, etc.); the slow path detaches (`( … ) &>/dev/null & exit 0`).
- **Config:** directory→`client_id` mapping in a user config file (TOML/JSON, e.g. `~/.config/<name>/config.toml`); one SQLite DB file per `client_id` (physical isolation, §4 Axis 2).

### What else you need (beyond application code)
- **Packaging:** **GoReleaser** for the cross-platform build matrix (darwin/linux/windows × amd64/arm64); bundle the per-platform sidecar + ONNX Runtime shared lib in each archive.
- **Claude Code plugin:** `.claude-plugin/plugin.json`, `.mcp.json`, `hooks/`, `skills/`; a marketplace repo with `.claude-plugin/marketplace.json`; PR into `anthropics/claude-plugins-community`.
- **Claude Desktop:** `.mcpb` bundle via `npx @anthropic-ai/mcpb` (`mcpb init` / `mcpb pack`) with a `manifest.json`.
- **MCP registries:** `server.json` → official MCP Registry (reverse-DNS namespace, GitHub OIDC name proof, automated in CI); then Smithery / Glama / mcp.so / PulseMCP.
- **CI/CD:** GitHub Actions — build matrix, `go test` + golden-file retrieval tests, OIDC publish to the MCP registry, GoReleaser on tag.
- **Testing:** Go table-driven tests; **golden tests for retrieval quality** (fixed corpus → expected top-K) so injection changes don't silently regress; an isolation test asserting Client A queries can *never* return Client B rows.

### Open questions to verify at build time
- **Go MCP SDK** exact version and API surface (it's young — pin and re-check).
- **`onnxruntime_go`/`hugot` CGo vs cross-compile** reality — confirms whether an all-Go embedder is even worth attempting vs. going straight to the sidecar.
- **`sqlite-vec` + driver combo** — confirm extension loading works with your chosen Go SQLite driver, or commit to in-process cosine.

---

### Strategic note
This bet's logic: **win on features + execution in a niche the platform owners are structurally disincentivized to serve well** — Anthropic won't make Claude Code memory great inside Cursor. The MCP standard is your distribution and integration leverage, not a competitive threat. Ship the wedge feature first (`rejected_approach` + per-client isolation), prove it with a sharp demo on Reddit/HN/Product Hunt, and layer the cross-tool breadth and paid sync on top once the core reliability story lands.

> Companion project: a web-action recorder that exports macros as MCP tools (see its own repo/decision.md). Both share the same "win on features in a platform-owner blind spot, use MCP as distribution" logic.
