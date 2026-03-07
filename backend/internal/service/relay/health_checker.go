package relay

import (
	"time"
)

// staleRelayMultiplier defines how many health check intervals a relay can remain
// without heartbeat before being automatically removed from memory and store.
const staleRelayMultiplier = 10

// healthCheckLoop periodically checks relay health
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

// doHealthCheck performs a single health check iteration.
// Marks relays unhealthy after one missed interval, removes stale relays after staleRelayMultiplier intervals.
func (m *Manager) doHealthCheck() {
	now := time.Now()
	healthyTimeout := m.healthCheckInterval
	staleTimeout := m.healthCheckInterval * staleRelayMultiplier

	// Read-lock to collect relays that need attention
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

	// Process unhealthy relays (needs write lock)
	for _, relayID := range unhealthyRelays {
		m.markRelayUnhealthy(relayID)
	}

	// Remove stale relays that have been silent for too long
	for _, relayID := range staleRelays {
		m.removeStaleRelay(relayID)
	}
}

// markRelayUnhealthy marks a relay as unhealthy.
// Re-checks elapsed time under write lock to prevent TOCTOU: the relay may have
// heartbeated between the RLock collection in doHealthCheck and this write-lock update.
func (m *Manager) markRelayUnhealthy(relayID string) {
	healthyTimeout := m.healthCheckInterval

	m.mu.Lock()
	defer m.mu.Unlock()

	relay, ok := m.relays[relayID]
	if !ok || !relay.Healthy {
		return
	}

	// Re-check: relay may have heartbeated since the RLock scan
	if time.Since(relay.LastHeartbeat) <= healthyTimeout {
		return
	}

	relay.Healthy = false
	m.logger.Warn("Relay marked unhealthy", "relay_id", relayID, "last_heartbeat", relay.LastHeartbeat)
}

// removeStaleRelay auto-removes a relay that has been unresponsive for an extended period.
// Re-checks staleness under write lock to prevent TOCTOU: the relay may have re-registered
// between the RLock collection in doHealthCheck and this write-lock deletion.
func (m *Manager) removeStaleRelay(relayID string) {
	staleTimeout := m.healthCheckInterval * staleRelayMultiplier

	m.mu.Lock()
	r, ok := m.relays[relayID]
	if !ok {
		m.mu.Unlock()
		return
	}
	// Re-check: relay may have heartbeated or re-registered since RLock scan
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
