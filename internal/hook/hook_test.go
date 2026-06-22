package hook

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sandeepshekhar26/recap/internal/retrieval"
	"github.com/sandeepshekhar26/recap/internal/store"
)

func TestParseInput(t *testing.T) {
	in, err := ParseInput(strings.NewReader(
		`{"session_id":"abc","cwd":"/work/x","source":"resume","hook_event_name":"SessionStart"}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if in.SessionID != "abc" || in.CWD != "/work/x" || in.Source != "resume" {
		t.Errorf("unexpected parse: %+v", in)
	}
}

func TestParseInputEmpty(t *testing.T) {
	if _, err := ParseInput(strings.NewReader("")); err != nil {
		t.Errorf("empty input should not error, got %v", err)
	}
}

func TestSessionStartContext(t *testing.T) {
	// empty result -> no injection
	if out, err := SessionStartContext(retrieval.Result{}); err != nil || out != "" {
		t.Errorf("empty result: got (%q, %v), want (\"\", nil)", out, err)
	}

	res := retrieval.Result{Rejections: []store.Rejection{
		{Approach: "use Mongo", ReasonRejected: "no transactions"},
	}}
	out, err := SessionStartContext(res)
	if err != nil {
		t.Fatalf("context: %v", err)
	}

	var parsed struct {
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if parsed.HookSpecificOutput.HookEventName != "SessionStart" {
		t.Errorf("hookEventName = %q", parsed.HookSpecificOutput.HookEventName)
	}
	if !strings.Contains(parsed.HookSpecificOutput.AdditionalContext, "use Mongo") {
		t.Errorf("additionalContext missing rejection: %q", parsed.HookSpecificOutput.AdditionalContext)
	}
}

func TestPromptContextEmpty(t *testing.T) {
	if out, err := PromptContext(nil); err != nil || out != "" {
		t.Errorf("empty memories: got (%q, %v), want (\"\", nil)", out, err)
	}
}
