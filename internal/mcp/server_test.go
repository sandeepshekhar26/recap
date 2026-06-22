package mcp

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sandeepshekhar26/recap/internal/embed"
	"github.com/sandeepshekhar26/recap/internal/retrieval"
	"github.com/sandeepshekhar26/recap/internal/store"
)

func testDeps(t *testing.T) Deps {
	t.Helper()
	db, err := store.Open(context.Background(), filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return Deps{
		Store:     db,
		Retriever: retrieval.New(db, embed.Nop{}),
		ClientID:  "c",
		ProjectID: "p",
	}
}

// connect wires an in-process client to a server built from d.
func connect(t *testing.T, d Deps) *sdk.ClientSession {
	t.Helper()
	ctx := context.Background()
	serverT, clientT := sdk.NewInMemoryTransports()

	srvSession, err := newServer("test", d).Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { srvSession.Close() })

	cs, err := sdk.NewClient(&sdk.Implementation{Name: "test-client", Version: "0"}, nil).
		Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

func callText(t *testing.T, cs *sdk.ClientSession, name string, args any) string {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*sdk.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

func TestServerListsTools(t *testing.T) {
	cs := connect(t, testDeps(t))

	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	got := make(map[string]bool, len(res.Tools))
	for _, tool := range res.Tools {
		got[tool.Name] = true
		if tool.Description == "" {
			t.Errorf("tool %q has empty description", tool.Name)
		}
	}
	if len(got) != len(toolNames) {
		t.Errorf("got %d tools, want %d", len(got), len(toolNames))
	}
	for _, name := range toolNames {
		if !got[name] {
			t.Errorf("missing tool %q", name)
		}
	}
}

// TestToolsEndToEnd drives the real handlers: save a rejection and a memory,
// then recall and assert both surface, with the rejection flagged.
func TestToolsEndToEnd(t *testing.T) {
	cs := connect(t, testDeps(t))

	if out := callText(t, cs, "memory_save_rejection", map[string]any{
		"approach": "use MongoDB", "reason_rejected": "need ACID transactions",
	}); !strings.Contains(out, "Recorded rejected approach") {
		t.Errorf("save_rejection: unexpected output %q", out)
	}

	if out := callText(t, cs, "memory_save", map[string]any{
		"type": "decision", "content": "use PostgreSQL", "rationale": "ACID + team familiarity",
	}); !strings.Contains(out, "Saved decision memory") {
		t.Errorf("save: unexpected output %q", out)
	}

	// Rejections always surface regardless of the query.
	out := callText(t, cs, "memory_recall", map[string]any{"query": "anything"})
	if !strings.Contains(out, "Already ruled out") || !strings.Contains(out, "use MongoDB") {
		t.Errorf("recall missing always-on rejection: %q", out)
	}
	// Keyword-matching query surfaces the saved memory (keyword-only in v0).
	out = callText(t, cs, "memory_recall", map[string]any{"query": "PostgreSQL"})
	if !strings.Contains(out, "use PostgreSQL") {
		t.Errorf("recall missing keyword-matched memory: %q", out)
	}
}

func TestToolValidation(t *testing.T) {
	cs := connect(t, testDeps(t))

	res, err := cs.CallTool(context.Background(), &sdk.CallToolParams{
		Name: "memory_save", Arguments: map[string]any{"type": "bogus", "content": "x"},
	})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if !res.IsError {
		t.Error("expected IsError for invalid memory type")
	}
}
