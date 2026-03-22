package acp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWriter_WriteRequest_Format(t *testing.T) {
	var buf strings.Builder
	w := NewWriter(&buf)

	id, err := w.WriteRequest("initialize", map[string]any{"key": "val"})
	if err != nil {
		t.Fatalf("WriteRequest() error = %v", err)
	}
	if id <= 0 {
		t.Errorf("WriteRequest() returned id = %d, want > 0", id)
	}

	output := buf.String()
	if !strings.HasSuffix(output, "\n") {
		t.Error("output should end with newline")
	}

	var req JSONRPCRequest
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &req); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if req.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want %q", req.JSONRPC, "2.0")
	}
	if req.Method != "initialize" {
		t.Errorf("Method = %q, want %q", req.Method, "initialize")
	}
	if req.ID != id {
		t.Errorf("ID = %d, want %d", req.ID, id)
	}
	if req.Params == nil {
		t.Error("Params should not be nil")
	}
}

func TestWriter_WriteRequest_NilParams(t *testing.T) {
	var buf strings.Builder
	w := NewWriter(&buf)

	_, err := w.WriteRequest("ping", nil)
	if err != nil {
		t.Fatalf("WriteRequest() error = %v", err)
	}

	var req JSONRPCRequest
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &req); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if req.Params != nil {
		t.Error("Params should be nil when no params provided")
	}
}

func TestWriter_WriteRequest_MonotonicIDs(t *testing.T) {
	var buf strings.Builder
	w := NewWriter(&buf)

	id1, _ := w.WriteRequest("method1", nil)
	id2, _ := w.WriteRequest("method2", nil)
	id3, _ := w.WriteRequest("method3", nil)

	if id2 <= id1 {
		t.Errorf("id2 (%d) should be > id1 (%d)", id2, id1)
	}
	if id3 <= id2 {
		t.Errorf("id3 (%d) should be > id2 (%d)", id3, id2)
	}
}

func TestWriter_WriteNotification_Format(t *testing.T) {
	var buf strings.Builder
	w := NewWriter(&buf)

	err := w.WriteNotification("session/cancel", map[string]any{"session_id": "s1"})
	if err != nil {
		t.Fatalf("WriteNotification() error = %v", err)
	}

	output := buf.String()
	if !strings.HasSuffix(output, "\n") {
		t.Error("output should end with newline")
	}

	var notif JSONRPCNotification
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &notif); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if notif.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want %q", notif.JSONRPC, "2.0")
	}
	if notif.Method != "session/cancel" {
		t.Errorf("Method = %q, want %q", notif.Method, "session/cancel")
	}
}

func TestWriter_WriteNotification_NilParams(t *testing.T) {
	var buf strings.Builder
	w := NewWriter(&buf)

	if err := w.WriteNotification("heartbeat", nil); err != nil {
		t.Fatalf("WriteNotification() error = %v", err)
	}

	var notif JSONRPCNotification
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &notif); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if notif.Params != nil {
		t.Error("Params should be nil when no params provided")
	}
}

func TestWriter_WriteNotification_NoIDField(t *testing.T) {
	var buf strings.Builder
	w := NewWriter(&buf)

	_ = w.WriteNotification("test", nil)

	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &raw); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if _, exists := raw["id"]; exists {
		t.Error("notification should not contain an 'id' field")
	}
}
