package relay

import (
	"time"
)

// healthCheckLoop periodically checks relay health
func (m *Manager) healthCheckLoop() {
	ticker := time.NewTicker(m.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			m.logger.Info("Health check loop stopped")
			return
		case <-ticker.C:
			m.doHealthCheck()
		}
	}
}

// doHealthCheck performs a single health check iteration
func (m *Manager) doHealthCheck() {
	now := time.Now()
	healthyTimeout := 30 * time.Second

	// Read-lock to collect relays that need attention
	m.mu.RLock()
	var unhealthyRelays []string

	for id, relay := range m.relays {
		if now.Sub(relay.LastHeartbeat) > healthyTimeout && relay.Healthy {
			unhealthyRelays = append(unhealthyRelays, id)
		}
	}
	m.mu.RUnlock()

	// Process unhealthy relays (needs write lock)
	for _, relayID := range unhealthyRelays {
		m.markRelayUnhealthy(relayID)
	}
}

// markRelayUnhealthy marks a relay as unhealthy
func (m *Manager) markRelayUnhealthy(relayID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	relay, ok := m.relays[relayID]
	if !ok || !relay.Healthy {
		return
	}

	relay.Healthy = false
	m.logger.Warn("Relay marked unhealthy", "relay_id", relayID, "last_heartbeat", relay.LastHeartbeat)
}
