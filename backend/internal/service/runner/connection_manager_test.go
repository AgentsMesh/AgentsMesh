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

// mockAgentTypesProvider implements interfaces.AgentTypesProvider for testing
type mockAgentTypesProvider struct{}

func (m *mockAgentTypesProvider) GetAgentTypesForRunner() []interfaces.AgentTypeInfo {
	return []interfaces.AgentTypeInfo{
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
	cm.SetTerminalOutputCallback(func(runnerID int64, data *runnerv1.TerminalOutputEvent) {})
	cm.SetAgentStatusCallback(func(runnerID int64, data *runnerv1.AgentStatusEvent) {})
	cm.SetPtyResizedCallback(func(runnerID int64, data *runnerv1.PtyResizedEvent) {})
	cm.SetInitializedCallback(func(runnerID int64, availableAgents []string) {})

	// Test provider and version setters
	cm.SetAgentTypesProvider(&mockAgentTypesProvider{})
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
	cm.AddConnection(1, "test-node", "test-org", stream)
	assert.Equal(t, int64(1), cm.ConnectionCount())

	cm.RemoveConnection(1)
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

	cm.AddConnection(1, "test-node", "test-org", stream)
	assert.True(t, cm.IsConnected(1))

	cm.RemoveConnection(1)
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
				cm.AddConnection(runnerID, "node", "org", stream)
				cm.IsConnected(runnerID)
				cm.GetConnection(runnerID)
				cm.UpdateHeartbeat(runnerID)
				cm.RemoveConnection(runnerID)
				stream.Close()
			}
		}(i)
	}

	wg.Wait()
	// Verify no race conditions or deadlocks occurred
	assert.Equal(t, int64(0), cm.ConnectionCount())
}

func TestConnectionManager_HandleHeartbeat(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream := newMockRunnerStream()
	defer stream.Close()

	// Add connection
	conn := cm.AddConnection(1, "test-node", "test-org", stream)
	initialPing := conn.GetLastPing()

	// Track callback invocation
	var callbackRunnerID int64
	var callbackData *runnerv1.HeartbeatData
	cm.SetHeartbeatCallback(func(runnerID int64, data *runnerv1.HeartbeatData) {
		callbackRunnerID = runnerID
		callbackData = data
	})

	time.Sleep(10 * time.Millisecond)

	// Handle heartbeat
	heartbeatData := &runnerv1.HeartbeatData{
		NodeId: "test-node",
	}
	cm.HandleHeartbeat(1, heartbeatData)

	// Verify last ping was updated
	assert.True(t, conn.GetLastPing().After(initialPing))

	// Verify callback was called
	assert.Equal(t, int64(1), callbackRunnerID)
	assert.Equal(t, heartbeatData, callbackData)
}

func TestConnectionManager_HandleHeartbeat_NoCallback(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream := newMockRunnerStream()
	defer stream.Close()

	cm.AddConnection(1, "test-node", "test-org", stream)

	// Should not panic when no callback is set
	cm.HandleHeartbeat(1, &runnerv1.HeartbeatData{NodeId: "test-node"})
}

func TestConnectionManager_HandlePodCreated(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream := newMockRunnerStream()
	defer stream.Close()

	cm.AddConnection(1, "test-node", "test-org", stream)

	var callbackRunnerID int64
	var callbackData *runnerv1.PodCreatedEvent
	cm.SetPodCreatedCallback(func(runnerID int64, data *runnerv1.PodCreatedEvent) {
		callbackRunnerID = runnerID
		callbackData = data
	})

	event := &runnerv1.PodCreatedEvent{
		PodKey: "test-pod",
	}
	cm.HandlePodCreated(1, event)

	assert.Equal(t, int64(1), callbackRunnerID)
	assert.Equal(t, event, callbackData)
}

func TestConnectionManager_HandlePodTerminated(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream := newMockRunnerStream()
	defer stream.Close()

	cm.AddConnection(1, "test-node", "test-org", stream)

	var callbackRunnerID int64
	var callbackData *runnerv1.PodTerminatedEvent
	cm.SetPodTerminatedCallback(func(runnerID int64, data *runnerv1.PodTerminatedEvent) {
		callbackRunnerID = runnerID
		callbackData = data
	})

	event := &runnerv1.PodTerminatedEvent{
		PodKey:   "test-pod",
		ExitCode: 0,
	}
	cm.HandlePodTerminated(1, event)

	assert.Equal(t, int64(1), callbackRunnerID)
	assert.Equal(t, event, callbackData)
}

func TestConnectionManager_HandleTerminalOutput(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream := newMockRunnerStream()
	defer stream.Close()

	cm.AddConnection(1, "test-node", "test-org", stream)

	var callbackRunnerID int64
	var callbackData *runnerv1.TerminalOutputEvent
	cm.SetTerminalOutputCallback(func(runnerID int64, data *runnerv1.TerminalOutputEvent) {
		callbackRunnerID = runnerID
		callbackData = data
	})

	event := &runnerv1.TerminalOutputEvent{
		PodKey: "test-pod",
		Data:   []byte("hello world"),
	}
	cm.HandleTerminalOutput(1, event)

	assert.Equal(t, int64(1), callbackRunnerID)
	assert.Equal(t, event, callbackData)
}

