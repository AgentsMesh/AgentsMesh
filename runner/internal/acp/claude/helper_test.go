package claude

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

// writeLine writes an NDJSON line to a writer.
func writeLine(w io.Writer, v any) {
	data, _ := json.Marshal(v)
	w.Write(append(data, '\n'))
}

// errorReader always returns an error on Read.
type errorReader struct{ err error }

func (r *errorReader) Read([]byte) (int, error) { return 0, r.err }

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// testFixture provides common setup for Transport tests.
type testFixture struct {
	Transport *Transport
	PW        *io.PipeWriter
	StdinPR   *io.PipeReader // stdin read end for verifying writes
	Cancel    context.CancelFunc

	mu                 sync.Mutex
	StateChanges       []string
	Chunks             []acp.ContentChunk
	ToolUpdates        []acp.ToolCallUpdate
	ToolResults        []acp.ToolCallResult
	ThinkingTexts      []string
	LogMessages        []string
	PermissionRequests []acp.PermissionRequest
}

// newFixture creates a transport wired to a pipe with all callbacks
// collecting into thread-safe slices. Call f.Close() when done.
func newFixture() *testFixture {
	pr, pw := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	f := &testFixture{PW: pw, Cancel: cancel}
	f.Transport = NewTransport(acp.EventCallbacks{
		OnStateChange:  func(s string) { f.mu.Lock(); f.StateChanges = append(f.StateChanges, s); f.mu.Unlock() },
		OnContentChunk: func(_ string, c acp.ContentChunk) { f.mu.Lock(); f.Chunks = append(f.Chunks, c); f.mu.Unlock() },
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
		OnLog: func(l, m string) { f.mu.Lock(); f.LogMessages = append(f.LogMessages, l+":"+m); f.mu.Unlock() },
		OnPermissionRequest: func(req acp.PermissionRequest) {
			f.mu.Lock()
			f.PermissionRequests = append(f.PermissionRequests, req)
			f.mu.Unlock()
		},
	}, slog.Default())
	f.Transport.Initialize(ctx, nil, pr, nil)
	go f.Transport.ReadLoop(ctx)
	return f
}

// newFixtureWithStdin creates a fixture with stdin wired for write verification.
func newFixtureWithStdin() *testFixture {
	pr, pw := io.Pipe()
	stdinPR, stdinPW := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	f := &testFixture{PW: pw, StdinPR: stdinPR, Cancel: cancel}
	f.Transport = NewTransport(acp.EventCallbacks{
		OnStateChange:  func(s string) { f.mu.Lock(); f.StateChanges = append(f.StateChanges, s); f.mu.Unlock() },
		OnContentChunk: func(_ string, c acp.ContentChunk) { f.mu.Lock(); f.Chunks = append(f.Chunks, c); f.mu.Unlock() },
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
		OnLog: func(l, m string) { f.mu.Lock(); f.LogMessages = append(f.LogMessages, l+":"+m); f.mu.Unlock() },
		OnPermissionRequest: func(req acp.PermissionRequest) {
			f.mu.Lock()
			f.PermissionRequests = append(f.PermissionRequests, req)
			f.mu.Unlock()
		},
	}, slog.Default())
	f.Transport.Initialize(ctx, stdinPW, pr, nil)
	go f.Transport.ReadLoop(ctx)
	return f
}

func (f *testFixture) Close() {
	f.Cancel()
	f.PW.Close()
	if f.StdinPR != nil {
		f.StdinPR.Close()
	}
}

// Drain waits briefly for async ReadLoop processing.
func (f *testFixture) Drain() { time.Sleep(100 * time.Millisecond) }

// Snap returns a thread-safe copy of collected state.
func (f *testFixture) Snap() (states []string, chunks []acp.ContentChunk, tools []acp.ToolCallUpdate, results []acp.ToolCallResult) {
	f.mu.Lock()
	defer f.mu.Unlock()
	states = append(states, f.StateChanges...)
	chunks = append(chunks, f.Chunks...)
	tools = append(tools, f.ToolUpdates...)
	results = append(results, f.ToolResults...)
	return
}
