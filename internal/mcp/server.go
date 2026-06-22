// Package mcp implements recap's Model Context Protocol server: the stdio
// server exposed to coding agents (Claude Code, Cursor, Codex) and the
// memory_* tool handlers.
//
// The tools are no-ops at this stage (ROADMAP Phase v0 §1) so that clients can
// discover them via tools/list; real behavior is wired against the storage
// layer in Phase v0 §4–§5.
package mcp

import (
	"context"
	"errors"
	"io"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Serve runs the recap MCP server over stdio. It blocks until the client
// disconnects or ctx is cancelled. A normal client disconnect (stdin EOF) or a
// cancelled context is a clean shutdown, not an error.
func Serve(ctx context.Context, version string) error {
	err := newServer(version).Run(ctx, &sdk.StdioTransport{})
	if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

// newServer builds a server with the memory_* tools registered. It is shared by
// Serve and the tests (via an in-memory transport).
func newServer(version string) *sdk.Server {
	s := sdk.NewServer(&sdk.Implementation{
		Name:    "recap",
		Title:   "recap — local-first memory for coding agents",
		Version: version,
	}, nil)
	registerTools(s)
	return s
}
