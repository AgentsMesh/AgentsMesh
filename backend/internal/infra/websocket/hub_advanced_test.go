package websocket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHubUnregisterNonExistentClient(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	// Try to unregister a client that was never registered
	client := mockClient(999, 1)
	hub.Unregister(client)
	time.Sleep(10 * time.Millisecond)

	// Should not panic, count should be 0
	assert.Equal(t, 0, hub.GetTotalClientCount())
}

func TestHubClientWithNoUserAndNoOrg(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	// Client with userID=0 and orgID=0 should use fallback shard
	client := mockClient(0, 0)
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 1, hub.GetTotalClientCount())

	hub.Unregister(client)
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 0, hub.GetTotalClientCount())
}

func TestHubClientWithOnlyOrgID(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	// Client with userID=0 but orgID set should use org-based sharding
	client := mockClient(0, 100)
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 1, hub.GetTotalClientCount())
}
