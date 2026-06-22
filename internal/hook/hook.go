// Package hook implements recap's Claude Code lifecycle hook handlers. Hooks
// read a JSON event on stdin and (for injection events) print a
// hookSpecificOutput.additionalContext object on stdout, which Claude Code 2.1+
// injects silently into the session (docs/STUDY.md §2).
//
// Hooks must be fast and must never disrupt the session: the cmd layer runs them
// best-effort (diagnostics to stderr, exit 0).
package hook

import (
	"encoding/json"
	"io"

	"github.com/sandeepshekhar26/recap/internal/retrieval"
)

// Input is the subset of the hook event payload recap uses, across event types.
type Input struct {
	SessionID string `json:"session_id"`
	CWD       string `json:"cwd"`
	Source    string `json:"source"` // SessionStart: startup|resume|clear|compact
	Reason    string `json:"reason"` // SessionEnd: clear|logout|prompt_input_exit|other
	Prompt    string `json:"prompt"` // UserPromptSubmit
	Event     string `json:"hook_event_name"`
}

// ParseInput decodes a hook payload from r. Empty input is not an error (some
// invocations may pass nothing on stdin).
func ParseInput(r io.Reader) (Input, error) {
	var in Input
	if err := json.NewDecoder(r).Decode(&in); err != nil && err != io.EOF {
		return Input{}, err
	}
	return in, nil
}

// SessionStartContext builds the stdout JSON that injects project memory at
// session start. Returns "" (emit nothing) when there is nothing to inject.
func SessionStartContext(res retrieval.Result) (string, error) {
	if !res.HasContent() {
		return "", nil
	}
	return contextJSON("SessionStart", "recap — project memory:\n"+retrieval.FormatRecall(res))
}

// PromptContext builds a lightweight, prompt-relevant injection for
// UserPromptSubmit. It deliberately injects only keyword-matched memories (not
// the always-on rejections, which session start already surfaced) to avoid
// context-poisoning via repetition (decision.md §10). Returns "" when empty.
func PromptContext(ms []retrieval.ScoredMemory) (string, error) {
	if len(ms) == 0 {
		return "", nil
	}
	return contextJSON("UserPromptSubmit", "recap — possibly relevant:\n"+retrieval.FormatMemories(ms))
}

func contextJSON(event, body string) (string, error) {
	type specific struct {
		HookEventName     string `json:"hookEventName"`
		AdditionalContext string `json:"additionalContext"`
	}
	out := struct {
		HookSpecificOutput specific `json:"hookSpecificOutput"`
	}{specific{HookEventName: event, AdditionalContext: body}}

	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
