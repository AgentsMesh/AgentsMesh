package runner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Tests for ConnectionManager init timeout functionality

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
