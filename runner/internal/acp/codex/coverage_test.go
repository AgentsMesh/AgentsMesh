package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

func TestTransport_ReadLoop_ReaderError(t *testing.T) {
	// Simulate a reader that returns an error immediately
	tr := NewTransport(acp.EventCallbacks{}, discardLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tr.Initialize(ctx, io.Discard, strings.NewReader(""), nil)
	// ReadLoop should exit on EOF without panic
	tr.ReadLoop(ctx)
}

func TestTransport_ReadLoop_ContextCancel(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	tr := NewTransport(acp.EventCallbacks{}, discardLogger())
	ctx, cancel := context.WithCancel(context.Background())
	tr.Initialize(ctx, io.Discard, pr, nil)

	done := make(chan struct{})
	go func() { tr.ReadLoop(ctx); close(done) }()

	cancel()
	pr.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ReadLoop did not exit on context cancel")
	}
}

func TestTransport_DispatchMessage_AgentRequest(t *testing.T) {
	// Test that incoming requests from the agent get a method-not-found error
	stdoutPR, stdoutPW := io.Pipe()
	stdinPR, stdinPW := io.Pipe()
	defer stdoutPR.Close()
	defer stdinPR.Close()

	tr := NewTransport(acp.EventCallbacks{}, discardLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tr.Initialize(ctx, stdinPW, stdoutPR, nil)
	go tr.ReadLoop(ctx)

	// Send a request (has both method and id) — agent requesting something
	msg := map[string]any{
		"jsonrpc": "2.0",
		"id":      42,
		"method":  "agent/unsupported",
		"params":  map[string]any{},
	}
	writeLine(stdoutPW, msg)

	// Read the error response from stdin
	scanner := bufio.NewScanner(stdinPR)
	scanner.Scan()
	var resp struct {
		Error *acp.JSONRPCError `json:"error"`
	}
	json.Unmarshal(scanner.Bytes(), &resp)
	if resp.Error == nil || resp.Error.Code != acp.ErrCodeMethodNotFound {
		t.Errorf("expected method-not-found error, got %+v", resp.Error)
	}
}

func TestTransport_NewSession_Error(t *testing.T) {
	stdoutPR, stdoutPW := io.Pipe()
	stdinPR, stdinPW := io.Pipe()
	defer stdoutPR.Close()
	defer stdinPR.Close()

	tr := NewTransport(acp.EventCallbacks{}, discardLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tr.Initialize(ctx, stdinPW, stdoutPR, nil)
	go tr.ReadLoop(ctx)

	go func() {
		scanner := bufio.NewScanner(stdinPR)
		scanner.Scan()
		var req acp.JSONRPCRequest
		json.Unmarshal(scanner.Bytes(), &req)
		writeResponse(stdoutPW, req.ID, nil, &acp.JSONRPCError{Code: -1, Message: "no thread"})
		io.Copy(io.Discard, stdinPR)
	}()

	_, err := tr.NewSession("", nil)
	if err == nil {
		t.Fatal("expected error from NewSession")
	}
}

func TestTransport_NewSession_ParseError(t *testing.T) {
	stdoutPR, stdoutPW := io.Pipe()
	stdinPR, stdinPW := io.Pipe()
	defer stdoutPR.Close()
	defer stdinPR.Close()

	tr := NewTransport(acp.EventCallbacks{}, discardLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tr.Initialize(ctx, stdinPW, stdoutPR, nil)
	go tr.ReadLoop(ctx)

	go func() {
		scanner := bufio.NewScanner(stdinPR)
		scanner.Scan()
		var req acp.JSONRPCRequest
		json.Unmarshal(scanner.Bytes(), &req)
		// Return invalid result that can't be parsed as threadStartResult
		writeResponse(stdoutPW, req.ID, "not-an-object", nil)
		io.Copy(io.Discard, stdinPR)
	}()

	_, err := tr.NewSession("", nil)
	if err == nil || !strings.Contains(err.Error(), "parse thread/start result") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestHandler_SessionIDTracking(t *testing.T) {
	// Verify that sessionID from NewSession is passed in callbacks
	f := newFixture()
	defer f.Close()

	// Manually set sessionID (simulates NewSession having been called)
	f.Transport.sessionMu.Lock()
	f.Transport.sessionID = "thread-xyz"
	f.Transport.sessionMu.Unlock()

	writeNotification(f.PW, "item/agentMessage/delta", agentMessageDelta{Delta: "hi"})
	f.Drain()

	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.Chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(f.Chunks))
	}
	// The fixture helper collects chunks but doesn't expose sessionID.
	// We verify via a custom callback.
}

func TestHandler_SessionIDInCallbacks(t *testing.T) {
	// Use a custom fixture to capture sessionID in callbacks
	stdoutPR, stdoutPW := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	var capturedSID string
	tr := NewTransport(acp.EventCallbacks{
		OnContentChunk: func(sid string, _ acp.ContentChunk) {
			mu.Lock()
			capturedSID = sid
			mu.Unlock()
		},
	}, discardLogger())
	tr.Initialize(ctx, io.Discard, stdoutPR, nil)
	go tr.ReadLoop(ctx)

	// Set session ID
	tr.sessionMu.Lock()
	tr.sessionID = "thread-abc"
	tr.sessionMu.Unlock()

	writeNotification(stdoutPW, "item/agentMessage/delta", agentMessageDelta{Delta: "test"})
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	got := capturedSID
	mu.Unlock()
	if got != "thread-abc" {
		t.Errorf("sessionID = %q, want thread-abc", got)
	}
}

func TestTransport_HandleResponse_UnparseableID(t *testing.T) {
	f := newFixture()
	defer f.Close()
	// Send a response with non-numeric ID — should be silently ignored
	msg := map[string]any{
		"jsonrpc": "2.0",
		"id":      "not-a-number",
		"result":  map[string]any{},
	}
	writeLine(f.PW, msg)
	f.Drain()
}

func TestTransport_HandleResponse_UnmatchedID(t *testing.T) {
	f := newFixture()
	defer f.Close()
	// Send a response with ID that nobody is waiting for
	writeResponse(f.PW, 99999, map[string]any{}, nil)
	f.Drain()
}

func TestTransport_Handshake_WriteInitializedError(t *testing.T) {
	// After successful initialize response, writing "initialized" notification fails
	stdoutPR, stdoutPW := io.Pipe()
	stdinPR, stdinPW := io.Pipe()
	defer stdoutPR.Close()

	tr := NewTransport(acp.EventCallbacks{}, discardLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tr.Initialize(ctx, stdinPW, stdoutPR, nil)
	go tr.ReadLoop(ctx)

	go func() {
		scanner := bufio.NewScanner(stdinPR)
		scanner.Scan()
		var req acp.JSONRPCRequest
		json.Unmarshal(scanner.Bytes(), &req)
		writeResponse(stdoutPW, req.ID, map[string]any{}, nil)
		// Close stdin reader immediately so "initialized" notification write fails
		stdinPR.Close()
	}()

	_, err := tr.Handshake(ctx)
	if err == nil {
		t.Fatal("expected error from write initialized")
	}
	if !strings.Contains(err.Error(), "write initialized") {
		t.Errorf("error = %v, want 'write initialized'", err)
	}
}

func TestTransport_DispatchApprovalRequest(t *testing.T) {
	f := newFixture()
	defer f.Close()

	// Send a JSON-RPC request (not notification!) for command approval
	writeLine(f.PW, map[string]any{
		"jsonrpc": "2.0",
		"id":      42,
		"method":  "item/commandExecution/requestApproval",
		"params": map[string]any{
			"command":     "rm -rf /tmp/test",
			"description": "Delete temp files",
		},
	})
	f.Drain()

	f.mu.Lock()
	defer f.mu.Unlock()

	// Verify StateWaitingPermission was emitted
	if len(f.StateChanges) != 1 || f.StateChanges[0] != acp.StateWaitingPermission {
		t.Errorf("StateChanges = %v, want [%q]", f.StateChanges, acp.StateWaitingPermission)
	}
	// Verify PermissionRequest callback
	if len(f.PermissionReqs) != 1 {
		t.Fatalf("expected 1 permission request, got %d", len(f.PermissionReqs))
	}
	pr := f.PermissionReqs[0]
	if pr.RequestID != "42" {
		t.Errorf("RequestID = %q, want %q", pr.RequestID, "42")
	}
	if pr.ToolName != "command" {
		t.Errorf("ToolName = %q, want %q", pr.ToolName, "command")
	}
	if pr.Description != "Delete temp files" {
		t.Errorf("Description = %q, want %q", pr.Description, "Delete temp files")
	}
}

func TestTransport_DispatchFileChangeApproval(t *testing.T) {
	f := newFixture()
	defer f.Close()

	// Send file change approval request
	writeLine(f.PW, map[string]any{
		"jsonrpc": "2.0",
		"id":      99,
		"method":  "item/fileChange/requestApproval",
		"params": map[string]any{
			"path": "/tmp/config.yaml",
		},
	})
	f.Drain()

	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.StateChanges) != 1 || f.StateChanges[0] != acp.StateWaitingPermission {
		t.Errorf("StateChanges = %v, want [%q]", f.StateChanges, acp.StateWaitingPermission)
	}
	if len(f.PermissionReqs) != 1 {
		t.Fatalf("expected 1 permission request, got %d", len(f.PermissionReqs))
	}
	pr := f.PermissionReqs[0]
	if pr.RequestID != "99" {
		t.Errorf("RequestID = %q, want %q", pr.RequestID, "99")
	}
	if pr.ToolName != "fileChange" {
		t.Errorf("ToolName = %q, want %q", pr.ToolName, "fileChange")
	}
	if pr.Description != "/tmp/config.yaml" {
		t.Errorf("Description = %q, want %q", pr.Description, "/tmp/config.yaml")
	}
}

func TestTransport_NewSession_StoresSessionID(t *testing.T) {
	stdoutPR, stdoutPW := io.Pipe()
	stdinPR, stdinPW := io.Pipe()
	defer stdoutPR.Close()
	defer stdinPR.Close()

	tr := NewTransport(acp.EventCallbacks{}, discardLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tr.Initialize(ctx, stdinPW, stdoutPR, nil)
	go tr.ReadLoop(ctx)

	go func() {
		scanner := bufio.NewScanner(stdinPR)
		scanner.Scan()
		var req acp.JSONRPCRequest
		json.Unmarshal(scanner.Bytes(), &req)
		writeResponse(stdoutPW, req.ID, map[string]any{
			"thread": map[string]string{"id": "thread-stored"},
		}, nil)
	}()

	sid, err := tr.NewSession("", nil)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if sid != "thread-stored" {
		t.Errorf("returned sid = %q", sid)
	}
	// Verify internal state
	if got := tr.getSessionID(); got != "thread-stored" {
		t.Errorf("stored sessionID = %q, want thread-stored", got)
	}
}
