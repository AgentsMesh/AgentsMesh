package runner

import (
	"sync"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/interfaces"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
)

// newTestLogger and newMockRunnerStream are defined in test_helper_test.go

// mockAgentsProvider implements interfaces.AgentsProvider for testing
type mockAgentsProvider struct{}

func (m *mockAgentsProvider) GetAgentsForRunner() []interfaces.AgentInfo {
	return []interfaces.AgentInfo{
		{Slug: "claude-code", Name: "Claude Code", Executable: "claude", LaunchCommand: "claude --model sonnet"},
	}
}

func TestNewRunnerConnectionManager(t *testing.T) {
	logger := newTestLogger()
	cm := NewRunnerConnectionManager(logger)
	defer cm.Close()

	assert.NotNil(t, cm)
	assert.Equal(t, 30*time.Second, cm.pingInterval)
	assert.Equal(t, DefaultInitTimeout, cm.initTimeout)
	assert.Equal(t, int64(0), cm.ConnectionCount())

	// Verify all shards are initialized
	for i := 0; i < numShards; i++ {
		assert.NotNil(t, cm.shards[i])
		assert.NotNil(t, cm.shards[i].connections)
	}
}

func TestConnectionManager_GetShard(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	// Test that same runner ID always maps to same shard
	shard1 := cm.getShard(100)
	shard2 := cm.getShard(100)
	assert.Same(t, shard1, shard2)

	// Test that different runner IDs may map to different shards
	// (not guaranteed but likely for sufficiently different IDs)
	shardA := cm.getShard(1)
	shardB := cm.getShard(256 + 1) // Should map to same shard as 1
	assert.Same(t, shardA, shardB)

	// Test negative runner ID handling (should work via unsigned conversion)
	shardNeg := cm.getShard(-1)
	assert.NotNil(t, shardNeg)
}

func TestConnectionManager_CallbackSetters(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	// Test SetHeartbeatCallback (using Proto type)
	cm.SetHeartbeatCallback(func(runnerID int64, data *runnerv1.HeartbeatData) {})
	assert.NotNil(t, cm.GetHeartbeatCallback())

	// Test SetDisconnectCallback
	cm.SetDisconnectCallback(func(runnerID int64) {})
	assert.NotNil(t, cm.GetDisconnectCallback())

	// Test other callbacks (using Proto types)
	cm.SetPodCreatedCallback(func(runnerID int64, data *runnerv1.PodCreatedEvent) {})
	cm.SetPodTerminatedCallback(func(runnerID int64, data *runnerv1.PodTerminatedEvent) {})
	cm.SetAgentStatusCallback(func(runnerID int64, data *runnerv1.AgentStatusEvent) {})
	cm.SetInitializedCallback(func(runnerID int64, availableAgents []string) {})

	// Test provider and version setters
	cm.SetAgentsProvider(&mockAgentsProvider{})
	cm.SetServerVersion("1.0.0")
	assert.Equal(t, "1.0.0", cm.serverVersion)
}

func TestConnectionManager_AddConnection(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream := newMockRunnerStream()
	defer stream.Close()

	// Add connection
	conn := cm.AddConnection(1, "test-node", "test-org", stream)
	assert.NotNil(t, conn)
	assert.Equal(t, int64(1), conn.RunnerID)
	assert.Equal(t, "test-node", conn.NodeID)
	assert.Equal(t, "test-org", conn.OrgSlug)
	assert.NotNil(t, conn.Send)
	assert.Equal(t, int64(1), cm.ConnectionCount())

	// Verify connection is stored
	stored := cm.GetConnection(1)
	assert.Same(t, conn, stored)
}

func TestConnectionManager_AddConnection_ReplacesExisting(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream1 := newMockRunnerStream()
	stream2 := newMockRunnerStream()
	defer stream1.Close()
	defer stream2.Close()

	// Add first connection
	conn1 := cm.AddConnection(1, "node-1", "org-1", stream1)
	assert.Equal(t, int64(1), cm.ConnectionCount())

	// Add second connection with same runner ID
	conn2 := cm.AddConnection(1, "node-1", "org-1", stream2)
	assert.Equal(t, int64(1), cm.ConnectionCount())

	// Verify old connection was closed and new one is stored
	assert.True(t, conn1.IsClosed())
	stored := cm.GetConnection(1)
	assert.Same(t, conn2, stored)
}

