package mockagent

import (
	"context"
	"sync"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

// pendingRegistry tracks outgoing JSON-RPC requests that expect a response
// (e.g. session/request_permission round-trips). It exposes a fast-path
// `register-then-emit` pattern so scenarios can claim a response slot before
// writing the request — eliminating a theoretical race where the agent's
// response could arrive (and be discarded) before the channel is registered.
type pendingRegistry struct {
	mu    sync.Mutex
	slots map[int64]chan *acp.JSONRPCMessage
}

func newPendingRegistry() *pendingRegistry {
	return &pendingRegistry{slots: make(map[int64]chan *acp.JSONRPCMessage)}
}

// reserve claims a slot for the given id and returns a channel that will
// receive the matching response (plus a cleanup function the caller MUST
// invoke when done). reserve must be called BEFORE the corresponding
// outgoing request is written to stdout to avoid the register-vs-emit race.
func (r *pendingRegistry) reserve(id int64) (<-chan *acp.JSONRPCMessage, func()) {
	ch := make(chan *acp.JSONRPCMessage, 1)
	r.mu.Lock()
	r.slots[id] = ch
	r.mu.Unlock()
	return ch, func() {
		r.mu.Lock()
		delete(r.slots, id)
		r.mu.Unlock()
	}
}

// deliver routes an inbound response to the channel reserved for its id.
// Returns silently if no slot was reserved (treated as "response for an
// abandoned scenario"; we don't error since the reader loop has no recourse).
func (r *pendingRegistry) deliver(msg *acp.JSONRPCMessage) {
	id, ok := msg.GetID()
	if !ok {
		return
	}
	r.mu.Lock()
	ch, found := r.slots[id]
	r.mu.Unlock()
	if !found {
		return
	}
	select {
	case ch <- msg:
	default:
	}
}

// awaitWith waits for a response on the given reserved channel, respecting
// ctx cancellation. The caller is responsible for the cleanup function that
// reserve returned — typically `defer cleanup()` right after `ch, cleanup := …`.
func awaitWith(ctx context.Context, ch <-chan *acp.JSONRPCMessage) (*acp.JSONRPCMessage, error) {
	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
