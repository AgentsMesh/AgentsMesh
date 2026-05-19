package relay

func (m *Manager) SelectRelayWithAffinity(orgSlug string) *RelayInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.relays) == 0 {
		m.logger.Warn("No relays registered", "org_slug", orgSlug)
		return nil
	}

	strictIDs := make([]string, 0, len(m.relays))
	lenientIDs := make([]string, 0, len(m.relays))
	for id, r := range m.relays {
		if isRelayAvailable(r) {
			strictIDs = append(strictIDs, id)
		} else if isRelayReachable(r) {
			lenientIDs = append(lenientIDs, id)
		}
	}

	ids := strictIDs
	if len(ids) == 0 {
		ids = lenientIDs
	}

	selected := m.selectFromCandidatesLocked(orgSlug, ids)
	if selected != nil {
		m.logger.Debug("Selected relay with org affinity",
			"relay_id", selected.ID,
			"org_slug", orgSlug,
			"connections", selected.CurrentConnections,
			"capacity", selected.Capacity,
			"cpu", selected.CPUUsage,
			"memory", selected.MemoryUsage)
	} else {
		m.logger.Warn("No suitable relay found",
			"org_slug", orgSlug,
			"total_relays", len(m.relays))
	}
	return selected
}