func TestConnectionManager_HandleAgentStatus(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream := newMockRunnerStream()
	defer stream.Close()

	cm.AddConnection(1, "test-node", "test-org", stream)

	var callbackRunnerID int64
	var callbackData *runnerv1.AgentStatusEvent
	cm.SetAgentStatusCallback(func(runnerID int64, data *runnerv1.AgentStatusEvent) {
		callbackRunnerID = runnerID
		callbackData = data
	})

	event := &runnerv1.AgentStatusEvent{
		PodKey: "test-pod",
		Status: "running",
	}
	cm.HandleAgentStatus(1, event)

	assert.Equal(t, int64(1), callbackRunnerID)
	assert.Equal(t, event, callbackData)
}

func TestConnectionManager_HandlePtyResized(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream := newMockRunnerStream()
	defer stream.Close()

	cm.AddConnection(1, "test-node", "test-org", stream)

	var callbackRunnerID int64
	var callbackData *runnerv1.PtyResizedEvent
	cm.SetPtyResizedCallback(func(runnerID int64, data *runnerv1.PtyResizedEvent) {
		callbackRunnerID = runnerID
		callbackData = data
	})

	event := &runnerv1.PtyResizedEvent{
		PodKey: "test-pod",
		Cols:   120,
		Rows:   40,
	}
	cm.HandlePtyResized(1, event)

	assert.Equal(t, int64(1), callbackRunnerID)
	assert.Equal(t, event, callbackData)
}

func TestConnectionManager_SetInitFailedCallback(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	var callbackRunnerID int64
	var callbackReason string
	cm.SetInitFailedCallback(func(runnerID int64, reason string) {
		callbackRunnerID = runnerID
		callbackReason = reason
	})

	// Verify the callback is set (internal field)
	assert.NotNil(t, cm.onInitFailed)

	// Call the callback directly to verify it works
	cm.onInitFailed(1, "timeout")
	assert.Equal(t, int64(1), callbackRunnerID)
	assert.Equal(t, "timeout", callbackReason)
}

func TestConnectionManager_SetInitTimeout(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	// Default timeout
	assert.Equal(t, DefaultInitTimeout, cm.initTimeout)

	// Set custom timeout
	cm.SetInitTimeout(60 * time.Second)
	assert.Equal(t, 60*time.Second, cm.initTimeout)
}

func TestConnectionManager_SetPingInterval(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	// Default interval
	assert.Equal(t, 30*time.Second, cm.pingInterval)

	// Set custom interval
	cm.SetPingInterval(10 * time.Second)
	assert.Equal(t, 10*time.Second, cm.pingInterval)
}

func TestConnectionManager_StartInitTimeoutChecker(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	// Just verify it starts without error
	cm.StartInitTimeoutChecker()

	// Wait a bit and verify it's running (by closing and ensuring no panic)
	time.Sleep(20 * time.Millisecond)
}

func TestConnectionManager_CheckInitTimeouts(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	// Set a very short timeout for testing
	cm.SetInitTimeout(1 * time.Millisecond)

	stream := newMockRunnerStream()
	defer stream.Close()

	// Track init failed callback
	var failedRunnerID int64
	var failedReason string
	cm.SetInitFailedCallback(func(runnerID int64, reason string) {
		failedRunnerID = runnerID
		failedReason = reason
	})

	// Add connection (not initialized)
	cm.AddConnection(1, "test-node", "test-org", stream)
	assert.Equal(t, int64(1), cm.ConnectionCount())

	// Wait for timeout to expire
	time.Sleep(10 * time.Millisecond)

	// Manually trigger check
	cm.checkInitTimeouts()

	// Connection should be removed due to timeout
	assert.Equal(t, int64(0), cm.ConnectionCount())
	assert.Equal(t, int64(1), failedRunnerID)
	assert.Contains(t, failedReason, "timeout")
}

func TestConnectionManager_CheckInitTimeouts_InitializedConnection(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	// Set a very short timeout
	cm.SetInitTimeout(1 * time.Millisecond)

	stream := newMockRunnerStream()
	defer stream.Close()

	// Add and initialize connection
	conn := cm.AddConnection(1, "test-node", "test-org", stream)
	conn.SetInitialized(true, []string{"claude-code"})

	// Wait for timeout
	time.Sleep(10 * time.Millisecond)

	// Trigger check
	cm.checkInitTimeouts()

	// Connection should NOT be removed (it's initialized)
	assert.Equal(t, int64(1), cm.ConnectionCount())
}

func TestConnectionManager_InitTimeoutLoop(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())

	// Set a very short timeout
	cm.SetInitTimeout(5 * time.Millisecond)

	stream := newMockRunnerStream()
	defer stream.Close()

	// Track init failed callback
	var failedRunnerID int64
	cm.SetInitFailedCallback(func(runnerID int64, reason string) {
		failedRunnerID = runnerID
	})

	// Add connection (not initialized)
	cm.AddConnection(1, "test-node", "test-org", stream)

	// Start the loop
	cm.StartInitTimeoutChecker()

	// Wait long enough for at least one check cycle (loop uses 10 second ticker normally)
	// But we'll just manually verify the check works
	time.Sleep(15 * time.Millisecond)
	cm.checkInitTimeouts()

	// Close should stop the loop
	cm.Close()

	// Connection should be removed
	assert.Equal(t, int64(0), cm.ConnectionCount())
	assert.Equal(t, int64(1), failedRunnerID)
}
