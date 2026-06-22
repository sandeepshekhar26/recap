package mcp

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sandeepshekhar26/recap/internal/retrieval"
	"github.com/sandeepshekhar26/recap/internal/store"
)

// toolNames is the canonical list of memory_* tools recap exposes.
var toolNames = []string{
	"memory_recall",
	"memory_search",
	"memory_save",
	"memory_save_rejection",
	"memory_list_rejections",
}

// registerTools adds the five memory_* tools, wired to the storage and
// retrieval layers via d.
func registerTools(s *sdk.Server, d Deps) {
	sdk.AddTool(s, &sdk.Tool{
		Name:        "memory_recall",
		Description: "Recall memories and active rejected approaches relevant to the current project. Call at the start of a task to avoid re-deciding settled questions or re-suggesting ruled-out approaches.",
	}, d.recall)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "memory_search",
		Description: "Search stored memories by keyword or semantic query.",
	}, d.search)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "memory_save",
		Description: "Save a decision, convention, or session summary, with optional rationale (the 'why').",
	}, d.save)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "memory_save_rejection",
		Description: "Record an approach that was tried or considered and rejected, plus why. Surfaced first on recall so it is never re-suggested.",
	}, d.saveRejection)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "memory_list_rejections",
		Description: "List rejected approaches for the current project.",
	}, d.listRejections)
}

// --- input schemas ---

// RecallInput is the argument schema for memory_recall.
type RecallInput struct {
	Query string `json:"query,omitempty" jsonschema:"what to recall; empty returns the most relevant recent memories plus all active rejections"`
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

// ListRejectionsInput is the (empty) argument schema for memory_list_rejections.
type ListRejectionsInput struct{}

// --- handlers ---

func (d Deps) recall(ctx context.Context, _ *sdk.CallToolRequest, in RecallInput) (*sdk.CallToolResult, any, error) {
	res, err := d.Retriever.Recall(ctx, retrieval.Query{
		ClientID: d.ClientID, ProjectID: d.ProjectID, Text: in.Query,
	}, retrieval.DefaultTokenBudget)
	if err != nil {
		return nil, nil, err
	}
	return text(retrieval.FormatRecall(res)), nil, nil
}

func (d Deps) search(ctx context.Context, _ *sdk.CallToolRequest, in SearchInput) (*sdk.CallToolResult, any, error) {
	res, err := d.Retriever.Recall(ctx, retrieval.Query{
		ClientID: d.ClientID, ProjectID: d.ProjectID, Text: in.Query,
	}, retrieval.DefaultTokenBudget)
	if err != nil {
		return nil, nil, err
	}
	return text(retrieval.FormatMemories(res.Memories)), nil, nil
}

func (d Deps) save(ctx context.Context, _ *sdk.CallToolRequest, in SaveInput) (*sdk.CallToolResult, any, error) {
	mt := store.MemoryType(in.Type)
	if !mt.Valid() {
		return errText(fmt.Sprintf("invalid type %q; want one of decision, convention, session_summary", in.Type)), nil, nil
	}
	if strings.TrimSpace(in.Content) == "" {
		return errText("content is required"), nil, nil
	}
	id, err := d.Store.SaveMemory(ctx, store.Memory{
		ClientID: d.ClientID, ProjectID: d.ProjectID,
		Type: mt, Content: in.Content, Rationale: in.Rationale,
	})
	if err != nil {
		return nil, nil, err
	}
	return text(fmt.Sprintf("Saved %s memory #%d.", mt, id)), nil, nil
}

func (d Deps) saveRejection(ctx context.Context, _ *sdk.CallToolRequest, in SaveRejectionInput) (*sdk.CallToolResult, any, error) {
	if strings.TrimSpace(in.Approach) == "" || strings.TrimSpace(in.ReasonRejected) == "" {
		return errText("both approach and reason_rejected are required"), nil, nil
	}
	id, err := d.Store.SaveRejection(ctx, store.Rejection{
		ClientID: d.ClientID, ProjectID: d.ProjectID,
		Approach: in.Approach, ReasonRejected: in.ReasonRejected,
	})
	if err != nil {
		return nil, nil, err
	}
	return text(fmt.Sprintf("Recorded rejected approach #%d. It will be surfaced on recall so it is not re-suggested.", id)), nil, nil
}

func (d Deps) listRejections(ctx context.Context, _ *sdk.CallToolRequest, _ ListRejectionsInput) (*sdk.CallToolResult, any, error) {
	rs, err := d.Store.ListRejections(ctx, d.ClientID, d.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	return text(retrieval.FormatRejections(rs)), nil, nil
}

// --- result helpers ---

func text(s string) *sdk.CallToolResult {
	return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: s}}}
}

func errText(s string) *sdk.CallToolResult {
	return &sdk.CallToolResult{IsError: true, Content: []sdk.Content{&sdk.TextContent{Text: s}}}
}
