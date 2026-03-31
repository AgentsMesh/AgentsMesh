package acp

import (
	"testing"
)

// --- Nil callbacks do not crash ---

func TestHandler_NilCallbacks_NoCrash(t *testing.T) {
	h := NewHandler(EventCallbacks{}, testLogger())

	// Agent message chunk
	params := mustMarshal(t, map[string]any{
		"sessionId": "s",
		"update": map[string]any{
			"sessionUpdate": "agent_message_chunk",
			"content":       map[string]any{"type": "text", "text": "hi"},
		},
	})
	h.HandleNotification("session/update", params)

	// Tool call
	params = mustMarshal(t, map[string]any{
		"sessionId": "s",
		"update": map[string]any{
			"sessionUpdate": "tool_call",
			"toolCallId":    "t1",
			"title":         "x",
			"status":        "running",
		},
	})
	h.HandleNotification("session/update", params)

	// Tool call update (completed — fires both OnToolCallUpdate and OnToolCallResult)
	params = mustMarshal(t, map[string]any{
		"sessionId": "s",
		"update": map[string]any{
			"sessionUpdate": "tool_call_update",
			"toolCallId":    "t1",
			"title":         "x",
			"status":        "completed",
		},
	})
	h.HandleNotification("session/update", params)

	// Plan
	params = mustMarshal(t, map[string]any{
		"sessionId": "s",
		"update": map[string]any{
			"sessionUpdate": "plan",
			"entries":       []map[string]any{},
		},
	})
	h.HandleNotification("session/update", params)

	// Thinking (agent_thought_chunk)
	params = mustMarshal(t, map[string]any{
		"sessionId": "s",
		"update": map[string]any{
			"sessionUpdate": "agent_thought_chunk",
			"content":       map[string]any{"text": "hmm"},
		},
	})
	h.HandleNotification("session/update", params)

	// Permission request
	params = mustMarshal(t, map[string]any{
		"sessionId": "s",
		"toolCall": map[string]any{
			"toolCallId": "tc-1",
			"title":      "t",
		},
		"options": []map[string]any{},
	})
	h.HandlePermissionRequest(1, params)
}
