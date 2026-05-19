package relay

import "sort"

const (
	strictCPUThreshold = 80
	strictMemThreshold = 80
)

// selectFromCandidatesLocked returns a copy (not a pointer to m.relays storage) so callers
// can read without holding the lock. Caller must hold m.mu.RLock.
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
	sortRelayPriorities(priorities)

	for _, p := range priorities {
		if r, ok := m.relays[p.id]; ok {
			relayCopy := *r
			return &relayCopy
		}
	}
	return nil
}

type relayPriority struct {
	id       string
	priority uint32
}

func sortRelayPriorities(priorities []relayPriority) {
	sort.Slice(priorities, func(i, j int) bool {
		if priorities[i].priority != priorities[j].priority {
			return priorities[i].priority < priorities[j].priority
		}
		return priorities[i].id < priorities[j].id
	})
}

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

func isRelayReachable(r *RelayInfo) bool {
	if !r.Healthy {
		return false
	}
	if r.Capacity > 0 && r.CurrentConnections >= r.Capacity {
		return false
	}
	return true
}

func isRelayAvailable(r *RelayInfo) bool {
	return isRelayReachable(r) && r.CPUUsage <= strictCPUThreshold && r.MemoryUsage <= strictMemThreshold
}
