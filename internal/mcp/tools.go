package mcp

import (
	"context"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// toolNames is the canonical list of memory_* tools recap exposes. Tests assert
// the server advertises exactly these.
var toolNames = []string{
	"memory_recall",
	"memory_search",
	"memory_save",
	"memory_save_rejection",
	"memory_list_rejections",
}

// registerTools adds the five memory_* tools. They are no-ops for now
// (Phase v0 §1) so MCP clients can discover them via tools/list. Behavior is
// implemented in Phase v0 §4 (rejections) and §5 (save/recall/search).
func registerTools(s *sdk.Server) {
	addStub[RecallInput](s, "memory_recall",
		"Recall memories relevant to the current project/context.", "§5")
	addStub[SearchInput](s, "memory_search",
		"Search stored memories by keyword or semantic query.", "§5")
	addStub[SaveInput](s, "memory_save",
		"Save a decision, convention, or summary, with optional rationale.", "§5")
	addStub[SaveRejectionInput](s, "memory_save_rejection",
		"Record a rejected approach and why, so it is never re-suggested.", "§4")
	addStub[ListRejectionsInput](s, "memory_list_rejections",
		"List rejected approaches for the current project.", "§4")
}

// RecallInput is the argument schema for memory_recall.
type RecallInput struct {
	Query     string `json:"query,omitempty" jsonschema:"what to recall; empty returns the most relevant recent memories"`
	ProjectID string `json:"project_id,omitempty" jsonschema:"optional project filter; defaults to the current project"`
}

// SearchInput is the argument schema for memory_search.
type SearchInput struct {
	Query string `json:"query" jsonschema:"keyword or semantic search query"`
}

// SaveInput is the argument schema for memory_save.
type SaveInput struct {
	Type      string `json:"type" jsonschema:"one of: decision, convention, session_summary"`
	Content   string `json:"content" jsonschema:"the memory text to store"`
	Rationale string `json:"rationale,omitempty" jsonschema:"why this is true/decided (optional)"`
}

// SaveRejectionInput is the argument schema for memory_save_rejection — recap's
// differentiator (decision.md §4).
type SaveRejectionInput struct {
	Approach       string `json:"approach" jsonschema:"the approach that was rejected"`
	ReasonRejected string `json:"reason_rejected" jsonschema:"why it was rejected"`
}

// ListRejectionsInput is the argument schema for memory_list_rejections.
type ListRejectionsInput struct {
	ProjectID string `json:"project_id,omitempty" jsonschema:"optional project filter; defaults to the current project"`
}

// addStub registers a tool whose handler reports that it is not implemented yet.
// The In type parameter supplies the tool's input JSON schema.
func addStub[In any](s *sdk.Server, name, desc, phase string) {
	sdk.AddTool[In, any](s, &sdk.Tool{Name: name, Description: desc},
		func(_ context.Context, _ *sdk.CallToolRequest, _ In) (*sdk.CallToolResult, any, error) {
			return &sdk.CallToolResult{
				Content: []sdk.Content{
					&sdk.TextContent{Text: fmt.Sprintf(
						"%s is not implemented yet (ROADMAP Phase v0 %s).", name, phase)},
				},
			}, nil, nil
		})
}
