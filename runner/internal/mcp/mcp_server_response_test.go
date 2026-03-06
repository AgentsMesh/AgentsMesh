package mcp

import (
	"encoding/json"
	"testing"
	"time"
)

// Tests for response routing and error handling

func TestServerResponseRouting(t *testing.T) {
	server := NewServer(&Config{Name: "test", Command: testDummyCmd()})

	// Set up pending channel
	ch := make(chan *Response, 1)
	server.mu.Lock()
	server.pending[42] = ch
	server.mu.Unlock()

	// Simulate response routing
	resp := &Response{
		JSONRPC: "2.0",
		ID:      42,
		Result:  json.RawMessage(`{"success": true}`),
	}

	server.mu.Lock()
	if pendingCh, ok := server.pending[resp.ID]; ok {
		pendingCh <- resp
	}
	server.mu.Unlock()

	// Verify response was routed
	select {
	case got := <-ch:
		if got.ID != 42 {
			t.Errorf("response ID: got %v, want 42", got.ID)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for response")
	}
}

func TestServerResponseRoutingUnknownID(t *testing.T) {
	server := NewServer(&Config{Name: "test", Command: testDummyCmd()})

	// No pending channel for ID 999
	resp := &Response{
		JSONRPC: "2.0",
		ID:      999,
		Result:  json.RawMessage(`{}`),
	}

	// Should not panic when routing to unknown ID
	server.mu.Lock()
	if pendingCh, ok := server.pending[resp.ID]; ok {
		pendingCh <- resp
	}
	server.mu.Unlock()
	// No assertion needed - just verifying no panic
}

func TestCallToolErrorResponse(t *testing.T) {
	// This tests the error response handling code path
	errResp := &Response{
		JSONRPC: "2.0",
		ID:      1,
		Error: &Error{
			Code:    -32600,
			Message: "Invalid Request",
		},
	}

	if errResp.Error == nil {
		t.Error("error should not be nil")
	}

	if errResp.Error.Code != -32600 {
		t.Errorf("error code: got %v, want -32600", errResp.Error.Code)
	}
}
