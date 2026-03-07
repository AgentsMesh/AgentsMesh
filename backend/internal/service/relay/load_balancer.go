package relay

// SelectRelayWithAffinity selects relay using rendezvous hashing (HRW) affinity.
// Same organization will consistently select the same healthy relay.
// Falls back to lenient checks when all relays fail strict availability thresholds.
func (m *Manager) SelectRelayWithAffinity(orgSlug string) *RelayInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.relays) == 0 {
		m.logger.Warn("No relays registered", "org_slug", orgSlug)
		return nil
	}

	// Single-pass: collect strict and lenient candidates simultaneously
	strictIDs := make([]string, 0, len(m.relays))
	lenientIDs := make([]string, 0, len(m.relays))
	for id, r := range m.relays {
		if isRelayAvailable(r) {
			strictIDs = append(strictIDs, id)
		} else if isRelayReachable(r) {
			lenientIDs = append(lenientIDs, id)
		}
	}

	// Prefer strict candidates; fall back to lenient if all relays are overloaded
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
