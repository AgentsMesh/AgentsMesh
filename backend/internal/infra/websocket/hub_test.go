package websocket

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClient creates a test client with given parameters
func mockClient(userID, orgID int64, podKey string, channelID int64, isEvents bool) *Client {
	return &Client{
		userID:    userID,
		orgID:     orgID,
		podKey:    podKey,
		channelID: channelID,
		isEvents:  isEvents,
		send:      make(chan []byte, 256),
	}
}

func TestNewHub(t *testing.T) {
	hub := NewHub()
	require.NotNil(t, hub)

	// Verify all shards are initialized
	for i := 0; i < hubShards; i++ {
		assert.NotNil(t, hub.shards[i])
		assert.NotNil(t, hub.shards[i].clients)
		assert.NotNil(t, hub.shards[i].podClients)
		assert.NotNil(t, hub.shards[i].channelClients)
		assert.NotNil(t, hub.shards[i].orgClients)
		assert.NotNil(t, hub.shards[i].userClients)
	}

	hub.Close()
}

func TestHubRegisterUnregister(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	client := mockClient(1, 1, "pod-1", 0, false)

	// Register
	hub.Register(client)
	time.Sleep(10 * time.Millisecond) // Wait for async registration

	assert.Equal(t, 1, hub.GetTotalClientCount())

	// Unregister
	hub.Unregister(client)
	time.Sleep(10 * time.Millisecond) // Wait for async unregistration

	assert.Equal(t, 0, hub.GetTotalClientCount())
}

func TestHubBroadcastToPod(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	// Create clients connected to same pod
	client1 := mockClient(1, 1, "pod-abc", 0, false)
	client2 := mockClient(2, 1, "pod-abc", 0, false)
	client3 := mockClient(3, 1, "pod-xyz", 0, false) // Different pod

	hub.Register(client1)
	hub.Register(client2)
	hub.Register(client3)
	time.Sleep(20 * time.Millisecond)

	// Broadcast to pod-abc
	msg := &Message{Type: "test", Data: []byte(`"hello"`)}
	hub.BroadcastToPod("pod-abc", msg)

	// client1 and client2 should receive
	select {
	case data := <-client1.send:
		assert.Contains(t, string(data), "test")
	case <-time.After(100 * time.Millisecond):
		t.Error("client1 didn't receive message")
	}

	select {
	case data := <-client2.send:
		assert.Contains(t, string(data), "test")
	case <-time.After(100 * time.Millisecond):
		t.Error("client2 didn't receive message")
	}

	// client3 should NOT receive
	select {
	case <-client3.send:
		t.Error("client3 should not receive message for different pod")
	case <-time.After(50 * time.Millisecond):
		// Expected
	}
}