func TestConnectionManager_RemoveConnection(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream := newMockRunnerStream()
	defer stream.Close()

	disconnected := false
	cm.SetDisconnectCallback(func(runnerID int64) {
		disconnected = true
	})

	// Add and remove connection
	conn := cm.AddConnection(1, "test-node", "test-org", stream)
	assert.Equal(t, int64(1), cm.ConnectionCount())

	cm.RemoveConnection(1, conn.Generation)
	assert.Equal(t, int64(0), cm.ConnectionCount())
	assert.Nil(t, cm.GetConnection(1))
	assert.True(t, disconnected)
}

func TestConnectionManager_IsConnected(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream := newMockRunnerStream()
	defer stream.Close()

	assert.False(t, cm.IsConnected(1))

	conn := cm.AddConnection(1, "test-node", "test-org", stream)
	assert.True(t, cm.IsConnected(1))

	cm.RemoveConnection(1, conn.Generation)
	assert.False(t, cm.IsConnected(1))
}

func TestConnectionManager_GetConnectedRunnerIDs(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	streams := make([]*MockRunnerStream, 3)
	for i := range streams {
		streams[i] = newMockRunnerStream()
		defer streams[i].Close()
	}

	// Initially empty
	ids := cm.GetConnectedRunnerIDs()
	assert.Empty(t, ids)

	// Add multiple connections
	cm.AddConnection(1, "node-1", "org", streams[0])
	cm.AddConnection(2, "node-2", "org", streams[1])
	cm.AddConnection(3, "node-3", "org", streams[2])

	ids = cm.GetConnectedRunnerIDs()
	assert.Len(t, ids, 3)
	assert.Contains(t, ids, int64(1))
	assert.Contains(t, ids, int64(2))
	assert.Contains(t, ids, int64(3))
}

func TestConnectionManager_Close(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())

	streams := make([]*MockRunnerStream, 3)
	for i := range streams {
		streams[i] = newMockRunnerStream()
	}

	// Add connections
	conns := make([]*GRPCConnection, 3)
	for i, s := range streams {
		conns[i] = cm.AddConnection(int64(i+1), "node", "org", s)
	}

	// Close manager
	cm.Close()

	// Verify all connections are closed
	for _, conn := range conns {
		assert.True(t, conn.IsClosed())
	}
	assert.Equal(t, int64(0), cm.ConnectionCount())
}

func TestConnectionManager_UpdateHeartbeat(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream := newMockRunnerStream()
	defer stream.Close()

	conn := cm.AddConnection(1, "test-node", "test-org", stream)
	initialPing := conn.GetLastPing()

	time.Sleep(10 * time.Millisecond)
	cm.UpdateHeartbeat(1)

	assert.True(t, conn.GetLastPing().After(initialPing))
}

func TestConnectionManager_HandleInitialized(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream := newMockRunnerStream()
	defer stream.Close()

	// Add connection first
	conn := cm.AddConnection(1, "test-node", "test-org", stream)

	// Track callback invocation
	var callbackRunnerID int64
	var callbackAgents []string
	cm.SetInitializedCallback(func(runnerID int64, availableAgents []string) {
		callbackRunnerID = runnerID
		callbackAgents = availableAgents
	})

	// Handle initialized
	cm.HandleInitialized(1, []string{"claude-code", "aider"})

	// Verify connection is marked as initialized
	assert.True(t, conn.IsInitialized())
	assert.Equal(t, []string{"claude-code", "aider"}, conn.GetAvailableAgents())

	// Verify callback was called
	assert.Equal(t, int64(1), callbackRunnerID)
	assert.Equal(t, []string{"claude-code", "aider"}, callbackAgents)
}

func TestConnectionManager_ConcurrentOperations(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			runnerID := int64(id % 50) // Reuse runner IDs to test contention

			for j := 0; j < numOperations; j++ {
				stream := newMockRunnerStream()
				conn := cm.AddConnection(runnerID, "node", "org", stream)
				cm.IsConnected(runnerID)
				cm.GetConnection(runnerID)
				cm.UpdateHeartbeat(runnerID)
				cm.RemoveConnection(runnerID, conn.Generation)
				stream.Close()
			}
		}(i)
	}

	wg.Wait()
	// Verify no race conditions or deadlocks occurred
	assert.Equal(t, int64(0), cm.ConnectionCount())
}
