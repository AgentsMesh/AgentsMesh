package claude

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

func TestTransport_SystemInit(t *testing.T) {
	pr, pw := io.Pipe()
	defer pr.Close()
	tr := NewTransport(acp.EventCallbacks{}, slog.Default())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tr.Initialize(ctx, nil, pr, nil)
	go tr.ReadLoop(ctx)

	// system/init sets session_id but no longer closes initCh
	writeLine(pw, map[string]any{
		"type": "system", "subtype": "init", "session_id": "sess-123",
	})
	time.Sleep(100 * time.Millisecond)

	tr.sessionMu.RLock()
	sid := tr.sessionID
	tr.sessionMu.RUnlock()
	if sid != "sess-123" {
		t.Errorf("session_id = %q, want sess-123", sid)
	}

	// initCh should still be open (not closed by system/init)
	select {
	case <-tr.initCh:
		t.Error("initCh should not be closed by system/init alone")
	default:
		// expected
	}
}

func TestTransport_SendPrompt(t *testing.T) {
	stdinPR, stdinPW := io.Pipe()
	defer stdinPR.Close()
	defer stdinPW.Close()
	stdoutPR, stdoutPW := io.Pipe()
	defer stdoutPR.Close()
	defer stdoutPW.Close()

	tr := NewTransport(acp.EventCallbacks{}, slog.Default())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tr.Initialize(ctx, stdinPW, stdoutPR, nil)

	// Set session ID so SendPrompt includes it
	tr.sessionMu.Lock()
	tr.sessionID = "test-session"
	tr.sessionMu.Unlock()

	var received string
	done := make(chan struct{})
	go func() { buf := make([]byte, 4096); n, _ := stdinPR.Read(buf); received = string(buf[:n]); close(done) }()

	if err := tr.SendPrompt("test-session", "hello world"); err != nil {
		t.Fatalf("SendPrompt: %v", err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
	var input userInput
	if err := json.Unmarshal([]byte(strings.TrimSpace(received)), &input); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if input.Message.Content != "hello world" {
		t.Errorf("content = %q", input.Message.Content)
	}
	if input.SessionID != "test-session" {
		t.Errorf("session_id = %q, want test-session", input.SessionID)
	}
}

func TestTransport_SendPrompt_WriteError(t *testing.T) {
	tr := NewTransport(acp.EventCallbacks{}, slog.Default())
	_, pw := io.Pipe()
	pw.Close()
	tr.stdin = pw
	err := tr.SendPrompt("s", "hello")
	if err == nil || !strings.Contains(err.Error(), "write user input") {
		t.Errorf("expected write error, got %v", err)
	}
}

func TestTransport_RespondToPermission_Allow(t *testing.T) {
	stdinPR, stdinPW := io.Pipe()
	defer stdinPR.Close()
	defer stdinPW.Close()

	tr := NewTransport(acp.EventCallbacks{}, slog.Default())
	tr.stdin = stdinPW

	done := make(chan string)
	go func() { buf := make([]byte, 4096); n, _ := stdinPR.Read(buf); done <- string(buf[:n]) }()

	if err := tr.RespondToPermission("req-1", true); err != nil {
		t.Fatalf("RespondToPermission: %v", err)
	}

	select {
	case received := <-done:
		var msg controlResponseMessage
		if err := json.Unmarshal([]byte(strings.TrimSpace(received)), &msg); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if msg.Type != "control_response" {
			t.Errorf("type = %q", msg.Type)
		}
		if msg.Response.RequestID != "req-1" {
			t.Errorf("request_id = %q", msg.Response.RequestID)
		}
		if msg.Response.Response.Behavior != "allow" {
			t.Errorf("behavior = %q", msg.Response.Response.Behavior)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestTransport_RespondToPermission_Deny(t *testing.T) {
	stdinPR, stdinPW := io.Pipe()
	defer stdinPR.Close()
	defer stdinPW.Close()

	tr := NewTransport(acp.EventCallbacks{}, slog.Default())
	tr.stdin = stdinPW

	done := make(chan string)
	go func() { buf := make([]byte, 4096); n, _ := stdinPR.Read(buf); done <- string(buf[:n]) }()

	if err := tr.RespondToPermission("req-2", false); err != nil {
		t.Fatalf("RespondToPermission: %v", err)
	}

	select {
	case received := <-done:
		var msg controlResponseMessage
		if err := json.Unmarshal([]byte(strings.TrimSpace(received)), &msg); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if msg.Response.Response.Behavior != "deny" {
			t.Errorf("behavior = %q, want deny", msg.Response.Response.Behavior)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestTransport_RespondToPermission_WriteError(t *testing.T) {
	tr := NewTransport(acp.EventCallbacks{}, slog.Default())
	_, pw := io.Pipe()
	pw.Close()
	tr.stdin = pw
	err := tr.RespondToPermission("req-3", true)
	if err == nil || !strings.Contains(err.Error(), "write control response") {
		t.Errorf("expected write error, got %v", err)
	}
}

func TestTransport_NewSession_ReturnsCachedID(t *testing.T) {
	tr := NewTransport(acp.EventCallbacks{}, slog.Default())
	tr.sessionMu.Lock()
	tr.sessionID = "cached"
	tr.sessionMu.Unlock()
	sid, err := tr.NewSession("", nil)
	if err != nil || sid != "cached" {
		t.Errorf("sid=%q err=%v", sid, err)
	}
}

func TestTransport_CancelSession(t *testing.T) {
	tr := NewTransport(acp.EventCallbacks{}, slog.Default())
	if err := tr.CancelSession("x"); err != nil {
		t.Errorf("got %v", err)
	}
}

func TestTransport_Handshake(t *testing.T) {
	stdoutPR, stdoutPW := io.Pipe()
	defer stdoutPR.Close()
	stdinPR, stdinPW := io.Pipe()
	defer stdinPR.Close()
	defer stdinPW.Close()

	tr := NewTransport(acp.EventCallbacks{}, slog.Default())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tr.Initialize(ctx, stdinPW, stdoutPR, nil)
	go tr.ReadLoop(ctx)

	// Drain the control_request initialize written to stdin
	go func() {
		buf := make([]byte, 4096)
		n, _ := stdinPR.Read(buf)
		var msg controlInitMessage
		json.Unmarshal(buf[:n-1], &msg) // strip trailing \n
		if msg.Type != "control_request" || msg.Request.Subtype != "initialize" {
			t.Errorf("unexpected init message: %s", string(buf[:n]))
		}
	}()

	// Respond with control_response to complete handshake
	writeLine(stdoutPW, map[string]any{
		"type": "control_response",
		"response": map[string]any{
			"subtype":    "success",
			"request_id": "init_1",
			"response":   map[string]any{},
		},
	})

	sid, err := tr.Handshake(ctx)
	if err != nil {
		t.Fatalf("Handshake: %v", err)
	}
	// session_id is discovered asynchronously via system/init, Handshake returns ""
	if sid != "" {
		t.Errorf("sid = %q, want empty", sid)
	}
}

func TestTransport_Handshake_ControlRequestInitialize(t *testing.T) {
	stdoutPR, stdoutPW := io.Pipe()
	defer stdoutPR.Close()
	stdinPR, stdinPW := io.Pipe()
	defer stdinPR.Close()
	defer stdinPW.Close()

	tr := NewTransport(acp.EventCallbacks{}, slog.Default())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tr.Initialize(ctx, stdinPW, stdoutPR, nil)
	go tr.ReadLoop(ctx)

	// Read and verify the control_request initialize from stdin
	stdinDone := make(chan controlInitMessage)
	go func() {
		buf := make([]byte, 4096)
		n, _ := stdinPR.Read(buf)
		var msg controlInitMessage
		json.Unmarshal(buf[:n-1], &msg)
		stdinDone <- msg
	}()

	// Simulate control_response
	writeLine(stdoutPW, map[string]any{
		"type": "control_response",
		"response": map[string]any{
			"subtype":    "success",
			"request_id": "init_1",
		},
	})

	sid, err := tr.Handshake(ctx)
	if err != nil {
		t.Fatalf("Handshake: %v", err)
	}
	if sid != "" {
		t.Errorf("sid = %q, want empty", sid)
	}

	select {
	case msg := <-stdinDone:
		if msg.Type != "control_request" {
			t.Errorf("type = %q, want control_request", msg.Type)
		}
		if msg.RequestID != "init_1" {
			t.Errorf("request_id = %q, want init_1", msg.RequestID)
		}
		if msg.Request.Subtype != "initialize" {
			t.Errorf("subtype = %q, want initialize", msg.Request.Subtype)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for control_request on stdin")
	}
}

func TestTransport_ControlResponse_ClosesInitCh(t *testing.T) {
	pr, pw := io.Pipe()
	defer pr.Close()
	tr := NewTransport(acp.EventCallbacks{}, slog.Default())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tr.Initialize(ctx, nil, pr, nil)
	go tr.ReadLoop(ctx)

	writeLine(pw, map[string]any{
		"type": "control_response",
		"response": map[string]any{
			"subtype":    "success",
			"request_id": "init_1",
		},
	})
	time.Sleep(100 * time.Millisecond)

	select {
	case <-tr.initCh:
		// expected: closed
	default:
		t.Error("initCh should be closed after control_response")
	}

	// Duplicate control_response should not panic
	writeLine(pw, map[string]any{
		"type": "control_response",
		"response": map[string]any{
			"subtype":    "success",
			"request_id": "init_1",
		},
	})
	time.Sleep(50 * time.Millisecond)
}

func TestTransport_Handshake_ContextCancel(t *testing.T) {
	tr := NewTransport(acp.EventCallbacks{}, slog.Default())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := tr.Handshake(ctx)
	if err == nil {
		t.Error("expected context cancel error")
	}
}

func TestTransport_Close(t *testing.T) {
	NewTransport(acp.EventCallbacks{}, slog.Default()).Close()
}