func TestHubBroadcastToChannel(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	client1 := mockClient(1, 1, "", 100, false)
	client2 := mockClient(2, 1, "", 100, false)
	client3 := mockClient(3, 1, "", 200, false) // Different channel

	hub.Register(client1)
	hub.Register(client2)
	hub.Register(client3)
	time.Sleep(20 * time.Millisecond)

	msg := &Message{Type: "channel_msg", Data: []byte(`"hello"`)}
	hub.BroadcastToChannel(100, msg)

	// client1 and client2 should receive
	select {
	case <-client1.send:
	case <-time.After(100 * time.Millisecond):
		t.Error("client1 didn't receive message")
	}

	select {
	case <-client2.send:
	case <-time.After(100 * time.Millisecond):
		t.Error("client2 didn't receive message")
	}

	// client3 should NOT receive
	select {
	case <-client3.send:
		t.Error("client3 should not receive message")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestHubBroadcastToOrg(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	client1 := mockClient(1, 100, "", 0, true) // Events channel
	client2 := mockClient(2, 100, "", 0, true) // Events channel
	client3 := mockClient(3, 200, "", 0, true) // Different org

	hub.Register(client1)
	hub.Register(client2)
	hub.Register(client3)
	time.Sleep(20 * time.Millisecond)

	hub.BroadcastToOrg(100, []byte(`{"event":"test"}`))

	// client1 and client2 should receive
	select {
	case <-client1.send:
	case <-time.After(100 * time.Millisecond):
		t.Error("client1 didn't receive message")
	}

	select {
	case <-client2.send:
	case <-time.After(100 * time.Millisecond):
		t.Error("client2 didn't receive message")
	}

	// client3 should NOT receive
	select {
	case <-client3.send:
		t.Error("client3 should not receive message")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestHubSendToUser(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	client1 := mockClient(123, 1, "", 0, true) // Events for user 123
	client2 := mockClient(123, 1, "", 0, true) // Same user
	client3 := mockClient(456, 1, "", 0, true) // Different user

	hub.Register(client1)
	hub.Register(client2)
	hub.Register(client3)
	time.Sleep(20 * time.Millisecond)

	hub.SendToUser(123, []byte(`{"notification":"hello"}`))

	// client1 and client2 should receive
	select {
	case <-client1.send:
	case <-time.After(100 * time.Millisecond):
		t.Error("client1 didn't receive message")
	}

	select {
	case <-client2.send:
	case <-time.After(100 * time.Millisecond):
		t.Error("client2 didn't receive message")
	}

	// client3 should NOT receive
	select {
	case <-client3.send:
		t.Error("client3 should not receive message")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestHubGetClientCounts(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	// Register various clients
	client1 := mockClient(1, 100, "pod-1", 0, true)
	client2 := mockClient(2, 100, "pod-1", 0, true)
	client3 := mockClient(3, 200, "pod-2", 0, true)

	hub.Register(client1)
	hub.Register(client2)
	hub.Register(client3)
	time.Sleep(20 * time.Millisecond)

	assert.Equal(t, 3, hub.GetTotalClientCount())
	assert.Equal(t, 2, hub.GetOrgClientCount(100))
	assert.Equal(t, 1, hub.GetOrgClientCount(200))
	assert.Equal(t, 2, hub.GetPodClientCount("pod-1"))
	assert.Equal(t, 1, hub.GetPodClientCount("pod-2"))
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
			clients[idx] = mockClient(int64(idx), int64(idx%10), "", 0, true)
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
		clients[i] = mockClient(int64(i), 1, "", 0, true)
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

func TestHubBroadcastToOrgJSON(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	client := mockClient(1, 100, "", 0, true)
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	msg := map[string]interface{}{
		"type": "notification",
		"data": "hello",
	}
	err := hub.BroadcastToOrgJSON(100, msg)
	assert.NoError(t, err)

	select {
	case data := <-client.send:
		assert.Contains(t, string(data), "notification")
	case <-time.After(100 * time.Millisecond):
		t.Error("client didn't receive message")
	}
}

func TestHubSendToUserJSON(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	client := mockClient(123, 1, "", 0, true)
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	msg := map[string]interface{}{
		"type": "alert",
		"data": "important",
	}
	err := hub.SendToUserJSON(123, msg)
	assert.NoError(t, err)

	select {
	case data := <-client.send:
		assert.Contains(t, string(data), "alert")
	case <-time.After(100 * time.Millisecond):
		t.Error("client didn't receive message")
	}
}

func TestHubShardedArchitecture(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	// Sharded hub automatically starts all shard goroutines in NewHub()
	// No explicit Run() call needed - goroutines are already running
	assert.NotNil(t, hub.shards[0], "shards should be initialized")
}

func TestHubCloseCleanup(t *testing.T) {
	hub := NewHub()

	// Register clients
	for i := 0; i < 10; i++ {
		client := mockClient(int64(i), 1, "", 0, true)
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

func TestHubUnregisterNonExistentClient(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	// Try to unregister a client that was never registered
	client := mockClient(999, 1, "", 0, true)
	hub.Unregister(client)
	time.Sleep(10 * time.Millisecond)

	// Should not panic, count should be 0
	assert.Equal(t, 0, hub.GetTotalClientCount())
}

func TestHubUnregisterWithPodKey(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	// Register client with pod key
	client := mockClient(1, 1, "pod-test-123", 0, false)
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 1, hub.GetPodClientCount("pod-test-123"))

	// Unregister
	hub.Unregister(client)
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 0, hub.GetPodClientCount("pod-test-123"))
}

func TestHubUnregisterWithChannel(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	// Register client with channel
	client := mockClient(1, 1, "", 500, false)
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	// Unregister
	hub.Unregister(client)
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 0, hub.GetTotalClientCount())
}

func TestHubBroadcastToPodEmptyPod(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	// Broadcast to non-existent pod should not panic
	msg := &Message{Type: "test", Data: []byte(`"hello"`)}
	hub.BroadcastToPod("non-existent-pod", msg)
}

func TestHubBroadcastToChannelEmptyChannel(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	// Broadcast to non-existent channel should not panic
	msg := &Message{Type: "test", Data: []byte(`"hello"`)}
	hub.BroadcastToChannel(99999, msg)
}

func TestHubBroadcastToOrgEmptyOrg(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	// Broadcast to non-existent org should not panic
	hub.BroadcastToOrg(99999, []byte(`{"test":"data"}`))
}

func TestHubSendToUserEmptyUser(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	// Send to non-existent user should not panic
	hub.SendToUser(99999, []byte(`{"test":"data"}`))
}

func TestHubBroadcastToOrgJSONError(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	// Test with invalid JSON data (channel that cannot be serialized)
	invalidData := make(chan int)
	err := hub.BroadcastToOrgJSON(1, invalidData)
	assert.Error(t, err)
}

func TestHubSendToUserJSONError(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	// Test with invalid JSON data
	invalidData := make(chan int)
	err := hub.SendToUserJSON(1, invalidData)
	assert.Error(t, err)
}

func TestHubGetShardByPod(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	// Test pod key sharding
	shard1 := hub.getShardByPod("pod-abc-123")
	shard2 := hub.getShardByPod("pod-abc-123")
	shard3 := hub.getShardByPod("pod-xyz-456")

	// Same pod should always go to same shard
	assert.Equal(t, shard1, shard2)
	// Different pods may or may not go to different shards (depends on hash)
	_ = shard3 // Just verify it doesn't panic
}

func TestHubGetShardByOrg(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	shard := hub.getShardByOrg(100)
	assert.True(t, shard < hubShards)
}

func TestHubGetShardByChannel(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	shard := hub.getShardByChannel(200)
	assert.True(t, shard < hubShards)
}

func TestHubClientWithNoUserAndNoOrg(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	// Client with userID=0 and orgID=0 should use fallback shard
	client := mockClient(0, 0, "", 0, false)
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
	client := mockClient(0, 100, "", 0, false)
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 1, hub.GetTotalClientCount())
}
