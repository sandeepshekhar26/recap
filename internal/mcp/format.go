package mcp

import (
	"fmt"
	"strings"

	"github.com/sandeepshekhar26/recap/internal/retrieval"
	"github.com/sandeepshekhar26/recap/internal/store"
)

// formatRecall renders a recall Result as human/agent-readable text: active
// rejections first (highest signal), then relevant memories. This is reused for
// the SessionStart hook injection (§6).
func formatRecall(res retrieval.Result) string {
	if len(res.Rejections) == 0 && len(res.Memories) == 0 {
		return "No memories yet for this project."
	}
	var b strings.Builder
	if len(res.Rejections) > 0 {
		b.WriteString("Already ruled out (do not re-suggest):\n")
		for _, r := range res.Rejections {
			fmt.Fprintf(&b, "- %s — because %s\n", r.Approach, r.ReasonRejected)
		}
	}
	if len(res.Memories) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("Relevant memories:\n")
		for _, m := range res.Memories {
			b.WriteString(formatMemoryLine(m.Memory))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// formatMemories renders just a list of memories (used by memory_search).
func formatMemories(ms []retrieval.ScoredMemory) string {
	if len(ms) == 0 {
		return "No matching memories."
	}
	var b strings.Builder
	for _, m := range ms {
		b.WriteString(formatMemoryLine(m.Memory))
	}
	return strings.TrimRight(b.String(), "\n")
}

// formatRejections renders a list of rejected approaches (memory_list_rejections).
func formatRejections(rs []store.Rejection) string {
	if len(rs) == 0 {
		return "No rejected approaches recorded for this project."
	}
	var b strings.Builder
	for _, r := range rs {
		fmt.Fprintf(&b, "- %s — because %s\n", r.Approach, r.ReasonRejected)
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatMemoryLine(m store.Memory) string {
	if strings.TrimSpace(m.Rationale) != "" {
		return fmt.Sprintf("- [%s] %s (why: %s)\n", m.Type, m.Content, m.Rationale)
	}
	return fmt.Sprintf("- [%s] %s\n", m.Type, m.Content)
}
