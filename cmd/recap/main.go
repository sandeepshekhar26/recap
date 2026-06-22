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
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sandeepshekhar26/recap/internal/config"
	"github.com/sandeepshekhar26/recap/internal/embed"
	"github.com/sandeepshekhar26/recap/internal/mcp"
	"github.com/sandeepshekhar26/recap/internal/retrieval"
	"github.com/sandeepshekhar26/recap/internal/store"
)

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

// cmdHook handles a Claude Code lifecycle hook event. See ROADMAP Phase v0 §6.
func cmdHook(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("hook: missing event name (e.g. session-start)")
	}
	return fmt.Errorf("hook %q: not implemented yet (ROADMAP Phase v0 §6)", args[0])
}

func usage() {
	fmt.Fprint(os.Stderr, `recap — local-first memory for coding agents

usage:
  recap serve            start the MCP stdio server
  recap hook <event>     handle a lifecycle hook (session-start|session-end|stop|user-prompt-submit)
  recap version          print version
  recap help             show this help
`)
}
