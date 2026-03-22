package acp

import (
	"encoding/json"
	"testing"
)

// --- Nil callbacks do not crash ---

func TestHandler_NilCallbacks_NoCrash(t *testing.T) {
	h := NewHandler(EventCallbacks{}, testLogger())

	// Content
	params := mustMarshal(t, map[string]any{
		"session_id": "s", "type": "content",
		"data": map[string]any{"text": "hi", "role": "assistant"},
	})
	h.HandleNotification("session/update", params)

	// Tool call
	params = mustMarshal(t, map[string]any{
		"session_id": "s", "type": "tool_call",
		"data": map[string]any{"tool_call_id": "t1", "tool_name": "x", "status": "running"},
	})
	h.HandleNotification("session/update", params)

	// Tool result
	params = mustMarshal(t, map[string]any{
		"session_id": "s", "type": "tool_result",
		"data": map[string]any{"tool_call_id": "t1", "tool_name": "x", "success": true},
	})
	h.HandleNotification("session/update", params)

	// Plan
	params = mustMarshal(t, map[string]any{
		"session_id": "s", "type": "plan",
		"data": map[string]any{"steps": []map[string]any{}},
	})
	h.HandleNotification("session/update", params)

	// Thinking
	params = mustMarshal(t, map[string]any{
		"session_id": "s", "type": "thinking",
		"data": map[string]any{"text": "hmm"},
	})
	h.HandleNotification("session/update", params)

	// Session complete
	h.HandleNotification("session/complete", json.RawMessage(`{}`))

	// Permission request
	params = mustMarshal(t, map[string]any{
		"session_id": "s", "request_id": "r", "tool_name": "t",
	})
	h.HandleNotification("permission/request", params)
}
