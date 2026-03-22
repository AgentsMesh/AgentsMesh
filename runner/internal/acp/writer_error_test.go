package acp

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestWriter_WriteResponse_WithResult(t *testing.T) {
	var buf strings.Builder
	w := NewWriter(&buf)

	var id int64 = 42
	result := map[string]any{"status": "ok"}
	if err := w.WriteResponse(id, result, nil); err != nil {
		t.Fatalf("WriteResponse() error = %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var resp JSONRPCResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want %q", resp.JSONRPC, "2.0")
	}
	if resp.ID == nil || *resp.ID != id {
		t.Errorf("ID = %v, want %d", resp.ID, id)
	}
	if resp.Result == nil {
		t.Error("Result should not be nil")
	}
	if resp.Error != nil {
		t.Error("Error should be nil for successful response")
	}
}

func TestWriter_WriteResponse_WithError(t *testing.T) {
	var buf strings.Builder
	w := NewWriter(&buf)

	var id int64 = 99
	rpcErr := &JSONRPCError{
		Code:    ErrCodeMethodNotFound,
		Message: "method not found",
	}
	if err := w.WriteResponse(id, nil, rpcErr); err != nil {
		t.Fatalf("WriteResponse() error = %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var resp JSONRPCResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if resp.Error.Code != ErrCodeMethodNotFound {
		t.Errorf("Error.Code = %d, want %d", resp.Error.Code, ErrCodeMethodNotFound)
	}
	if resp.Error.Message != "method not found" {
		t.Errorf("Error.Message = %q, want %q", resp.Error.Message, "method not found")
	}
	if resp.Result != nil {
		t.Error("Result should be nil when error is present")
	}
}

func TestWriter_WriteResponse_ErrorTakesPrecedenceOverResult(t *testing.T) {
	var buf strings.Builder
	w := NewWriter(&buf)

	var id int64 = 7
	rpcErr := &JSONRPCError{Code: ErrCodeInternal, Message: "internal error"}
	if err := w.WriteResponse(id, map[string]any{"data": "ignored"}, rpcErr); err != nil {
		t.Fatalf("WriteResponse() error = %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var resp JSONRPCResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if resp.Result != nil {
		t.Error("Result should be nil when error is present")
	}
	if resp.Error == nil {
		t.Fatal("Error should not be nil")
	}
}

func TestWriter_WriteResponse_NewlineTerminated(t *testing.T) {
	var buf strings.Builder
	w := NewWriter(&buf)

	var id int64 = 1
	_ = w.WriteResponse(id, "ok", nil)

	if !strings.HasSuffix(buf.String(), "\n") {
		t.Error("output should end with newline")
	}
}

// --- Marshal error tests ---

// errWriter is an io.Writer that always returns an error.
type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, fmt.Errorf("mock write error")
}

func TestWriter_WriteRequest_MarshalError(t *testing.T) {
	var buf strings.Builder
	w := NewWriter(&buf)

	_, err := w.WriteRequest("test", make(chan int))
	if err == nil {
		t.Error("expected marshal error for channel params")
	}
}

func TestWriter_WriteNotification_MarshalError(t *testing.T) {
	var buf strings.Builder
	w := NewWriter(&buf)

	err := w.WriteNotification("test", make(chan int))
	if err == nil {
		t.Error("expected marshal error for channel params")
	}
}

func TestWriter_WriteResponse_MarshalError(t *testing.T) {
	var buf strings.Builder
	w := NewWriter(&buf)

	var id int64 = 1
	err := w.WriteResponse(id, make(chan int), nil)
	if err == nil {
		t.Error("expected marshal error for channel result")
	}
}

func TestWriter_WriteJSON_WriteError(t *testing.T) {
	w := NewWriter(errWriter{})

	_, err := w.WriteRequest("test", nil)
	if err == nil {
		t.Error("expected write error from underlying writer")
	}
}

func TestWriter_WriteNotification_WriteError(t *testing.T) {
	w := NewWriter(errWriter{})

	err := w.WriteNotification("test", nil)
	if err == nil {
		t.Error("expected write error from underlying writer")
	}
}

func TestWriter_WriteResponse_WriteError(t *testing.T) {
	w := NewWriter(errWriter{})

	var id int64 = 1
	err := w.WriteResponse(id, "ok", nil)
	if err == nil {
		t.Error("expected write error from underlying writer")
	}
}
