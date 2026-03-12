package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRemoveConnection_GenerationMismatch verifies that a stale defer call
// (from an old connection) does not remove a newer connection.
func TestRemoveConnection_GenerationMismatch(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream1 := newMockRunnerStream()
	stream2 := newMockRunnerStream()
	defer stream1.Close()
	defer stream2.Close()

	disconnected := false
	cm.SetDisconnectCallback(func(runnerID int64) {
		disconnected = true
	})

	// Simulate: old connection established
	oldConn := cm.AddConnection(1, "node", "org", stream1)
	oldGen := oldConn.Generation

	// Simulate: new connection replaces old one (e.g., runner reconnected)
	newConn := cm.AddConnection(1, "node", "org", stream2)
	assert.NotEqual(t, oldGen, newConn.Generation, "new connection should have different generation")
	assert.True(t, oldConn.IsClosed(), "old connection should be closed by AddConnection")
	assert.Equal(t, int64(1), cm.ConnectionCount())

	// Simulate: old connection's defer fires with stale generation
	cm.RemoveConnection(1, oldGen)

	// New connection must survive
	assert.Equal(t, int64(1), cm.ConnectionCount(), "new connection should not be removed")
	stored := cm.GetConnection(1)
	assert.Same(t, newConn, stored, "should still be the new connection")
	assert.False(t, disconnected, "disconnect callback should NOT fire for generation mismatch")
}

// TestRemoveConnection_MatchingGeneration verifies that removal works
// when the generation matches (normal disconnect scenario).
func TestRemoveConnection_MatchingGeneration(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream := newMockRunnerStream()
	defer stream.Close()

	disconnected := false
	cm.SetDisconnectCallback(func(runnerID int64) {
		disconnected = true
	})

	conn := cm.AddConnection(1, "node", "org", stream)
	cm.RemoveConnection(1, conn.Generation)

	assert.Equal(t, int64(0), cm.ConnectionCount())
	assert.Nil(t, cm.GetConnection(1))
	assert.True(t, disconnected, "disconnect callback should fire for matching generation")
}

// TestRemoveConnection_NonExistentRunner verifies no-op for unknown runner.
func TestRemoveConnection_NonExistentRunner(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	disconnected := false
	cm.SetDisconnectCallback(func(runnerID int64) {
		disconnected = true
	})

	// Should not panic or call disconnect callback
	cm.RemoveConnection(999, 1)
	assert.False(t, disconnected)
}

// TestConnectionGeneration_MonotonicallyIncreasing verifies that generation IDs
// are unique and monotonically increasing across connections.
func TestConnectionGeneration_MonotonicallyIncreasing(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	streams := make([]*MockRunnerStream, 5)
	for i := range streams {
		streams[i] = newMockRunnerStream()
		defer streams[i].Close()
	}

	var generations []int64
	for i, s := range streams {
		conn := cm.AddConnection(int64(i+1), "node", "org", s)
		generations = append(generations, conn.Generation)
	}

	for i := 1; i < len(generations); i++ {
		assert.Greater(t, generations[i], generations[i-1],
			"generation IDs should be monotonically increasing")
	}
}

// TestOnlineEventSent verifies the per-connection deduplication flag.
func TestOnlineEventSent(t *testing.T) {
	stream := newMockRunnerStream()
	defer stream.Close()

	conn := NewGRPCConnection(1, 1, "node", "org", stream)

	assert.False(t, conn.IsOnlineEventSent(), "should be false initially")

	conn.MarkOnlineEventSent()
	assert.True(t, conn.IsOnlineEventSent(), "should be true after marking")

	// Verify a new connection starts fresh
	conn2 := NewGRPCConnection(1, 2, "node", "org", stream)
	assert.False(t, conn2.IsOnlineEventSent(), "new connection should start with false")
}
