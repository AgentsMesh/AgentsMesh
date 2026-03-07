package relay

import "sort"

// Availability thresholds for relay selection.
const (
	// strictCPUThreshold is the maximum CPU usage (%) for strict availability.
	strictCPUThreshold = 80
	// strictMemThreshold is the maximum memory usage (%) for strict availability.
	strictMemThreshold = 80
)

// selectFromCandidatesLocked applies org-affinity hash among a set of relay IDs.
// Returns a copy of the selected relay to prevent data races.
// Caller must hold m.mu.RLock.
func (m *Manager) selectFromCandidatesLocked(orgSlug string, candidateIDs []string) *RelayInfo {
	if len(candidateIDs) == 0 {
		return nil
	}

	priorities := make([]relayPriority, len(candidateIDs))
	for i, id := range candidateIDs {
		priorities[i] = relayPriority{
			id:       id,
			priority: hashStringPair(orgSlug, id),
		}
	}
	// sort.Slice on (priority, id) is deterministic regardless of input order;
	// no need to pre-sort candidateIDs.
	sortRelayPriorities(priorities)

	// Return first found relay (should all exist since pre-filtered from m.relays)
	for _, p := range priorities {
		if r, ok := m.relays[p.id]; ok {
			relayCopy := *r
			return &relayCopy
		}
	}
	return nil
}

// relayPriority pairs a relay ID with its rendezvous hash priority.
type relayPriority struct {
	id       string
	priority uint32
}

// sortRelayPriorities sorts by ascending priority with ID tie-breaker.
// Deterministic regardless of input order.
func sortRelayPriorities(priorities []relayPriority) {
	sort.Slice(priorities, func(i, j int) bool {
		if priorities[i].priority != priorities[j].priority {
			return priorities[i].priority < priorities[j].priority
		}
		return priorities[i].id < priorities[j].id
	})
}

// hashString computes a 32-bit FNV-1a hash of the string.
// Uses manual loop to avoid heap allocations (no fnv.New32a or []byte conversion).
func hashString(s string) uint32 {
	const (
		offset32 = uint32(2166136261)
		prime32  = uint32(16777619)
	)
	h := offset32
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime32
	}
	return h
}

// hashStringPair computes FNV-1a of two strings without allocating a concatenated copy.
// Produces the same result as hashString(a + b) since FNV-1a is a streaming hash.
func hashStringPair(a, b string) uint32 {
	const (
		offset32 = uint32(2166136261)
		prime32  = uint32(16777619)
	)
	h := offset32
	for i := 0; i < len(a); i++ {
		h ^= uint32(a[i])
		h *= prime32
	}
	for i := 0; i < len(b); i++ {
		h ^= uint32(b[i])
		h *= prime32
	}
	return h
}

// isRelayReachable is a lenient check: only requires healthy + not at hard capacity.
// Used as fallback when all relays fail the strict isRelayAvailable check
// (e.g., all CPUs above 80%). A high-load relay is better than no relay.
func isRelayReachable(r *RelayInfo) bool {
	if !r.Healthy {
		return false
	}
	if r.Capacity > 0 && r.CurrentConnections >= r.Capacity {
		return false
	}
	return true
}

// isRelayAvailable checks if a relay is eligible for selection.
// Extends isRelayReachable with strict CPU and memory thresholds.
func isRelayAvailable(r *RelayInfo) bool {
	return isRelayReachable(r) && r.CPUUsage <= strictCPUThreshold && r.MemoryUsage <= strictMemThreshold
}
