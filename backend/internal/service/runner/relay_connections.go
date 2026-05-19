package runner

import (
	"sync"
	"time"
)

type RelayConnectionInfo struct {
	PodKey      string    `json:"pod_key"`
	RelayURL    string    `json:"relay_url"`
	SessionID   string    `json:"session_id"`
	Connected   bool      `json:"connected"`
	ConnectedAt time.Time `json:"connected_at"`
}

type RelayConnectionCache struct {
	mu    sync.RWMutex
	cache map[int64][]RelayConnectionInfo
}

func NewRelayConnectionCache() *RelayConnectionCache {
	return &RelayConnectionCache{
		cache: make(map[int64][]RelayConnectionInfo),
	}
}

func (c *RelayConnectionCache) Update(runnerID int64, connections []RelayConnectionInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(connections) == 0 {
		delete(c.cache, runnerID)
	} else {
		c.cache[runnerID] = connections
	}
}

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

func (c *RelayConnectionCache) Delete(runnerID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, runnerID)
}

func (c *RelayConnectionCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

func (c *RelayConnectionCache) TotalConnections() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	total := 0
	for _, conns := range c.cache {
		total += len(conns)
	}
	return total
}
