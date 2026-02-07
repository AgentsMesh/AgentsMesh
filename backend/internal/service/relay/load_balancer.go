package relay

import (
	"hash/fnv"
	"sort"
)

// SelectRelayOptions holds options for relay selection
type SelectRelayOptions struct {
	Region          string   // Preferred region
	ExcludeRelayIDs []string // Relays to exclude from selection
}

// SelectRelay selects the best relay for a given runner region using weighted scoring
// Factors considered: connection load, CPU, memory, latency, and region affinity
func (m *Manager) SelectRelay(runnerRegion string) *RelayInfo {
	return m.SelectRelayWithOptions(SelectRelayOptions{Region: runnerRegion})
}

// SelectRelayWithOptions selects the best relay with additional options
func (m *Manager) SelectRelayWithOptions(opts SelectRelayOptions) *RelayInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Build exclusion set for O(1) lookup
	excluded := make(map[string]bool)
	for _, id := range opts.ExcludeRelayIDs {
		excluded[id] = true
	}

	var best *RelayInfo
	bestScore := float64(-1)
	cfg := m.loadBalancingConfig

	for _, relay := range m.relays {
		// Skip unhealthy, excluded, or at-capacity relays
		if !relay.Healthy || excluded[relay.ID] {
			continue
		}
		if relay.Capacity > 0 && relay.CurrentConnections >= relay.Capacity {
			continue
		}

		// Calculate weighted score (higher is better)
		score := m.calculateRelayScore(relay, opts.Region, cfg)

		if score > bestScore {
			bestScore = score
			best = relay
		}
	}

	if best != nil {
		m.logger.Debug("Selected relay",
			"relay_id", best.ID,
			"region", best.Region,
			"score", bestScore,
			"connections", best.CurrentConnections,
			"capacity", best.Capacity,
			"cpu", best.CPUUsage,
			"memory", best.MemoryUsage,
			"latency_ms", best.AvgLatencyMs)

		// Return copy to prevent data races
		bestCopy := *best
		return &bestCopy
	}

	return nil
}

// calculateRelayScore calculates the weighted score for relay selection
func (m *Manager) calculateRelayScore(relay *RelayInfo, region string, cfg LoadBalancingConfig) float64 {
	score := 100.0

	// Connection load factor: prefer relays with more available capacity
	if relay.Capacity > 0 {
		availableRatio := 1 - float64(relay.CurrentConnections)/float64(relay.Capacity)
		score += availableRatio * cfg.ConnectionWeight * 100
	}

	// CPU factor: prefer relays with lower CPU usage
	if relay.CPUUsage >= 0 {
		cpuAvailable := (100 - relay.CPUUsage) / 100
		score += cpuAvailable * cfg.CPUWeight * 100
	}

	// Memory factor: prefer relays with lower memory usage
	if relay.MemoryUsage >= 0 {
		memAvailable := (100 - relay.MemoryUsage) / 100
		score += memAvailable * cfg.MemoryWeight * 100
	}

	// Latency factor: prefer relays with lower latency
	// Normalize latency: 0ms = 1.0, 500ms = 0.0 (capped)
	if relay.AvgLatencyMs >= 0 {
		latencyFactor := 1 - float64(min(relay.AvgLatencyMs, 500))/500
		score += latencyFactor * cfg.LatencyWeight * 100
	}

	// Region affinity bonus
	if region != "" && relay.Region == region {
		score += cfg.RegionBonus
	}

	return score
}

// SelectRelayForPod selects relay for a pod using org-affinity based selection
// Returns a copy to prevent data races
func (m *Manager) SelectRelayForPod(orgSlug string) *RelayInfo {
	return m.SelectRelayWithAffinity(orgSlug)
}

// SelectRelayWithAffinity selects relay using open addressing affinity algorithm
// Same organization will consistently select the same healthy relay
func (m *Manager) SelectRelayWithAffinity(orgSlug string) *RelayInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.relays) == 0 {
		return nil
	}

	// 1. Collect all relay IDs and sort them (as basis for probe sequence)
	relayIDs := make([]string, 0, len(m.relays))
	for id := range m.relays {
		relayIDs = append(relayIDs, id)
	}
	sort.Strings(relayIDs)

	// 2. Generate probe sequence based on orgSlug (open addressing)
	// For each relay ID, compute hash(orgSlug + relayID) as priority
	type relayPriority struct {
		id       string
		priority uint32
	}
	priorities := make([]relayPriority, len(relayIDs))
	for i, id := range relayIDs {
		// hash(orgSlug + relayID) ensures:
		// - Same org has fixed priority for same relay
		// - Different orgs have different priority orders (load distribution)
		priorities[i] = relayPriority{
			id:       id,
			priority: hashString(orgSlug + id),
		}
	}

	// Sort by priority to get probe sequence for this org
	sort.Slice(priorities, func(i, j int) bool {
		return priorities[i].priority < priorities[j].priority
	})

	// 3. Find first available relay in probe sequence
	for _, p := range priorities {
		relay := m.relays[p.id]

		if !relay.Healthy {
			continue
		}
		if relay.Capacity > 0 && relay.CurrentConnections >= relay.Capacity {
			continue
		}
		// Skip overloaded relays (CPU > 80% or Memory > 80%)
		if relay.CPUUsage > 80 || relay.MemoryUsage > 80 {
			continue
		}

		m.logger.Debug("Selected relay with org affinity",
			"relay_id", relay.ID,
			"org_slug", orgSlug,
			"connections", relay.CurrentConnections,
			"capacity", relay.Capacity,
			"cpu", relay.CPUUsage,
			"memory", relay.MemoryUsage)

		// Return copy to prevent data races
		relayCopy := *relay
		return &relayCopy
	}

	return nil
}

// hashString computes a 32-bit FNV-1a hash of the string
func hashString(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}
