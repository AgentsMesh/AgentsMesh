package acp

import (
	"testing"
)

// --- Thinking update (session/update type=thinking) ---

func TestHandler_ThinkingUpdate(t *testing.T) {
	var received []ThinkingUpdate
	h := NewHandler(EventCallbacks{
		OnThinkingUpdate: func(_ string, update ThinkingUpdate) {
			received = append(received, update)
		},
	}, testLogger())

	params := mustMarshal(t, map[string]any{
		"session_id": "sess-1",
		"type":       "thinking",
		"data":       map[string]any{"text": "Let me think about this..."},
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
		"session_id": "sess-1",
		"type":       "unknown_type",
		"data":       map[string]any{},
	})
	h.HandleNotification("session/update", params)
}
