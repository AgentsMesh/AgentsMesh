package acp

import (
	"encoding/json"
	"testing"
)

// --- Permission request (now via HandlePermissionRequest) ---

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
		"sessionId": "sess-1",
		"toolCall": map[string]any{
			"toolCallId": "tc-1",
			"title":      "exec_command",
		},
		"options": []map[string]any{
			{"optionId": "allow-once", "name": "Allow once", "kind": "allow_once"},
		},
	})
	h.HandlePermissionRequest(5, params)

	if len(received) != 1 {
		t.Fatalf("expected 1 permission request, got %d", len(received))
	}
	if received[0].SessionID != "sess-1" {
		t.Errorf("SessionID = %q, want %q", received[0].SessionID, "sess-1")
	}
	if received[0].RequestID != "5" {
		t.Errorf("RequestID = %q, want %q", received[0].RequestID, "5")
	}
	if received[0].ToolName != "exec_command" {
		t.Errorf("ToolName = %q, want %q", received[0].ToolName, "exec_command")
	}
	if received[0].Description != "Tool: tc-1" {
		t.Errorf("Description = %q, want %q", received[0].Description, "Tool: tc-1")
	}

	if len(stateChanges) != 1 || stateChanges[0] != StateWaitingPermission {
		t.Errorf("stateChanges = %v, want [%q]", stateChanges, StateWaitingPermission)
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
	h.HandlePermissionRequest(99, json.RawMessage(`{broken`))
}

func TestHandler_InvalidJSON_ContentData_NoCrash(t *testing.T) {
	h := NewHandler(EventCallbacks{
		OnContentChunk: func(_ string, _ ContentChunk) {
			t.Error("OnContentChunk should not be called with invalid data")
		},
	}, testLogger())

	params := mustMarshal(t, map[string]any{
		"sessionId": "sess-1",
		"update":    "not a json object",
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
		"sessionId": "sess-1",
		"update":    12345,
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
		"sessionId": "sess-1",
		"update":    []int{1, 2, 3},
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
		"sessionId": "sess-1",
		"update":    "not_an_object",
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
		"sessionId": "sess-1",
		"update":    true,
	})
	h.HandleNotification("session/update", params)
}

// --- SelectOptionID tests ---

func TestHandler_SelectOptionID_Approve(t *testing.T) {
	h := NewHandler(EventCallbacks{
		OnStateChange:       func(string) {},
		OnPermissionRequest: func(PermissionRequest) {},
	}, testLogger())

	params := mustMarshal(t, map[string]any{
		"sessionId": "sess-1",
		"toolCall":  map[string]any{"toolCallId": "tc-1", "title": "exec"},
		"options": []map[string]any{
			{"optionId": "opt-allow", "name": "Allow once", "kind": "allow_once"},
			{"optionId": "opt-reject", "name": "Reject once", "kind": "reject_once"},
		},
	})
	h.HandlePermissionRequest(10, params)

	got := h.SelectOptionID("10", true)
	if got != "opt-allow" {
		t.Errorf("SelectOptionID(approved=true) = %q, want %q", got, "opt-allow")
	}
}

func TestHandler_SelectOptionID_Deny(t *testing.T) {
	h := NewHandler(EventCallbacks{
		OnStateChange:       func(string) {},
		OnPermissionRequest: func(PermissionRequest) {},
	}, testLogger())

	params := mustMarshal(t, map[string]any{
		"sessionId": "sess-1",
		"toolCall":  map[string]any{"toolCallId": "tc-2", "title": "write_file"},
		"options": []map[string]any{
			{"optionId": "opt-allow", "name": "Allow once", "kind": "allow_once"},
			{"optionId": "opt-reject", "name": "Reject once", "kind": "reject_once"},
		},
	})
	h.HandlePermissionRequest(11, params)

	got := h.SelectOptionID("11", false)
	if got != "opt-reject" {
		t.Errorf("SelectOptionID(approved=false) = %q, want %q", got, "opt-reject")
	}
}

func TestHandler_SelectOptionID_Fallback(t *testing.T) {
	h := NewHandler(EventCallbacks{
		OnStateChange:       func(string) {},
		OnPermissionRequest: func(PermissionRequest) {},
	}, testLogger())

	// Options with no matching kind for deny
	params := mustMarshal(t, map[string]any{
		"sessionId": "sess-1",
		"toolCall":  map[string]any{"toolCallId": "tc-3", "title": "read"},
		"options": []map[string]any{
			{"optionId": "opt-first", "name": "Allow once", "kind": "allow_once"},
		},
	})
	h.HandlePermissionRequest(12, params)

	// Deny with no reject_once option → falls back to first option
	got := h.SelectOptionID("12", false)
	if got != "opt-first" {
		t.Errorf("SelectOptionID(fallback) = %q, want %q", got, "opt-first")
	}
}

func TestHandler_SelectOptionID_Empty(t *testing.T) {
	h := NewHandler(EventCallbacks{}, testLogger())

	// No stored options for this requestID → returns ""
	got := h.SelectOptionID("999", true)
	if got != "" {
		t.Errorf("SelectOptionID(empty) = %q, want %q", got, "")
	}
}
