package acp

import (
	"encoding/json"
	"testing"
)

// --- Permission request ---

func TestHandler_PermissionRequest(t *testing.T) {
	var received []PermissionRequest
	var stateChanges []string
	h := NewHandler(EventCallbacks{
		OnPermissionRequest: func(req PermissionRequest) {
			received = append(received, req)
		},
		OnStateChange: func(newState string) {
			stateChanges = append(stateChanges, newState)
		},
	}, testLogger())

	params := mustMarshal(t, map[string]any{
		"session_id":     "sess-1",
		"request_id":     "perm-1",
		"tool_name":      "exec_command",
		"arguments_json": `{"cmd":"rm -rf /"}`,
		"description":    "Delete everything",
	})
	h.HandleNotification("permission/request", params)

	if len(received) != 1 {
		t.Fatalf("expected 1 permission request, got %d", len(received))
	}
	if received[0].SessionID != "sess-1" {
		t.Errorf("SessionID = %q, want %q", received[0].SessionID, "sess-1")
	}
	if received[0].RequestID != "perm-1" {
		t.Errorf("RequestID = %q, want %q", received[0].RequestID, "perm-1")
	}
	if received[0].ToolName != "exec_command" {
		t.Errorf("ToolName = %q, want %q", received[0].ToolName, "exec_command")
	}
	if received[0].Description != "Delete everything" {
		t.Errorf("Description = %q, want %q", received[0].Description, "Delete everything")
	}

	if len(stateChanges) != 1 || stateChanges[0] != StateWaitingPermission {
		t.Errorf("stateChanges = %v, want [%q]", stateChanges, StateWaitingPermission)
	}
}

// --- Session complete ---

func TestHandler_SessionComplete(t *testing.T) {
	var stateChanges []string
	h := NewHandler(EventCallbacks{
		OnStateChange: func(newState string) {
			stateChanges = append(stateChanges, newState)
		},
	}, testLogger())

	h.HandleNotification("session/complete", json.RawMessage(`{}`))

	if len(stateChanges) != 1 || stateChanges[0] != StateIdle {
		t.Errorf("stateChanges = %v, want [%q]", stateChanges, StateIdle)
	}
}

// --- Unknown method does not crash ---

func TestHandler_UnknownMethod_NoCrash(t *testing.T) {
	h := NewHandler(EventCallbacks{}, testLogger())
	h.HandleNotification("unknown/method", json.RawMessage(`{"foo":"bar"}`))
}

// --- Invalid JSON params do not crash ---

func TestHandler_InvalidJSON_SessionUpdate_NoCrash(t *testing.T) {
	h := NewHandler(EventCallbacks{
		OnContentChunk: func(_ string, _ ContentChunk) {
			t.Error("OnContentChunk should not be called with invalid JSON")
		},
	}, testLogger())
	h.HandleNotification("session/update", json.RawMessage(`not valid json`))
}

func TestHandler_InvalidJSON_PermissionRequest_NoCrash(t *testing.T) {
	h := NewHandler(EventCallbacks{
		OnPermissionRequest: func(_ PermissionRequest) {
			t.Error("OnPermissionRequest should not be called with invalid JSON")
		},
	}, testLogger())
	h.HandleNotification("permission/request", json.RawMessage(`{broken`))
}

func TestHandler_InvalidJSON_ContentData_NoCrash(t *testing.T) {
	h := NewHandler(EventCallbacks{
		OnContentChunk: func(_ string, _ ContentChunk) {
			t.Error("OnContentChunk should not be called with invalid data")
		},
	}, testLogger())

	params := mustMarshal(t, map[string]any{
		"session_id": "sess-1",
		"type":       "content",
		"data":       "not a json object",
	})
	h.HandleNotification("session/update", params)
}

func TestHandler_InvalidJSON_ToolCallData_NoCrash(t *testing.T) {
	h := NewHandler(EventCallbacks{
		OnToolCallUpdate: func(_ string, _ ToolCallUpdate) {
			t.Error("OnToolCallUpdate should not be called with invalid data")
		},
	}, testLogger())

	params := mustMarshal(t, map[string]any{
		"session_id": "sess-1",
		"type":       "tool_call",
		"data":       12345,
	})
	h.HandleNotification("session/update", params)
}

func TestHandler_InvalidJSON_PlanData_NoCrash(t *testing.T) {
	h := NewHandler(EventCallbacks{
		OnPlanUpdate: func(_ string, _ PlanUpdate) {
			t.Error("OnPlanUpdate should not be called with invalid data")
		},
	}, testLogger())

	params := mustMarshal(t, map[string]any{
		"session_id": "sess-1",
		"type":       "plan",
		"data":       []int{1, 2, 3},
	})
	h.HandleNotification("session/update", params)
}

func TestHandler_InvalidJSON_ToolResultData_NoCrash(t *testing.T) {
	h := NewHandler(EventCallbacks{
		OnToolCallResult: func(_ string, _ ToolCallResult) {
			t.Error("OnToolCallResult should not be called with invalid data")
		},
	}, testLogger())

	params := mustMarshal(t, map[string]any{
		"session_id": "sess-1",
		"type":       "tool_result",
		"data":       "not_an_object",
	})
	h.HandleNotification("session/update", params)
}

func TestHandler_InvalidJSON_ThinkingData_NoCrash(t *testing.T) {
	h := NewHandler(EventCallbacks{
		OnThinkingUpdate: func(_ string, _ ThinkingUpdate) {
			t.Error("OnThinkingUpdate should not be called with invalid data")
		},
	}, testLogger())

	params := mustMarshal(t, map[string]any{
		"session_id": "sess-1",
		"type":       "thinking",
		"data":       true,
	})
	h.HandleNotification("session/update", params)
}
