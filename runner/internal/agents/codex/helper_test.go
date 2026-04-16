package codex

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func writeLine(w io.Writer, v any) {
	data, _ := json.Marshal(v)
	w.Write(append(data, '\n'))
}

func writeNotification(w io.Writer, method string, params any) {
	msg := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if params != nil {
		data, _ := json.Marshal(params)
		msg["params"] = json.RawMessage(data)
	}
	writeLine(w, msg)
}

func writeResponse(w io.Writer, id int64, result any, rpcErr *acp.JSONRPCError) {
	msg := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
	}
	if result != nil {
		data, _ := json.Marshal(result)
		msg["result"] = json.RawMessage(data)
	}
	if rpcErr != nil {
		data, _ := json.Marshal(rpcErr)
		msg["error"] = json.RawMessage(data)
	}
	writeLine(w, msg)
}

type testFixture struct {
	transport *transport
	PW        *io.PipeWriter // write to transport's stdin (reader side)
	StdinPR   *io.PipeReader // read what transport writes to process stdin
	Cancel    context.CancelFunc

	mu             sync.Mutex
	StateChanges   []string
	Chunks         []acp.ContentChunk
	ToolUpdates    []acp.ToolCallUpdate
	ToolResults    []acp.ToolCallResult
	ThinkingTexts  []string
	PlanSteps      []acp.PlanStep
	PermissionReqs []acp.PermissionRequest
	LogMessages    []string
}

func newFixture() *testFixture {
	// stdout pipe: test writes -> transport reads
	stdoutPR, stdoutPW := io.Pipe()
	// stdin pipe: transport writes -> test reads
	stdinPR, stdinPW := io.Pipe()

	ctx, cancel := context.WithCancel(context.Background())
	f := &testFixture{
		PW:      stdoutPW,
		StdinPR: stdinPR,
		Cancel:  cancel,
	}
	f.transport = newTransport(acp.EventCallbacks{
		OnStateChange: func(s string) {
			f.mu.Lock()
			f.StateChanges = append(f.StateChanges, s)
			f.mu.Unlock()
		},
		OnContentChunk: func(_ string, c acp.ContentChunk) {
			f.mu.Lock()
			f.Chunks = append(f.Chunks, c)
			f.mu.Unlock()
		},
		OnToolCallUpdate: func(_ string, u acp.ToolCallUpdate) {
			f.mu.Lock()
			f.ToolUpdates = append(f.ToolUpdates, u)
			f.mu.Unlock()
		},
		OnToolCallResult: func(_ string, r acp.ToolCallResult) {
			f.mu.Lock()
			f.ToolResults = append(f.ToolResults, r)
			f.mu.Unlock()
		},
		OnThinkingUpdate: func(_ string, u acp.ThinkingUpdate) {
			f.mu.Lock()
			f.ThinkingTexts = append(f.ThinkingTexts, u.Text)
			f.mu.Unlock()
		},
		OnPlanUpdate: func(_ string, u acp.PlanUpdate) {
			f.mu.Lock()
			f.PlanSteps = append(f.PlanSteps, u.Steps...)
			f.mu.Unlock()
		},
		OnPermissionRequest: func(req acp.PermissionRequest) {
			f.mu.Lock()
			f.PermissionReqs = append(f.PermissionReqs, req)
			f.mu.Unlock()
		},
		OnLog: func(l, m string) {
			f.mu.Lock()
			f.LogMessages = append(f.LogMessages, l+":"+m)
			f.mu.Unlock()
		},
	}, discardLogger())

	f.transport.Initialize(ctx, stdinPW, stdoutPR, nil)
	go f.transport.ReadLoop(ctx)
	return f
}

func (f *testFixture) Close() {
	f.Cancel()
	f.PW.Close()
	f.StdinPR.Close()
}

func (f *testFixture) Drain() { time.Sleep(100 * time.Millisecond) }
