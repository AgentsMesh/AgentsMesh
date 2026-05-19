package relay

import (
	"time"
)

const staleRelayMultiplier = 10

func (m *Manager) healthCheckLoop() {
	defer m.wg.Done()

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

func (m *Manager) doHealthCheck() {
	now := time.Now()
	healthyTimeout := m.healthCheckInterval
	staleTimeout := m.healthCheckInterval * staleRelayMultiplier

	m.mu.RLock()
	var unhealthyRelays []string
	var staleRelays []string

	for id, r := range m.relays {
		elapsed := now.Sub(r.LastHeartbeat)
		if elapsed > staleTimeout {
			staleRelays = append(staleRelays, id)
		} else if elapsed > healthyTimeout && r.Healthy {
			unhealthyRelays = append(unhealthyRelays, id)
		}
	}
	m.mu.RUnlock()

	for _, relayID := range unhealthyRelays {
		m.markRelayUnhealthy(relayID)
	}

	for _, relayID := range staleRelays {
		m.removeStaleRelay(relayID)
	}
}

func (m *Manager) markRelayUnhealthy(relayID string) {
	healthyTimeout := m.healthCheckInterval

	m.mu.Lock()
	defer m.mu.Unlock()

	relay, ok := m.relays[relayID]
	if !ok || !relay.Healthy {
		return
	}

	if time.Since(relay.LastHeartbeat) <= healthyTimeout {
		return
	}

	relay.Healthy = false
	m.logger.Warn("Relay marked unhealthy", "relay_id", relayID, "last_heartbeat", relay.LastHeartbeat)
}

func (m *Manager) removeStaleRelay(relayID string) {
	staleTimeout := m.healthCheckInterval * staleRelayMultiplier

	m.mu.Lock()
	r, ok := m.relays[relayID]
	if !ok {
		m.mu.Unlock()
		return
	}
	if time.Since(r.LastHeartbeat) <= staleTimeout {
		m.mu.Unlock()
		return
	}
	lastHB := r.LastHeartbeat
	delete(m.relays, relayID)
	m.mu.Unlock()

	m.deleteFromStore(relayID)
	m.logger.Warn("Stale relay auto-removed",
		"relay_id", relayID,
		"last_heartbeat", lastHB)
}
