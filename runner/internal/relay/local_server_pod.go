package relay

import (
	"context"
	"errors"
)

// RegisterPod records an accepted token for a pod. Subsequent incoming
// browser connections that present any registered token (via ?token=) are
// accepted. Multi-user shared pods can have multiple live tokens at once;
// tokens are cleared en bloc when UnregisterPod is called or the pod
// terminates. The backend-issued JWT TTL bounds individual token lifetime.
func (s *LocalServer) RegisterPod(podKey, expectedToken string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	lane, ok := s.pods[podKey]
	if !ok {
		lane = &localPodLane{
			expectedTokens: make(map[string]struct{}),
			handlers:       make(map[byte]func([]byte)),
			reqHandlers:    make(map[byte]RequestHandler),
			conns:          make(map[*localConn]struct{}),
		}
		s.pods[podKey] = lane
	}
	lane.mu.Lock()
	if lane.expectedTokens == nil {
		lane.expectedTokens = make(map[string]struct{})
	}
	lane.expectedTokens[expectedToken] = struct{}{}
	lane.mu.Unlock()
}

// UnregisterPod stops accepting new connections for this pod and closes all
// existing browser conns. Subsequent connection attempts are rejected.
func (s *LocalServer) UnregisterPod(podKey string) {
	s.mu.Lock()
	lane, ok := s.pods[podKey]
	if ok {
		delete(s.pods, podKey)
	}
	s.mu.Unlock()
	if !ok {
		return
	}
	conns := lane.shutdown()
	for _, c := range conns {
		c.close()
	}
	for _, c := range conns {
		c.waitWriter()
	}
}

// SetMessageHandler registers an inbound handler for a given message type on
// the pod. Replaces any prior handler for the same type. No-op when the pod
// is not registered.
func (s *LocalServer) SetMessageHandler(podKey string, msgType byte, handler func([]byte)) {
	lane := s.lookupLane(podKey)
	if lane == nil {
		return
	}
	lane.mu.Lock()
	defer lane.mu.Unlock()
	if lane.handlers == nil {
		return
	}
	lane.handlers[msgType] = handler
}

// SetRequestHandler registers a request/response handler for a message type.
// Unlike SetMessageHandler (fire-and-forget broadcast), the handler is given a
// reply func bound to the originating connection, so it answers only that
// browser. Used for snapshot-on-resubscribe: a late joiner's request must not
// re-deliver state to already-synced browsers (which would double-apply
// append-style Loopal bg-task output).
func (s *LocalServer) SetRequestHandler(podKey string, msgType byte, handler RequestHandler) {
	lane := s.lookupLane(podKey)
	if lane == nil {
		return
	}
	lane.mu.Lock()
	defer lane.mu.Unlock()
	if lane.reqHandlers == nil {
		return
	}
	lane.reqHandlers[msgType] = handler
}

// Send broadcasts a message to every browser connected for this pod. The frame
// is encoded once and shared (read-only) across conns, avoiding a per-conn
// re-encode on the hot PTY-output fanout path. Enqueue is non-blocking and one
// overloaded connection cannot delay or fail healthy peers. Returns nil even
// when there are no listeners or an individual connection is closed.
func (s *LocalServer) Send(podKey string, msgType byte, payload []byte) error {
	lane := s.lookupLane(podKey)
	if lane == nil {
		return nil
	}
	frame := EncodeMessage(msgType, payload)
	for _, c := range lane.snapshotConns() {
		c.enqueue(frame)
	}
	return nil
}

// Flush waits until every currently connected browser socket writer has
// processed all frames accepted before this call. Connections flush in
// parallel so one slow browser cannot prevent a healthy peer from receiving
// its marker before the shared context deadline.
func (s *LocalServer) Flush(ctx context.Context, podKey string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	lane := s.lookupLane(podKey)
	if lane == nil {
		return nil
	}
	conns := lane.snapshotConns()
	results := make(chan error, len(conns))
	for _, conn := range conns {
		go func(c *localConn) {
			results <- c.flush(ctx)
		}(conn)
	}

	var flushErr error
	for range conns {
		if err := <-results; err != nil {
			flushErr = errors.Join(flushErr, err)
		}
	}
	return flushErr
}

// IsPodConnected reports whether at least one browser is currently connected
// for this pod.
func (s *LocalServer) IsPodConnected(podKey string) bool {
	lane := s.lookupLane(podKey)
	if lane == nil {
		return false
	}
	lane.mu.RLock()
	defer lane.mu.RUnlock()
	return len(lane.conns) > 0
}

func (s *LocalServer) lookupLane(podKey string) *localPodLane {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pods[podKey]
}

func (l *localPodLane) snapshotConns() []*localConn {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]*localConn, 0, len(l.conns))
	for c := range l.conns {
		out = append(out, c)
	}
	return out
}

func (l *localPodLane) removeConn(conn *localConn) {
	l.mu.Lock()
	if l.conns != nil {
		delete(l.conns, conn)
	}
	l.mu.Unlock()
}

func (l *localPodLane) shutdown() []*localConn {
	l.mu.Lock()
	conns := make([]*localConn, 0, len(l.conns))
	for c := range l.conns {
		conns = append(conns, c)
	}
	l.conns = nil
	l.handlers = nil
	l.reqHandlers = nil
	l.expectedTokens = nil
	l.mu.Unlock()
	return conns
}
