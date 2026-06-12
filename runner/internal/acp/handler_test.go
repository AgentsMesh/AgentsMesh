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

func mustMarshal(t *testing.T, v any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("mustMarshal: %v", err)
	}
	return data
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
		"sessionId": "sess-1",
		"update": map[string]any{
			"sessionUpdate": "agent_message_chunk",
			"content":       map[string]any{"type": "text", "text": "Hello world"},
		},
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

func TestHandler_LoopalExtStripsPrefix(t *testing.T) {
	var gotSession, gotKind string
	var gotData json.RawMessage
	h := NewHandler(EventCallbacks{
		OnLoopalExt: func(sessionID, kind string, data json.RawMessage) {
			gotSession, gotKind, gotData = sessionID, kind, data
		},
	}, testLogger())

	params := mustMarshal(t, map[string]any{
		"sessionId": "sess-1",
		"data":      map[string]any{"id": "bg1", "description": "npm test"},
	})
	h.HandleNotification("_loopal/bgTask.spawned", params)

	if gotKind != "bgTask.spawned" {
		t.Errorf("kind = %q, want bgTask.spawned", gotKind)
	}
	if gotSession != "sess-1" {
		t.Errorf("sessionID = %q, want sess-1", gotSession)
	}
	var d struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(gotData, &d); err != nil || d.ID != "bg1" {
		t.Errorf("data = %s, want id=bg1", gotData)
	}
}

func TestHandler_UnknownNotificationDropped(t *testing.T) {
	called := false
	h := NewHandler(EventCallbacks{
		OnLoopalExt: func(_, _ string, _ json.RawMessage) { called = true },
	}, testLogger())
	h.HandleNotification("some/unknown", mustMarshal(t, map[string]any{}))
	if called {
		t.Error("OnLoopalExt should not fire for non-_loopal notifications")
	}
}

func TestHandler_LoopalExtDropsSessionLevel(t *testing.T) {
	called := false
	h := NewHandler(EventCallbacks{
		OnLoopalExt: func(_, _ string, _ json.RawMessage) { called = true },
	}, testLogger())
	// permission_resolved carries no `data` field; forwarding it would push a
	// nil RawMessage into sendAcpViaRelay and panic on the nil-map flatten.
	// It must be dropped by the panel-kind allowlist.
	h.HandleNotification("_loopal/permission_resolved", mustMarshal(t, map[string]any{
		"sessionId": "s1", "toolCallId": "tc1",
	}))
	if called {
		t.Error("session-level _loopal extension must not reach OnLoopalExt")
	}
}

func TestLoopalPanelKinds_MatchesProtocol(t *testing.T) {
	// Pin the allowlist to Loopal's panel.rs emitters. A typo here (or a drift
	// from panel.rs) would silently drop a control-panel signal with no error
	// log, so guard the exact set.
	expected := []string{
		"bgTask.spawned", "bgTask.output", "bgTask.completed",
		"crons", "tasks", "mcp", "topology.spawn", "goal",
		"mode", "thinking", "model",
	}
	for _, k := range expected {
		if !loopalPanelKinds[k] {
			t.Errorf("loopalPanelKinds missing %q — Loopal panel signal would be silently dropped", k)
		}
	}
	if len(loopalPanelKinds) != len(expected) {
		t.Errorf("loopalPanelKinds has %d entries, want %d — an extra entry would forward a non-panel signal", len(loopalPanelKinds), len(expected))
	}
}

func TestHandler_PermissionModeRoutesToConfigChange(t *testing.T) {
	var gotSession string
	var gotUpdate ConfigUpdate
	loopalExtCalled := false
	h := NewHandler(EventCallbacks{
		OnConfigChange: func(sessionID string, update ConfigUpdate) {
			gotSession, gotUpdate = sessionID, update
		},
		OnLoopalExt: func(_, _ string, _ json.RawMessage) { loopalExtCalled = true },
	}, testLogger())

	h.HandleNotification("_loopal/permission_mode", mustMarshal(t, map[string]any{
		"sessionId": "sess-1",
		"data":      map[string]any{"permission_mode": "ask_dangerous"},
	}))

	if gotSession != "sess-1" {
		t.Errorf("sessionID = %q, want sess-1", gotSession)
	}
	if gotUpdate.PermissionMode != "ask_dangerous" {
		t.Errorf("PermissionMode = %q, want ask_dangerous", gotUpdate.PermissionMode)
	}
	if loopalExtCalled {
		t.Error("permission_mode must route to OnConfigChange, not OnLoopalExt")
	}
}

