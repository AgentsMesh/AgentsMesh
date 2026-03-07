package runner

import (
	"sync"
	"time"
)

// RelayConnectionInfo represents a single relay connection for a pod
type RelayConnectionInfo struct {
	PodKey      string    `json:"pod_key"`
	RelayURL    string    `json:"relay_url"`
	SessionID   string    `json:"session_id"`
	Connected   bool      `json:"connected"`
	ConnectedAt time.Time `json:"connected_at"`
}

// RelayConnectionCache caches relay connection status per runner.
// Data is refreshed on each heartbeat (every 30s).
// This is stored in memory only - no persistence needed.
type RelayConnectionCache struct {
	mu    sync.RWMutex
	cache map[int64][]RelayConnectionInfo // runnerID -> connections
}

// NewRelayConnectionCache creates a new relay connection cache
func NewRelayConnectionCache() *RelayConnectionCache {
	return &RelayConnectionCache{
		cache: make(map[int64][]RelayConnectionInfo),
	}
}

// Update updates relay connections for a runner (called on heartbeat)
func (c *RelayConnectionCache) Update(runnerID int64, connections []RelayConnectionInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(connections) == 0 {
		delete(c.cache, runnerID)
	} else {
		c.cache[runnerID] = connections
	}
}

// Get returns a copy of relay connections for a runner.
// Returns a copy to prevent external modification of internal state.
func (c *RelayConnectionCache) Get(runnerID int64) []RelayConnectionInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	src := c.cache[runnerID]
	if src == nil {
		return nil
	}
	result := make([]RelayConnectionInfo, len(src))
	copy(result, src)
	return result
}

// Delete removes relay connections for a runner (called when runner disconnects)
func (c *RelayConnectionCache) Delete(runnerID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, runnerID)
}

// Count returns the total number of runners with relay connections
func (c *RelayConnectionCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// TotalConnections returns the total number of relay connections across all runners
func (c *RelayConnectionCache) TotalConnections() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	total := 0
	for _, conns := range c.cache {
		total += len(conns)
	}
	return total
}
