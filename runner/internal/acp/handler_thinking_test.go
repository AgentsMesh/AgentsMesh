package acp

import (
	"testing"
)

// --- Thinking update (session/update sessionUpdate=agent_thought_chunk) ---

func TestHandler_ThinkingUpdate(t *testing.T) {
	var received []ThinkingUpdate
	h := NewHandler(EventCallbacks{
		OnThinkingUpdate: func(_ string, update ThinkingUpdate) {
			received = append(received, update)
		},
	}, testLogger())

	params := mustMarshal(t, map[string]any{
		"sessionId": "sess-1",
		"update": map[string]any{
			"sessionUpdate": "agent_thought_chunk",
			"content":       map[string]any{"text": "Let me think about this..."},
		},
	})
	h.HandleNotification("session/update", params)

	if len(received) != 1 {
		t.Fatalf("expected 1 thinking update, got %d", len(received))
	}
	if received[0].Text != "Let me think about this..." {
		t.Errorf("Text = %q, want %q", received[0].Text, "Let me think about this...")
	}
}

// --- Unknown session/update type ---

func TestHandler_UnknownSessionUpdateType_NoCrash(t *testing.T) {
	h := NewHandler(EventCallbacks{}, testLogger())

	params := mustMarshal(t, map[string]any{
		"sessionId": "sess-1",
		"update": map[string]any{
			"sessionUpdate": "unknown_type",
		},
	})
	h.HandleNotification("session/update", params)
}
