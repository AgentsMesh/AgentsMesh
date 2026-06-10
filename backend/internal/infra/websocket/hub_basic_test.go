package websocket

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClient creates a test connect-events client with given parameters.
func mockClient(userID, orgID int64) *Client {
	return &Client{
		userID: userID,
		orgID:  orgID,
		send:   make(chan []byte, 256),
	}
}

func TestNewHub(t *testing.T) {
	hub := NewHub()
	require.NotNil(t, hub)

	// Verify all shards are initialized
	for i := 0; i < hubShards; i++ {
		assert.NotNil(t, hub.shards[i])
		assert.NotNil(t, hub.shards[i].clients)
		assert.NotNil(t, hub.shards[i].orgClients)
		assert.NotNil(t, hub.shards[i].userClients)
	}

	hub.Close()
}

func TestHubRegisterUnregister(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	client := mockClient(1, 1)

	hub.Register(client)
	time.Sleep(10 * time.Millisecond) // Wait for async registration

	assert.Equal(t, 1, hub.GetTotalClientCount())

	hub.Unregister(client)
	time.Sleep(10 * time.Millisecond) // Wait for async unregistration

	assert.Equal(t, 0, hub.GetTotalClientCount())
}

func TestHubGetClientCounts(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	client1 := mockClient(1, 100)
	client2 := mockClient(2, 100)
	client3 := mockClient(3, 200)

	hub.Register(client1)
	hub.Register(client2)
	hub.Register(client3)
	time.Sleep(20 * time.Millisecond)

	assert.Equal(t, 3, hub.GetTotalClientCount())
	assert.Equal(t, 2, hub.GetOrgClientCount(100))
	assert.Equal(t, 1, hub.GetOrgClientCount(200))
	assert.Equal(t, 1, hub.GetUserClientCount(1))
	assert.Equal(t, 1, hub.GetUserClientCount(2))
}

func TestHubConcurrentOperations(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	var wg sync.WaitGroup
	clientCount := 100

	// Concurrent registrations
	clients := make([]*Client, clientCount)
	for i := 0; i < clientCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			clients[idx] = mockClient(int64(idx), int64(idx%10))
			hub.Register(clients[idx])
		}(i)
	}
	wg.Wait()
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, clientCount, hub.GetTotalClientCount())

	// Concurrent broadcasts
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(orgID int64) {
			defer wg.Done()
			hub.BroadcastToOrg(orgID, []byte(`{"event":"concurrent"}`))
		}(int64(i))
	}
	wg.Wait()

	// Concurrent unregistrations
	for i := 0; i < clientCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			hub.Unregister(clients[idx])
		}(i)
	}
	wg.Wait()
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 0, hub.GetTotalClientCount())
}

func TestHubShardDistribution(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	// Create clients with different user IDs
	clients := make([]*Client, 256)
	for i := 0; i < 256; i++ {
		clients[i] = mockClient(int64(i), 1)
		hub.Register(clients[i])
	}
	time.Sleep(50 * time.Millisecond)

	// Verify clients are distributed across shards
	nonEmptyShards := 0
	for i := 0; i < hubShards; i++ {
		hub.shards[i].mu.RLock()
		if len(hub.shards[i].clients) > 0 {
			nonEmptyShards++
		}
		hub.shards[i].mu.RUnlock()
	}

	// With 256 clients and 64 shards, we expect good distribution
	assert.Greater(t, nonEmptyShards, 10, "clients should be distributed across multiple shards")
}

func TestHubShardedArchitecture(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	// Sharded hub automatically starts all shard goroutines in NewHub()
	assert.NotNil(t, hub.shards[0], "shards should be initialized")
}

func TestHubCloseCleanup(t *testing.T) {
	hub := NewHub()

	// Register clients
	for i := 0; i < 10; i++ {
		client := mockClient(int64(i), 1)
		hub.Register(client)
	}
	time.Sleep(20 * time.Millisecond)

	assert.Equal(t, 10, hub.GetTotalClientCount())

	// Close should clean up all clients
	hub.Close()

	// Give time for cleanup
	time.Sleep(50 * time.Millisecond)

	// Verify all shards are empty
	for i := 0; i < hubShards; i++ {
		hub.shards[i].mu.RLock()
		assert.Equal(t, 0, len(hub.shards[i].clients), "shard %d should be empty", i)
		hub.shards[i].mu.RUnlock()
	}
}
