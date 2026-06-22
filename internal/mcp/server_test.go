package mcp

import (
	"context"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestServerListsTools connects an in-process client to the server over an
// in-memory transport and asserts it advertises exactly the five memory_*
// tools. This is the Phase v0 §1 acceptance check: "a client can list them".
func TestServerListsTools(t *testing.T) {
	ctx := context.Background()
	serverT, clientT := sdk.NewInMemoryTransports()

	srv := newServer("test")
	srvSession, err := srv.Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	defer srvSession.Close()

	client := sdk.NewClient(&sdk.Implementation{Name: "test-client", Version: "0"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer cs.Close()

	res, err := cs.ListTools(ctx, nil)
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
		t.Errorf("got %d tools, want %d: %v", len(got), len(toolNames), res.Tools)
	}
	for _, name := range toolNames {
		if !got[name] {
			t.Errorf("missing tool %q", name)
		}
	}
}
