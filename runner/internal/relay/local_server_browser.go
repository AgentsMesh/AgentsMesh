package relay

import (
	"net/http"
	"time"
)

// handleBrowser authenticates and attaches one local browser WebSocket.
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
	client := newLocalConn(conn, lane, s.logger.With("pod_key", podKey))
	client.socket.SetReadLimit(localReadLimitBytes)
	_ = client.socket.SetReadDeadline(time.Now().Add(localReadTimeout))
	client.socket.SetPongHandler(func(string) error {
		return client.socket.SetReadDeadline(time.Now().Add(localReadTimeout))
	})

	lane.mu.Lock()
	if lane.conns == nil {
		lane.mu.Unlock()
		client.close()
		client.waitWriter()
		return
	}
	lane.conns[client] = struct{}{}
	lane.mu.Unlock()

	s.logger.Info("Local relay browser connected", "pod_key", podKey)
	go s.readLoop(podKey, lane, client)
}
