// Command recap is the local-first memory layer for coding agents.
//
// It runs in two modes:
//
//	recap serve            # MCP stdio server exposing the memory_* tools
//	recap hook <event>     # lifecycle hook handler (session-start, session-end, stop, ...)
//	recap version          # print version
//
// See CLAUDE.md and docs/TECH.md for the architecture and the working loop.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sandeepshekhar26/recap/internal/config"
	"github.com/sandeepshekhar26/recap/internal/embed"
	"github.com/sandeepshekhar26/recap/internal/hook"
	"github.com/sandeepshekhar26/recap/internal/mcp"
	"github.com/sandeepshekhar26/recap/internal/retrieval"
	"github.com/sandeepshekhar26/recap/internal/store"
	"github.com/sandeepshekhar26/recap/internal/viewer"
)

// promptInjectBudget caps the tokens injected per prompt by the
// user-prompt-submit hook (kept small to avoid context poisoning).
const promptInjectBudget = 500

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "0.0.0-dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "recap:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return fmt.Errorf("no subcommand given")
	}

	switch args[0] {
	case "serve":
		return cmdServe(args[1:])
	case "hook":
		return cmdHook(args[1:])
	case "viewer":
		return cmdViewer(args[1:])
	case "version", "--version", "-v":
		fmt.Println("recap", version)
		return nil
	case "help", "--help", "-h":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

// cmdServe starts the MCP stdio server, shutting down cleanly on SIGINT/SIGTERM.
func cmdServe(_ []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	sc, err := openScope(ctx, cwd)
	if err != nil {
		return err
	}
	defer sc.store.Close()

	return mcp.Serve(ctx, version, mcp.Deps{
		Store:     sc.store,
		Retriever: retrieval.New(sc.store, embed.Nop{}),
		ClientID:  sc.clientID,
		ProjectID: sc.projectID,
	})
}

// cmdViewer serves the local web UI for browsing/deleting the current client's
// memories, shutting down cleanly on SIGINT/SIGTERM.
func cmdViewer(args []string) error {
	fs := flag.NewFlagSet("viewer", flag.ContinueOnError)
	addr := fs.String("addr", "127.0.0.1:37788", "address to listen on")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	sc, err := openScope(ctx, cwd)
	if err != nil {
		return err
	}
	defer sc.store.Close()

	srv := &http.Server{
		Addr:    *addr,
		Handler: viewer.New(sc.store, sc.clientID, sc.projectID).Handler(),
	}
	go func() {
		<-ctx.Done()
		srv.Close()
	}()

	fmt.Fprintf(os.Stderr, "recap viewer: http://%s  (client=%s, project=%s)\n", *addr, sc.clientID, sc.projectID)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// scope is a resolved client/project plus its open per-client database.
type scope struct {
	store     *store.DB
	clientID  string
	projectID string
}

// openScope resolves the client_id (directory rules) and project_id (nearest
// .git) for cwd and opens that client's database. The caller must Close it.
func openScope(ctx context.Context, cwd string) (scope, error) {
	cfg, err := config.Load()
	if err != nil {
		return scope{}, err
	}
	clientID := cfg.ResolveClientID(cwd)
	dbPath, err := cfg.DBPath(clientID)
	if err != nil {
		return scope{}, err
	}
	db, err := store.Open(ctx, dbPath)
	if err != nil {
		return scope{}, err
	}
	return scope{store: db, clientID: clientID, projectID: store.ResolveProjectID(cwd)}, nil
}

// cmdHook handles a Claude Code lifecycle hook event. Hooks are best-effort: any
// operational error is logged to stderr but the process still exits 0, so a
// misconfigured recap never disrupts the coding session.
func cmdHook(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("hook: missing event name (e.g. session-start)")
	}
	if err := dispatchHook(args[0]); err != nil {
		fmt.Fprintln(os.Stderr, "recap hook:", err)
	}
	return nil
}

func dispatchHook(event string) error {
	in, err := hook.ParseInput(os.Stdin)
	if err != nil {
		return err
	}
	cwd := in.CWD
	if cwd == "" {
		if cwd, err = os.Getwd(); err != nil {
			return err
		}
	}
	ctx := context.Background()

	switch event {
	case "session-start":
		return hookSessionStart(ctx, cwd)
	case "user-prompt-submit":
		return hookUserPromptSubmit(ctx, cwd, in)
	case "session-end":
		return hookSessionEnd(ctx, cwd, in)
	case "stop":
		// Incremental per-response capture needs an LLM to compress the
		// transcript; deferred to v1 (with the Ollama/sidecar work). No-op.
		return nil
	default:
		return fmt.Errorf("unknown event %q", event)
	}
}

// hookSessionStart injects the project's rejections + relevant memories as
// additionalContext at session start.
func hookSessionStart(ctx context.Context, cwd string) error {
	sc, err := openScope(ctx, cwd)
	if err != nil {
		return err
	}
	defer sc.store.Close()

	res, err := retrieval.New(sc.store, embed.Nop{}).Recall(ctx,
		retrieval.Query{ClientID: sc.clientID, ProjectID: sc.projectID}, retrieval.DefaultTokenBudget)
	if err != nil {
		return err
	}
	out, err := hook.SessionStartContext(res)
	if err != nil {
		return err
	}
	if out != "" {
		fmt.Println(out)
	}
	return nil
}

// hookUserPromptSubmit injects a small set of prompt-relevant memories.
func hookUserPromptSubmit(ctx context.Context, cwd string, in hook.Input) error {
	if strings.TrimSpace(in.Prompt) == "" {
		return nil
	}
	sc, err := openScope(ctx, cwd)
	if err != nil {
		return err
	}
	defer sc.store.Close()

	res, err := retrieval.New(sc.store, embed.Nop{}).Recall(ctx,
		retrieval.Query{ClientID: sc.clientID, ProjectID: sc.projectID, Text: in.Prompt}, promptInjectBudget)
	if err != nil {
		return err
	}
	out, err := hook.PromptContext(res.Memories)
	if err != nil {
		return err
	}
	if out != "" {
		fmt.Println(out)
	}
	return nil
}

// hookSessionEnd records lightweight session bookkeeping. (LLM-based observation
// compression is deferred to v1; see docs/JOURNAL.md.)
func hookSessionEnd(ctx context.Context, cwd string, in hook.Input) error {
	if in.SessionID == "" {
		return nil
	}
	sc, err := openScope(ctx, cwd)
	if err != nil {
		return err
	}
	defer sc.store.Close()

	return sc.store.UpsertSession(ctx, store.Session{
		ID:        in.SessionID,
		ClientID:  sc.clientID,
		ProjectID: sc.projectID,
		EndedAt:   time.Now().Unix(),
	})
}

func usage() {
	fmt.Fprint(os.Stderr, `recap — local-first memory for coding agents

usage:
  recap serve            start the MCP stdio server
  recap hook <event>     handle a lifecycle hook (session-start|session-end|stop|user-prompt-submit)
  recap viewer [--addr]  serve the local web viewer (default 127.0.0.1:37788)
  recap version          print version
  recap help             show this help
`)
}
