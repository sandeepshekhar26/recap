// Package mcp implements recap's Model Context Protocol server: the stdio
// server exposed to coding agents (Claude Code, Cursor, Codex) and the
// memory_* tool handlers, wired to the storage and retrieval layers.
package mcp

import (
	"context"
	"errors"
	"io"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sandeepshekhar26/recap/internal/retrieval"
	"github.com/sandeepshekhar26/recap/internal/store"
)

// Deps are the dependencies the tool handlers need. ClientID and ProjectID are
// resolved once at startup from the server's working directory; every tool call
// operates within that client/project scope.
type Deps struct {
	Store     *store.DB
	Retriever *retrieval.Retriever
	ClientID  string
	ProjectID string
}

// Serve runs the recap MCP server over stdio. It blocks until the client
// disconnects or ctx is cancelled. A normal client disconnect (stdin EOF) or a
// cancelled context is a clean shutdown, not an error.
func Serve(ctx context.Context, version string, d Deps) error {
	err := newServer(version, d).Run(ctx, &sdk.StdioTransport{})
	if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

// newServer builds a server with the memory_* tools registered. Shared by Serve
// and the tests (via an in-memory transport).
func newServer(version string, d Deps) *sdk.Server {
	s := sdk.NewServer(&sdk.Implementation{
		Name:    "recap",
		Title:   "recap — local-first memory for coding agents",
		Version: version,
	}, nil)
	registerTools(s, d)
	return s
}
