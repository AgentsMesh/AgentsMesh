package relay

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
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
			conns:          make(map[*websocket.Conn]struct{}),
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
	lane.mu.Lock()
	for c := range lane.conns {
		_ = c.Close()
	}
	lane.conns = nil
	lane.handlers = nil
	lane.reqHandlers = nil
	lane.expectedTokens = nil
	lane.mu.Unlock()
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

// writeFrame writes one pre-encoded frame to a single browser connection,
// serialized by the lane's writeMu so concurrent Send / reply goroutines never
// call WriteMessage on the same conn at once (gorilla forbids it).
func (l *localPodLane) writeFrame(conn *websocket.Conn, frame []byte) error {
	l.writeMu.Lock()
	defer l.writeMu.Unlock()
	_ = conn.SetWriteDeadline(time.Now().Add(localWriteTimeout))
	return conn.WriteMessage(websocket.BinaryMessage, frame)
}

// writeConn frames then writes one message to a single connection. Used by the
// per-connection reply path; broadcast (Send) encodes once and calls writeFrame.
func (l *localPodLane) writeConn(conn *websocket.Conn, msgType byte, payload []byte) error {
	return l.writeFrame(conn, EncodeMessage(msgType, payload))
}

// Send broadcasts a message to every browser connected for this pod. The frame
// is encoded once and shared (read-only) across conns, avoiding a per-conn
// re-encode on the hot PTY-output fanout path. Returns nil even when there are
// no listeners — output drops are expected. On per-conn write error we close the
// conn so the read loop drops it from lane.conns immediately.
func (s *LocalServer) Send(podKey string, msgType byte, payload []byte) error {
	lane := s.lookupLane(podKey)
	if lane == nil {
		return nil
	}
	frame := EncodeMessage(msgType, payload)
	for _, c := range lane.snapshotConns() {
		if err := lane.writeFrame(c, frame); err != nil {
			_ = c.Close()
		}
	}
	return nil
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

// handleBrowser is the WebSocket entry point for browser connections.
// Path: /browser/relay?token=<jwt>&pod=<pod_key>
func (s *LocalServer) handleBrowser(w http.ResponseWriter, r *http.Request) {
	podKey := r.URL.Query().Get("pod")
	token := r.URL.Query().Get("token")
	if podKey == "" || token == "" {
		http.Error(w, "pod and token required", http.StatusBadRequest)
		return
	}

	lane := s.lookupLane(podKey)
	if lane == nil {
		http.Error(w, "unknown pod", http.StatusNotFound)
		return
	}
	lane.mu.RLock()
	_, accepted := lane.expectedTokens[token]
	lane.mu.RUnlock()
	if !accepted {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := localUpgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Warn("Local relay upgrade failed", "pod_key", podKey, "error", err)
		return
	}
	conn.SetReadLimit(localReadLimitBytes)
	_ = conn.SetReadDeadline(time.Now().Add(localReadTimeout))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(localReadTimeout))
	})

	lane.mu.Lock()
	if lane.conns == nil {
		lane.mu.Unlock()
		_ = conn.Close()
		return
	}
	lane.conns[conn] = struct{}{}
	lane.mu.Unlock()

	s.logger.Info("Local relay browser connected", "pod_key", podKey)

	go s.readLoop(podKey, lane, conn)
}

func (s *LocalServer) readLoop(podKey string, lane *localPodLane, conn *websocket.Conn) {
	defer func() {
		lane.mu.Lock()
		delete(lane.conns, conn)
		lane.mu.Unlock()
		_ = conn.Close()
		s.logger.Info("Local relay browser disconnected", "pod_key", podKey)
	}()

	for {
		mt, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(localReadTimeout))
		if mt != websocket.BinaryMessage || len(data) < 1 {
			continue
		}
		msgType := data[0]
		payload := data[1:]
		lane.mu.RLock()
		reqHandler := lane.reqHandlers[msgType]
		handler := lane.handlers[msgType]
		lane.mu.RUnlock()
		if reqHandler != nil {
			reqHandler(payload, func(mt byte, p []byte) {
				if err := lane.writeConn(conn, mt, p); err != nil {
					_ = conn.Close()
				}
			})
		} else if handler != nil {
			handler(payload)
		}
	}
}

func (l *localPodLane) snapshotConns() []*websocket.Conn {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]*websocket.Conn, 0, len(l.conns))
	for c := range l.conns {
		out = append(out, c)
	}
	return out
}
