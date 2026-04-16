package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

func TestTransport_Handshake(t *testing.T) {
	stdoutPR, stdoutPW := io.Pipe()
	stdinPR, stdinPW := io.Pipe()
	defer stdoutPR.Close()
	defer stdinPR.Close()

	tr := newTransport(acp.EventCallbacks{}, slog.Default())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tr.Initialize(ctx, stdinPW, stdoutPR, nil)
	go tr.ReadLoop(ctx)

	go func() {
		scanner := bufio.NewScanner(stdinPR)
		scanner.Scan()
		var req acp.JSONRPCRequest
		json.Unmarshal(scanner.Bytes(), &req)
		writeResponse(stdoutPW, req.ID, map[string]any{"server_info": map[string]string{"name": "codex"}}, nil)
		io.Copy(io.Discard, stdinPR)
	}()

	sid, err := tr.Handshake(ctx)
	if err != nil {
		t.Fatalf("Handshake: %v", err)
	}
	if sid != "" {
		t.Errorf("expected empty session_id, got %q", sid)
	}
}

func TestTransport_Handshake_Error(t *testing.T) {
	stdoutPR, stdoutPW := io.Pipe()
	stdinPR, stdinPW := io.Pipe()
	defer stdoutPR.Close()
	defer stdinPR.Close()

	tr := newTransport(acp.EventCallbacks{}, discardLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tr.Initialize(ctx, stdinPW, stdoutPR, nil)
	go tr.ReadLoop(ctx)

	go func() {
		scanner := bufio.NewScanner(stdinPR)
		scanner.Scan()
		var req acp.JSONRPCRequest
		json.Unmarshal(scanner.Bytes(), &req)
		writeResponse(stdoutPW, req.ID, nil, &acp.JSONRPCError{Code: -32600, Message: "bad request"})
		io.Copy(io.Discard, stdinPR)
	}()

	_, err := tr.Handshake(ctx)
	if err == nil {
		t.Fatal("expected error from handshake")
	}
}

func TestTransport_NewSession(t *testing.T) {
	stdoutPR, stdoutPW := io.Pipe()
	stdinPR, stdinPW := io.Pipe()
	defer stdoutPR.Close()
	defer stdinPR.Close()

	tr := newTransport(acp.EventCallbacks{}, discardLogger())
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
			"thread": map[string]string{"id": "thread-abc"},
		}, nil)
	}()

	sid, err := tr.NewSession("", nil)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if sid != "thread-abc" {
		t.Errorf("session_id = %q, want thread-abc", sid)
	}
}

func TestTransport_SendPrompt(t *testing.T) {
	stdoutPR, stdoutPW := io.Pipe()
	stdinPR, stdinPW := io.Pipe()
	defer stdoutPR.Close()
	defer stdinPR.Close()

	tr := newTransport(acp.EventCallbacks{}, discardLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tr.Initialize(ctx, stdinPW, stdoutPR, nil)
	go tr.ReadLoop(ctx)

	received := make(chan acp.JSONRPCRequest, 1)
	go func() {
		scanner := bufio.NewScanner(stdinPR)
		scanner.Scan()
		var req acp.JSONRPCRequest
		json.Unmarshal(scanner.Bytes(), &req)
		received <- req
		writeResponse(stdoutPW, req.ID, map[string]any{}, nil)
	}()

	if err := tr.SendPrompt("thread-1", "hello codex"); err != nil {
		t.Fatalf("SendPrompt: %v", err)
	}

	select {
	case req := <-received:
		if req.Method != "turn/start" {
			t.Errorf("method = %q, want turn/start", req.Method)
		}
		var params turnStartParams
		json.Unmarshal(req.Params, &params)
		if params.ThreadID != "thread-1" {
			t.Errorf("threadId = %q", params.ThreadID)
		}
		if len(params.Input) == 0 || params.Input[0].Text != "hello codex" {
			t.Errorf("input = %+v", params.Input)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for turn/start")
	}
}

func TestTransport_RespondToPermission(t *testing.T) {
	stdoutPR, _ := io.Pipe()
	stdinPR, stdinPW := io.Pipe()
	defer stdoutPR.Close()
	defer stdinPR.Close()

	tr := newTransport(acp.EventCallbacks{}, discardLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tr.Initialize(ctx, stdinPW, stdoutPR, nil)

	received := make(chan json.RawMessage, 1)
	go func() {
		scanner := bufio.NewScanner(stdinPR)
		scanner.Scan()
		received <- json.RawMessage(scanner.Bytes())
	}()

	if err := tr.RespondToPermission("42", true, nil); err != nil {
		t.Fatalf("RespondToPermission: %v", err)
	}

	select {
	case raw := <-received:
		var msg struct {
			ID     *int64          `json:"id"`
			Result json.RawMessage `json:"result"`
		}
		json.Unmarshal(raw, &msg)
		if msg.ID == nil || *msg.ID != 42 {
			t.Errorf("expected id=42, got %v", msg.ID)
		}
		var result map[string]any
		json.Unmarshal(msg.Result, &result)
		if result["decision"] != "accept" {
			t.Errorf("decision = %v, want accept", result["decision"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestTransport_RespondToPermission_Decline(t *testing.T) {
	stdoutPR, _ := io.Pipe()
	stdinPR, stdinPW := io.Pipe()
	defer stdoutPR.Close()
	defer stdinPR.Close()

	tr := newTransport(acp.EventCallbacks{}, discardLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tr.Initialize(ctx, stdinPW, stdoutPR, nil)

	received := make(chan json.RawMessage, 1)
	go func() {
		scanner := bufio.NewScanner(stdinPR)
		scanner.Scan()
		received <- json.RawMessage(scanner.Bytes())
	}()

	if err := tr.RespondToPermission("7", false, nil); err != nil {
		t.Fatalf("RespondToPermission: %v", err)
	}

	select {
	case raw := <-received:
		var msg struct {
			ID     *int64          `json:"id"`
			Result json.RawMessage `json:"result"`
		}
		json.Unmarshal(raw, &msg)
		if msg.ID == nil || *msg.ID != 7 {
			t.Errorf("expected id=7, got %v", msg.ID)
		}
		var result map[string]any
		json.Unmarshal(msg.Result, &result)
		if result["decision"] != "decline" {
			t.Errorf("decision = %v, want decline", result["decision"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestTransport_CancelSession(t *testing.T) {
	stdoutPR, _ := io.Pipe()
	stdinPR, stdinPW := io.Pipe()
	defer stdoutPR.Close()
	defer stdinPR.Close()

	tr := newTransport(acp.EventCallbacks{}, discardLogger())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tr.Initialize(ctx, stdinPW, stdoutPR, nil)

	received := make(chan json.RawMessage, 1)
	go func() {
		scanner := bufio.NewScanner(stdinPR)
		scanner.Scan()
		received <- json.RawMessage(scanner.Bytes())
	}()

	if err := tr.CancelSession("thread-1"); err != nil {
		t.Fatalf("CancelSession: %v", err)
	}

	select {
	case raw := <-received:
		var msg struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		json.Unmarshal(raw, &msg)
		if msg.Method != "turn/interrupt" {
			t.Errorf("method = %q", msg.Method)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestTransport_Close(t *testing.T) {
	newTransport(acp.EventCallbacks{}, discardLogger()).Close()
}
