package acp

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// RequestTracker manages pending JSON-RPC request/response matching.
// Both ACPTransport and Codex Transport use this to avoid duplication.
type RequestTracker struct {
	Writer *Writer
	Logger *slog.Logger
	Ctx    func() <-chan struct{} // returns ctx.Done() channel

	pending   map[int64]chan *JSONRPCResponse
	pendingMu sync.Mutex
}

// NewRequestTracker creates a tracker for matching RPC responses.
func NewRequestTracker(writer *Writer, logger *slog.Logger, ctxDone func() <-chan struct{}) *RequestTracker {
	return &RequestTracker{
		Writer:  writer,
		Logger:  logger,
		Ctx:     ctxDone,
		pending: make(map[int64]chan *JSONRPCResponse),
	}
}

// PendingRequest is a handle returned by SendRequest, holding the
// response channel. Pass it to WaitResponse to collect the reply.
type PendingRequest struct {
	ID int64
	ch chan *JSONRPCResponse
}

// SendRequest pre-registers a response channel, then writes the request.
// Returns a PendingRequest token for use with WaitResponse.
func (rt *RequestTracker) SendRequest(method string, params any) (*PendingRequest, error) {
	id := NextRequestID()
	ch := make(chan *JSONRPCResponse, 1)

	rt.pendingMu.Lock()
	rt.pending[id] = ch
	rt.pendingMu.Unlock()

	if err := rt.Writer.WriteRequestWithID(id, method, params); err != nil {
		rt.pendingMu.Lock()
		delete(rt.pending, id)
		rt.pendingMu.Unlock()
		return nil, err
	}
	return &PendingRequest{ID: id, ch: ch}, nil
}

// WaitResponse blocks until a response arrives on the pending request's
// channel, the timeout expires, or the context is cancelled.
func (rt *RequestTracker) WaitResponse(pr *PendingRequest, timeout time.Duration) (*JSONRPCResponse, error) {
	select {
	case resp := <-pr.ch:
		return resp, nil
	case <-time.After(timeout):
		rt.pendingMu.Lock()
		delete(rt.pending, pr.ID)
		rt.pendingMu.Unlock()
		return nil, fmt.Errorf("timeout waiting for response id=%d", pr.ID)
	case <-rt.Ctx():
		return nil, fmt.Errorf("context cancelled")
	}
}

// HandleResponse matches an inbound response to a pending request.
func (rt *RequestTracker) HandleResponse(msg *JSONRPCMessage) {
	id, ok := msg.GetID()
	if !ok {
		rt.Logger.Warn("response with unparseable ID")
		return
	}

	rt.pendingMu.Lock()
	ch, exists := rt.pending[id]
	if exists {
		delete(rt.pending, id)
	}
	rt.pendingMu.Unlock()

	if !exists {
		rt.Logger.Warn("unmatched response", "id", id)
		return
	}

	resp := &JSONRPCResponse{
		JSONRPC: msg.JSONRPC,
		ID:      &id,
		Result:  msg.Result,
		Error:   msg.Error,
	}
	ch <- resp
}

// RejectRequest sends a "method not found" error response for an
// unsupported inbound request from the agent.
func (rt *RequestTracker) RejectRequest(msg *JSONRPCMessage) {
	id, _ := msg.GetID()
	_ = rt.Writer.WriteResponse(id, nil, &JSONRPCError{
		Code:    ErrCodeMethodNotFound,
		Message: fmt.Sprintf("method %q not supported", msg.Method),
	})
}
