package relay

type Stats struct {
	TotalRelays      int `json:"total_relays"`
	HealthyRelays    int `json:"healthy_relays"`
	TotalConnections int `json:"total_connections"`
}

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

func (m *Manager) GetRelays() []*RelayInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	relays := make([]*RelayInfo, 0, len(m.relays))
	for _, relay := range m.relays {
		relayCopy := *relay
		relays = append(relays, &relayCopy)
	}
	return relays
}

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

func (m *Manager) GetRelayByID(relayID string) *RelayInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if relay, ok := m.relays[relayID]; ok {
		relayCopy := *relay
		return &relayCopy
	}
	return nil
}
