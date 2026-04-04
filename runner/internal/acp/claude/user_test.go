package claude

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

func TestTransport_UserToolResult(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "user", "message": map[string]any{
		"role": "user", "content": []map[string]any{{
			"type": "tool_result", "tool_use_id": "t1", "content": "file data", "is_error": false,
		}},
	}})
	f.Drain()
	_, _, _, results := f.Snap()
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ToolCallID != "t1" || !results[0].Success || results[0].ResultText != "file data" {
		t.Errorf("result = %+v", results[0])
	}
}

func TestTransport_UserToolResult_Error(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "user", "message": map[string]any{
		"role": "user", "content": []map[string]any{{
			"type": "tool_result", "tool_use_id": "t2", "content": "denied", "is_error": true,
		}},
	}})
	f.Drain()
	_, _, _, results := f.Snap()
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Success || results[0].ErrorMessage != "denied" {
		t.Errorf("result = %+v", results[0])
	}
}

func TestTransport_UserPlainText(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "user", "message": map[string]any{
		"role": "user", "content": "just text",
	}})
	f.Drain()
	_, _, _, results := f.Snap()
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestTransport_ToolResultContentArray(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "user", "message": map[string]any{
		"role": "user", "content": []map[string]any{{
			"type": "tool_result", "tool_use_id": "ta",
			"content":  []map[string]any{{"type": "text", "text": "a"}, {"type": "text", "text": "b"}},
			"is_error": false,
		}},
	}})
	f.Drain()
	_, _, _, results := f.Snap()
	if len(results) != 1 || results[0].ResultText != "a\nb" {
		t.Errorf("result = %+v", results)
	}
}

func TestExtractToolResultText(t *testing.T) {
	cases := []struct {
		name string
		raw  json.RawMessage
		want string
	}{
		{"string", json.RawMessage(`"simple"`), "simple"},
		{"array", json.RawMessage(`[{"type":"text","text":"a"},{"type":"text","text":"b"}]`), "a\nb"},
		{"empty", nil, ""},
		{"fallback", json.RawMessage(`42`), "42"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExtractToolResultText(tc.raw); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTransport_SystemNonInit(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "system", "subtype": "api_retry"})
	f.Drain()
	states, _, _, _ := f.Snap()
	if len(states) != 0 {
		t.Errorf("expected 0 state changes, got %d", len(states))
	}
}

func TestTransport_ResultSuccess(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "result", "subtype": "success"})
	f.Drain()
	states, _, _, _ := f.Snap()
	if len(states) != 1 || states[0] != acp.StateIdle {
		t.Errorf("states = %v", states)
	}
}

func TestTransport_ResultError(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "result", "subtype": "error_max_turns", "result": "exceeded"})
	f.Drain()
	states, _, _, _ := f.Snap()
	if len(states) != 1 || states[0] != acp.StateIdle {
		t.Errorf("states = %v", states)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.LogMessages) != 1 || !strings.Contains(f.LogMessages[0], "error_max_turns") {
		t.Errorf("logs = %v", f.LogMessages)
	}
}
