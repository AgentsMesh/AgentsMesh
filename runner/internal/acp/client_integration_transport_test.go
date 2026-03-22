package acp

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestACPTransport_HandleResponse_UnparseableID(t *testing.T) {
	transport := NewACPTransport(EventCallbacks{}, slog.Default())
	transport.Initialize(context.Background(), os.Stdout, os.Stdin, nil)

	raw := json.RawMessage(`"not_a_number"`)
	msg := &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      &raw,
		Result:  json.RawMessage(`{}`),
	}

	// Should not panic -- logs a warning and returns.
	transport.tracker.HandleResponse(msg)
}

func TestACPTransport_HandleResponse_UnmatchedID(t *testing.T) {
	transport := NewACPTransport(EventCallbacks{}, slog.Default())
	transport.Initialize(context.Background(), os.Stdout, os.Stdin, nil)

	raw := json.RawMessage(`999`)
	msg := &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      &raw,
		Result:  json.RawMessage(`{}`),
	}

	// Should not panic -- logs a warning about unmatched response.
	transport.tracker.HandleResponse(msg)
}

func TestACPTransport_WaitResponse_Timeout(t *testing.T) {
	transport := NewACPTransport(EventCallbacks{}, slog.Default())
	transport.Initialize(context.Background(), os.Stdout, os.Stdin, nil)

	// Create a PendingRequest with a channel that will never receive
	pr := &PendingRequest{
		ID: NextRequestID(),
		ch: make(chan *JSONRPCResponse, 1),
	}

	_, err := transport.tracker.WaitResponse(pr, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestACPTransport_WaitResponse_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	transport := NewACPTransport(EventCallbacks{}, slog.Default())
	transport.Initialize(ctx, os.Stdout, os.Stdin, nil)

	pr := &PendingRequest{
		ID: NextRequestID(),
		ch: make(chan *JSONRPCResponse, 1),
	}

	cancel()

	_, err := transport.tracker.WaitResponse(pr, 30*time.Second)
	if err == nil {
		t.Fatal("expected context cancel error")
	}
}

func TestACPTransport_DispatchMessage_Request(t *testing.T) {
	pr, pw, _ := os.Pipe()
	defer pr.Close()

	transport := NewACPTransport(EventCallbacks{}, slog.Default())
	transport.Initialize(context.Background(), pw, os.Stdin, nil)

	raw := json.RawMessage(`42`)
	msg := &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      &raw,
		Method:  "some/unsupported_method",
	}

	transport.dispatchMessage(msg)
	pw.Close()

	reader := NewReader(pr, slog.Default())
	resp, err := reader.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	if !resp.IsResponse() {
		t.Fatal("expected a response message")
	}
	if resp.Error == nil {
		t.Fatal("expected an error response")
	}
	if resp.Error.Code != ErrCodeMethodNotFound {
		t.Errorf("expected method-not-found code, got %d", resp.Error.Code)
	}
}