func TestHandler_PermissionModeEmptyIgnored(t *testing.T) {
	called := false
	h := NewHandler(EventCallbacks{
		OnConfigChange: func(_ string, _ ConfigUpdate) { called = true },
	}, testLogger())
	h.HandleNotification("_loopal/permission_mode", mustMarshal(t, map[string]any{
		"sessionId": "s1", "data": map[string]any{},
	}))
	if called {
		t.Error("empty permission_mode must not fire OnConfigChange")
	}
}

func TestHandler_ContentUpdate_UserMessage(t *testing.T) {
	var received []ContentChunk
	h := NewHandler(EventCallbacks{
		OnContentChunk: func(_ string, chunk ContentChunk) {
			received = append(received, chunk)
		},
	}, testLogger())

	params := mustMarshal(t, map[string]any{
		"sessionId": "sess-1",
		"update": map[string]any{
			"sessionUpdate": "user_message_chunk",
			"content":       map[string]any{"type": "text", "text": "User says hi"},
		},
	})
	h.HandleNotification("session/update", params)

	if len(received) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(received))
	}
	if received[0].Role != "user" {
		t.Errorf("Role = %q, want %q", received[0].Role, "user")
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
		"sessionId": "sess-1",
		"update": map[string]any{
			"sessionUpdate": "tool_call",
			"toolCallId":    "tc-1",
			"title":         "read_file",
			"status":        "running",
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

func TestHandler_ToolCallUpdate_PendingNormalized(t *testing.T) {
	var received []ToolCallUpdate
	h := NewHandler(EventCallbacks{
		OnToolCallUpdate: func(_ string, update ToolCallUpdate) {
			received = append(received, update)
		},
	}, testLogger())

	params := mustMarshal(t, map[string]any{
		"sessionId": "sess-1",
		"update": map[string]any{
			"sessionUpdate": "tool_call",
			"toolCallId":    "tc-2",
			"title":         "write_file",
			"status":        "pending",
		},
	})
	h.HandleNotification("session/update", params)

	if len(received) != 1 {
		t.Fatalf("expected 1 update, got %d", len(received))
	}
	if received[0].Status != "running" {
		t.Errorf("Status = %q, want %q (pending normalized to running)", received[0].Status, "running")
	}
}

func TestHandler_ToolResultUpdate(t *testing.T) {
	var updates []ToolCallUpdate
	var results []ToolCallResult
	h := NewHandler(EventCallbacks{
		OnToolCallUpdate: func(_ string, update ToolCallUpdate) {
			updates = append(updates, update)
		},
		OnToolCallResult: func(_ string, result ToolCallResult) {
			results = append(results, result)
		},
	}, testLogger())

	params := mustMarshal(t, map[string]any{
		"sessionId": "sess-1",
		"update": map[string]any{
			"sessionUpdate": "tool_call_update",
			"toolCallId":    "tc-2",
			"title":         "write_file",
			"status":        "completed",
		},
	})
	h.HandleNotification("session/update", params)

	if len(updates) != 1 {
		t.Fatalf("expected 1 tool call update, got %d", len(updates))
	}
	if updates[0].ToolCallID != "tc-2" {
		t.Errorf("ToolCallID = %q, want %q", updates[0].ToolCallID, "tc-2")
	}
	if updates[0].Status != "completed" {
		t.Errorf("Status = %q, want %q", updates[0].Status, "completed")
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 tool call result, got %d", len(results))
	}
	if results[0].ToolCallID != "tc-2" {
		t.Errorf("ToolCallID = %q, want %q", results[0].ToolCallID, "tc-2")
	}
	if !results[0].Success {
		t.Error("Success should be true for status=completed")
	}
}

func TestHandler_ToolResultUpdate_Failure(t *testing.T) {
	var results []ToolCallResult
	h := NewHandler(EventCallbacks{
		OnToolCallResult: func(_ string, result ToolCallResult) {
			results = append(results, result)
		},
	}, testLogger())

	params := mustMarshal(t, map[string]any{
		"sessionId": "sess-1",
		"update": map[string]any{
			"sessionUpdate": "tool_call_update",
			"toolCallId":    "tc-3",
			"title":         "exec",
			"status":        "failed",
		},
	})
	h.HandleNotification("session/update", params)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Success {
		t.Error("Success should be false for status=failed")
	}
	if results[0].ToolName != "exec" {
		t.Errorf("ToolName = %q, want %q", results[0].ToolName, "exec")
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
		"sessionId": "sess-1",
		"update": map[string]any{
			"sessionUpdate": "plan",
			"entries": []map[string]any{
				{"content": "Read config", "priority": "high", "status": "completed"},
				{"content": "Update code", "priority": "medium", "status": "in_progress"},
				{"content": "Run tests", "priority": "low", "status": "pending"},
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
