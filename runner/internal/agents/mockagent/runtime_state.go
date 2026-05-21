package mockagent

import (
	"context"
	"sync"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

// runtimeState carries the shared state scenarios need: the writer (for
// emitting notifications/requests), a registry of pending outgoing requests
// (e.g. session/request_permission round-trip), a WaitGroup so
// runACPWithIO can drain in-flight scenario goroutines on EOF, and the
// current configuration (settable via session/control_request).
type runtimeState struct {
	writer    *acp.Writer
	wg        sync.WaitGroup
	pendingMu sync.Mutex
	pending   map[int64]chan *acp.JSONRPCMessage
	configMu  sync.RWMutex
	mode      string
	model     string
}

func newRuntimeState(writer *acp.Writer) *runtimeState {
	return &runtimeState{
		writer:  writer,
		pending: make(map[int64]chan *acp.JSONRPCMessage),
	}
}

// awaitResponse registers a channel for the given outgoing request id and
// blocks until the matching JSON-RPC response arrives (or ctx is done).
func (s *runtimeState) awaitResponse(ctx context.Context, id int64) (*acp.JSONRPCMessage, error) {
	ch := make(chan *acp.JSONRPCMessage, 1)
	s.pendingMu.Lock()
	s.pending[id] = ch
	s.pendingMu.Unlock()
	defer func() {
		s.pendingMu.Lock()
		delete(s.pending, id)
		s.pendingMu.Unlock()
	}()
	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *runtimeState) deliverResponse(msg *acp.JSONRPCMessage) {
	id, ok := msg.GetID()
	if !ok {
		return
	}
	s.pendingMu.Lock()
	ch, found := s.pending[id]
	s.pendingMu.Unlock()
	if !found {
		return
	}
	select {
	case ch <- msg:
	default:
	}
}

func (s *runtimeState) setPermissionMode(mode string) {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	s.mode = mode
}

func (s *runtimeState) setModel(model string) {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	s.model = model
}

func (s *runtimeState) permissionMode() string {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.mode
}
