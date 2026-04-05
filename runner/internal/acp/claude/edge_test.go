package claude

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

func TestTransport_InvalidJSON(t *testing.T) {
	tr := NewTransport(acp.EventCallbacks{
		OnStateChange: func(string) {},
	}, discardLogger())
	ctx := context.Background()
	tr.Initialize(ctx, nil, strings.NewReader("bad json\n{\"type\":\"result\",\"subtype\":\"success\"}\n"), nil)
	tr.ReadLoop(ctx)
}

func TestTransport_ReadLoop_EmptyLine(t *testing.T) {
	tr := NewTransport(acp.EventCallbacks{
		OnStateChange: func(string) {},
	}, discardLogger())
	ctx := context.Background()
	tr.Initialize(ctx, nil, strings.NewReader("\n\n{\"type\":\"result\",\"subtype\":\"success\"}\n"), nil)
	tr.ReadLoop(ctx)
}

func TestTransport_ReadLoop_ScannerError(t *testing.T) {
	tr := NewTransport(acp.EventCallbacks{}, discardLogger())
	ctx := context.Background()
	tr.Initialize(ctx, nil, &errorReader{err: io.ErrUnexpectedEOF}, nil)
	tr.ReadLoop(ctx) // must not panic
}

func TestTransport_ReadLoop_CtxDoneAfterScan(t *testing.T) {
	pr, pw := io.Pipe()
	tr := NewTransport(acp.EventCallbacks{}, discardLogger())
	ctx, cancel := context.WithCancel(context.Background())
	tr.Initialize(ctx, nil, pr, nil)
	done := make(chan struct{})
	go func() { tr.ReadLoop(ctx); close(done) }()
	writeLine(pw, map[string]any{"type": "result", "subtype": "success"})
	time.Sleep(50 * time.Millisecond)
	cancel()
	pw.Close()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ReadLoop did not exit")
	}
}

func TestTransport_ReadLoop_ContextCancel(t *testing.T) {
	pr, pw := io.Pipe()
	tr := NewTransport(acp.EventCallbacks{}, discardLogger())
	ctx, cancel := context.WithCancel(context.Background())
	tr.Initialize(ctx, nil, pr, nil)
	done := make(chan struct{})
	go func() { tr.ReadLoop(ctx); close(done) }()
	cancel()
	pw.Close()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ReadLoop did not exit")
	}
}

func TestTransport_HandleMessage_UnknownType(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "totally_unknown"})
	f.Drain()
}

func TestTransport_StreamEvent_ParseError(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "stream_event", "event": "bad"})
	f.Drain()
}

func TestTransport_StreamEvent_MessageStartDeltaStop(t *testing.T) {
	f := newFixture()
	defer f.Close()
	for _, et := range []string{"message_start", "message_delta", "message_stop"} {
		writeLine(f.PW, map[string]any{"type": "stream_event", "event": map[string]any{"type": et}})
	}
	f.Drain()
}

func TestTransport_StreamEvent_UnknownEventType(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "stream_event", "event": map[string]any{"type": "future"}})
	f.Drain()
}

func TestTransport_ContentBlockStart_ParseError(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "stream_event", "event": map[string]any{
		"type": "content_block_start", "index": 0, "content_block": "bad",
	}})
	f.Drain()
}

func TestTransport_ContentBlockDelta_ParseError(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "stream_event", "event": map[string]any{
		"type": "content_block_delta", "index": 0, "delta": "bad",
	}})
	f.Drain()
}

func TestTransport_HandleUser_MessageParseError(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "user", "message": "bad"})
	f.Drain()
}

func TestTransport_AssistantNonStreaming(t *testing.T) {
	// When no stream_event precedes the assistant message (non-streaming response),
	// the content should be extracted and delivered as a chunk.
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{"type": "assistant", "message": map[string]any{
		"role": "assistant", "content": []map[string]any{{"type": "text", "text": "Hello from assistant"}},
	}})
	f.Drain()
	_, chunks, _, _ := f.Snap()
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk (non-streaming assistant), got %d", len(chunks))
	}
	if len(chunks) > 0 && chunks[0].Text != "Hello from assistant" {
		t.Errorf("chunk text = %q", chunks[0].Text)
	}
}

func TestTransport_Close_NoOp(t *testing.T) {
	tr := NewTransport(acp.EventCallbacks{}, discardLogger())
	tr.Close() // no-op, should not panic
}

// --- control_request tests ---

func TestTransport_ControlRequest_CanUseTool(t *testing.T) {
	f := newFixtureWithStdin()
	defer f.Close()

	// Set session ID for the transport
	f.Transport.sessionMu.Lock()
	f.Transport.sessionID = "sess-1"
	f.Transport.sessionMu.Unlock()

	input, _ := json.Marshal(map[string]any{"file_path": "/tmp/test.txt"})
	writeLine(f.PW, map[string]any{
		"type":       "control_request",
		"request_id": "cr-1",
		"request": map[string]any{
			"subtype":   "can_use_tool",
			"tool_name": "Write",
			"input":     json.RawMessage(input),
		},
	})
	f.Drain()

	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.PermissionRequests) != 1 {
		t.Fatalf("expected 1 permission request, got %d", len(f.PermissionRequests))
	}
	req := f.PermissionRequests[0]
	if req.RequestID != "cr-1" {
		t.Errorf("request_id = %q", req.RequestID)
	}
	if req.ToolName != "Write" {
		t.Errorf("tool_name = %q", req.ToolName)
	}
	if req.SessionID != "sess-1" {
		t.Errorf("session_id = %q", req.SessionID)
	}
	if len(f.StateChanges) < 1 || f.StateChanges[0] != acp.StateWaitingPermission {
		t.Errorf("states = %v, expected waiting_permission first", f.StateChanges)
	}
}

func TestTransport_ControlRequest_UnknownSubtype(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{
		"type":       "control_request",
		"request_id": "cr-2",
		"request":    map[string]any{"subtype": "unknown_subtype"},
	})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.PermissionRequests) != 0 {
		t.Errorf("expected 0 permission requests, got %d", len(f.PermissionRequests))
	}
}

func TestTransport_ControlRequest_MissingRequestID(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{
		"type":    "control_request",
		"request": map[string]any{"subtype": "can_use_tool", "tool_name": "Read"},
	})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.PermissionRequests) != 0 {
		t.Errorf("expected 0 permission requests (missing request_id), got %d", len(f.PermissionRequests))
	}
}

func TestTransport_ControlRequest_ParseError(t *testing.T) {
	f := newFixture()
	defer f.Close()
	writeLine(f.PW, map[string]any{
		"type":       "control_request",
		"request_id": "cr-3",
		"request":    "not-an-object",
	})
	f.Drain()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.PermissionRequests) != 0 {
		t.Errorf("expected 0 permission requests (parse error), got %d", len(f.PermissionRequests))
	}
}
