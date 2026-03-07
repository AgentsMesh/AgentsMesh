package relay

// Stats returns relay statistics
type Stats struct {
	TotalRelays      int `json:"total_relays"`
	HealthyRelays    int `json:"healthy_relays"`
	TotalConnections int `json:"total_connections"`
}

// GetStats returns current statistics
func (m *Manager) GetStats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := Stats{
		TotalRelays: len(m.relays),
	}

	for _, relay := range m.relays {
		if relay.Healthy {
			stats.HealthyRelays++
		}
		stats.TotalConnections += relay.CurrentConnections
	}

	return stats
}

// GetRelays returns all registered relays
func (m *Manager) GetRelays() []*RelayInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	relays := make([]*RelayInfo, 0, len(m.relays))
	for _, relay := range m.relays {
		// Copy to avoid data race
		relayCopy := *relay
		relays = append(relays, &relayCopy)
	}
	return relays
}

// GetHealthyRelayCount returns count of healthy relays
func (m *Manager) GetHealthyRelayCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, relay := range m.relays {
		if relay.Healthy {
			count++
		}
	}
	return count
}

// HasHealthyRelays checks if there are any healthy relays.
// Inlined with early return to avoid iterating all relays.
func (m *Manager) HasHealthyRelays() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, r := range m.relays {
		if r.Healthy {
			return true
		}
	}
	return false
}

// GetRelayByID returns a relay by ID
func (m *Manager) GetRelayByID(relayID string) *RelayInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if relay, ok := m.relays[relayID]; ok {
		relayCopy := *relay
		return &relayCopy
	}
	return nil
}
