package runner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRelayConnectionCache_NewCache(t *testing.T) {
	cache := NewRelayConnectionCache()
	assert.NotNil(t, cache)
	assert.Equal(t, 0, cache.Count())
	assert.Equal(t, 0, cache.TotalConnections())
}

func TestRelayConnectionCache_UpdateAndGet(t *testing.T) {
	cache := NewRelayConnectionCache()

	// Add connections for runner 1
	connections := []RelayConnectionInfo{
		{
			PodKey:      "pod-1",
			RelayURL:    "wss://relay.example.com",
			SessionID:   "session-1",
			Connected:   true,
			ConnectedAt: time.Now(),
		},
		{
			PodKey:      "pod-2",
			RelayURL:    "wss://relay.example.com",
			SessionID:   "session-2",
			Connected:   true,
			ConnectedAt: time.Now(),
		},
	}

	cache.Update(1, connections)

	// Verify
	result := cache.Get(1)
	assert.Len(t, result, 2)
	assert.Equal(t, "pod-1", result[0].PodKey)
	assert.Equal(t, "pod-2", result[1].PodKey)
	assert.Equal(t, 1, cache.Count())
	assert.Equal(t, 2, cache.TotalConnections())
}

func TestRelayConnectionCache_UpdateEmpty(t *testing.T) {
	cache := NewRelayConnectionCache()

	// Add connections
	connections := []RelayConnectionInfo{
		{PodKey: "pod-1"},
	}
	cache.Update(1, connections)
	assert.Equal(t, 1, cache.Count())

	// Update with empty slice should remove entry
	cache.Update(1, []RelayConnectionInfo{})
	assert.Equal(t, 0, cache.Count())
	assert.Nil(t, cache.Get(1))
}

func TestRelayConnectionCache_Delete(t *testing.T) {
	cache := NewRelayConnectionCache()

	// Add connections
	cache.Update(1, []RelayConnectionInfo{{PodKey: "pod-1"}})
	cache.Update(2, []RelayConnectionInfo{{PodKey: "pod-2"}})
	assert.Equal(t, 2, cache.Count())

	// Delete runner 1
	cache.Delete(1)
	assert.Equal(t, 1, cache.Count())
	assert.Nil(t, cache.Get(1))
	assert.NotNil(t, cache.Get(2))
}

func TestRelayConnectionCache_GetNonExistent(t *testing.T) {
	cache := NewRelayConnectionCache()
	result := cache.Get(999)
	assert.Nil(t, result)
}

func TestRelayConnectionCache_TotalConnections(t *testing.T) {
	cache := NewRelayConnectionCache()

	// Add connections for multiple runners
	cache.Update(1, []RelayConnectionInfo{
		{PodKey: "pod-1"},
		{PodKey: "pod-2"},
	})
	cache.Update(2, []RelayConnectionInfo{
		{PodKey: "pod-3"},
	})
	cache.Update(3, []RelayConnectionInfo{
		{PodKey: "pod-4"},
		{PodKey: "pod-5"},
		{PodKey: "pod-6"},
	})

	assert.Equal(t, 3, cache.Count())
	assert.Equal(t, 6, cache.TotalConnections())
}
