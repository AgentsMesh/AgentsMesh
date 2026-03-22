package acp

import (
	"encoding/json"
	"io"
	"log/slog"
	"sync"
	"testing"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHandler_ContentUpdate(t *testing.T) {
	var received []ContentChunk
	var mu sync.Mutex
	h := NewHandler(EventCallbacks{
		OnContentChunk: func(sessionID string, chunk ContentChunk) {
			mu.Lock()
			defer mu.Unlock()
			received = append(received, chunk)
		},
	}, testLogger())

	params := mustMarshal(t, map[string]any{
		"session_id": "sess-1",
		"type":       "content",
		"data":       map[string]any{"text": "Hello world", "role": "assistant"},
	})
	h.HandleNotification("session/update", params)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(received))
	}
	if received[0].Text != "Hello world" {
		t.Errorf("Text = %q, want %q", received[0].Text, "Hello world")
	}
	if received[0].Role != "assistant" {
		t.Errorf("Role = %q, want %q", received[0].Role, "assistant")
	}
}

func TestHandler_ToolCallUpdate(t *testing.T) {
	var received []ToolCallUpdate
	var mu sync.Mutex
	h := NewHandler(EventCallbacks{
		OnToolCallUpdate: func(sessionID string, update ToolCallUpdate) {
			mu.Lock()
			defer mu.Unlock()
			received = append(received, update)
		},
	}, testLogger())

	params := mustMarshal(t, map[string]any{
		"session_id": "sess-1",
		"type":       "tool_call",
		"data": map[string]any{
			"tool_call_id":   "tc-1",
			"tool_name":      "read_file",
			"status":         "running",
			"arguments_json": `{"path":"/tmp/test"}`,
		},
	})
	h.HandleNotification("session/update", params)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 update, got %d", len(received))
	}
	if received[0].ToolCallID != "tc-1" {
		t.Errorf("ToolCallID = %q, want %q", received[0].ToolCallID, "tc-1")
	}
	if received[0].ToolName != "read_file" {
		t.Errorf("ToolName = %q, want %q", received[0].ToolName, "read_file")
	}
	if received[0].Status != "running" {
		t.Errorf("Status = %q, want %q", received[0].Status, "running")
	}
}

func TestHandler_ToolResultUpdate(t *testing.T) {
	var received []ToolCallResult
	var mu sync.Mutex
	h := NewHandler(EventCallbacks{
		OnToolCallResult: func(sessionID string, result ToolCallResult) {
			mu.Lock()
			defer mu.Unlock()
			received = append(received, result)
		},
	}, testLogger())

	params := mustMarshal(t, map[string]any{
		"session_id": "sess-1",
		"type":       "tool_result",
		"data": map[string]any{
			"tool_call_id":  "tc-2",
			"tool_name":     "write_file",
			"success":       true,
			"result_text":   "file written",
			"error_message": "",
		},
	})
	h.HandleNotification("session/update", params)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 result, got %d", len(received))
	}
	if received[0].ToolCallID != "tc-2" {
		t.Errorf("ToolCallID = %q, want %q", received[0].ToolCallID, "tc-2")
	}
	if !received[0].Success {
		t.Error("Success should be true")
	}
	if received[0].ResultText != "file written" {
		t.Errorf("ResultText = %q, want %q", received[0].ResultText, "file written")
	}
}

func TestHandler_ToolResultUpdate_Failure(t *testing.T) {
	var received []ToolCallResult
	h := NewHandler(EventCallbacks{
		OnToolCallResult: func(_ string, result ToolCallResult) {
			received = append(received, result)
		},
	}, testLogger())

	params := mustMarshal(t, map[string]any{
		"session_id": "sess-1",
		"type":       "tool_result",
		"data": map[string]any{
			"tool_call_id":  "tc-3",
			"tool_name":     "exec",
			"success":       false,
			"result_text":   "",
			"error_message": "permission denied",
		},
	})
	h.HandleNotification("session/update", params)

	if len(received) != 1 {
		t.Fatalf("expected 1 result, got %d", len(received))
	}
	if received[0].Success {
		t.Error("Success should be false")
	}
	if received[0].ErrorMessage != "permission denied" {
		t.Errorf("ErrorMessage = %q, want %q", received[0].ErrorMessage, "permission denied")
	}
}

func TestHandler_PlanUpdate(t *testing.T) {
	var received []PlanUpdate
	h := NewHandler(EventCallbacks{
		OnPlanUpdate: func(_ string, update PlanUpdate) {
			received = append(received, update)
		},
	}, testLogger())

	params := mustMarshal(t, map[string]any{
		"session_id": "sess-1",
		"type":       "plan",
		"data": map[string]any{
			"steps": []map[string]any{
				{"title": "Read config", "status": "completed"},
				{"title": "Update code", "status": "in_progress"},
				{"title": "Run tests", "status": "pending"},
			},
		},
	})
	h.HandleNotification("session/update", params)

	if len(received) != 1 {
		t.Fatalf("expected 1 plan update, got %d", len(received))
	}
	if len(received[0].Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(received[0].Steps))
	}
	if received[0].Steps[0].Title != "Read config" {
		t.Errorf("Step[0].Title = %q, want %q", received[0].Steps[0].Title, "Read config")
	}
	if received[0].Steps[1].Status != "in_progress" {
		t.Errorf("Step[1].Status = %q, want %q", received[0].Steps[1].Status, "in_progress")
	}
}

func mustMarshal(t *testing.T, v any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("mustMarshal: %v", err)
	}
	return data
}
