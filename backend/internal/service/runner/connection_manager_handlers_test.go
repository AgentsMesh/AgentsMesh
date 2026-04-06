package runner

import (
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
)

// Tests for ConnectionManager event handlers

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

// NOTE: TestConnectionManager_HandlePtyOutput removed - output is exclusively streamed via Relay

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

func TestConnectionManager_HandlePodInitProgress(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	var called bool
	var receivedRunnerID int64
	var receivedData *runnerv1.PodInitProgressEvent
	cm.SetPodInitProgressCallback(func(runnerID int64, data *runnerv1.PodInitProgressEvent) {
		called = true
		receivedRunnerID = runnerID
		receivedData = data
	})

	event := &runnerv1.PodInitProgressEvent{
		PodKey:   "test-pod",
		Phase:    "pulling_image",
		Progress: 50,
		Message:  "Pulling container image...",
	}

	cm.HandlePodInitProgress(1, event)

	assert.True(t, called)
	assert.Equal(t, int64(1), receivedRunnerID)
	assert.Equal(t, "test-pod", receivedData.PodKey)
	assert.Equal(t, "pulling_image", receivedData.Phase)
	assert.Equal(t, int32(50), receivedData.Progress)
}

func TestConnectionManager_HandlePodInitProgress_NoCallback(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	// No callback set - should not panic
	event := &runnerv1.PodInitProgressEvent{
		PodKey:   "test-pod",
		Phase:    "init",
		Progress: 10,
	}

	// This should not panic
	cm.HandlePodInitProgress(1, event)
}

func TestConnectionManager_HandleSandboxesStatus(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream := newMockRunnerStream()
	defer stream.Close()

	conn := cm.AddConnection(1, "test-node", "test-org", stream)
	initialPing := conn.GetLastPing()

	var callbackRunnerID int64
	var callbackData *runnerv1.SandboxesStatusEvent
	cm.SetSandboxesStatusCallback(func(runnerID int64, data *runnerv1.SandboxesStatusEvent) {
		callbackRunnerID = runnerID
		callbackData = data
	})

	time.Sleep(10 * time.Millisecond)

	event := &runnerv1.SandboxesStatusEvent{
		RequestId: "req-123",
		Sandboxes: []*runnerv1.SandboxStatus{
			{PodKey: "pod-1", Exists: true},
			{PodKey: "pod-2", Exists: false},
		},
	}
	cm.HandleSandboxesStatus(1, event)

	// Verify last ping was updated (heartbeat)
	assert.True(t, conn.GetLastPing().After(initialPing))

	// Verify callback was called with correct data
	assert.Equal(t, int64(1), callbackRunnerID)
	assert.Equal(t, "req-123", callbackData.RequestId)
	assert.Len(t, callbackData.Sandboxes, 2)
	assert.Equal(t, "pod-1", callbackData.Sandboxes[0].PodKey)
	assert.True(t, callbackData.Sandboxes[0].Exists)
}

func TestConnectionManager_HandleSandboxesStatus_NoCallback(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream := newMockRunnerStream()
	defer stream.Close()

	cm.AddConnection(1, "test-node", "test-org", stream)

	// No callback set - should not panic
	event := &runnerv1.SandboxesStatusEvent{
		RequestId: "req-456",
		Sandboxes: []*runnerv1.SandboxStatus{
			{PodKey: "pod-1", Exists: true},
		},
	}
	cm.HandleSandboxesStatus(1, event)
}

func TestConnectionManager_HandleRequestRelayToken(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream := newMockRunnerStream()
	defer stream.Close()

	conn := cm.AddConnection(1, "test-node", "test-org", stream)
	initialPing := conn.GetLastPing()

	var callbackRunnerID int64
	var callbackData *runnerv1.RequestRelayTokenEvent
	cm.SetRequestRelayTokenCallback(func(runnerID int64, data *runnerv1.RequestRelayTokenEvent) {
		callbackRunnerID = runnerID
		callbackData = data
	})

	time.Sleep(10 * time.Millisecond)

	event := &runnerv1.RequestRelayTokenEvent{
		PodKey: "test-pod",
	}
	cm.HandleRequestRelayToken(1, event)

	// Verify last ping was updated (heartbeat)
	assert.True(t, conn.GetLastPing().After(initialPing))

	// Verify callback was called with correct data
	assert.Equal(t, int64(1), callbackRunnerID)
	assert.Equal(t, "test-pod", callbackData.PodKey)
}

func TestConnectionManager_HandleRequestRelayToken_NoCallback(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	// No callback set - should not panic
	event := &runnerv1.RequestRelayTokenEvent{
		PodKey: "test-pod",
	}
	cm.HandleRequestRelayToken(1, event)
}

func TestConnectionManager_HandlePodError(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream := newMockRunnerStream()
	defer stream.Close()

	conn := cm.AddConnection(1, "test-node", "test-org", stream)
	initialPing := conn.GetLastPing()

	var callbackRunnerID int64
	var callbackData *runnerv1.ErrorEvent
	cm.SetPodErrorCallback(func(runnerID int64, data *runnerv1.ErrorEvent) {
		callbackRunnerID = runnerID
		callbackData = data
	})

	time.Sleep(10 * time.Millisecond)

	event := &runnerv1.ErrorEvent{
		PodKey:  "test-pod",
		Code:    "GIT_AUTH_FAILED",
		Message: "authentication failed for repository",
	}
	cm.HandlePodError(1, event)

	// Verify last ping was updated (heartbeat)
	assert.True(t, conn.GetLastPing().After(initialPing))

	// Verify callback was called with correct data
	assert.Equal(t, int64(1), callbackRunnerID)
	assert.Equal(t, "test-pod", callbackData.PodKey)
	assert.Equal(t, "GIT_AUTH_FAILED", callbackData.Code)
	assert.Equal(t, "authentication failed for repository", callbackData.Message)
}

func TestConnectionManager_HandlePodError_NoCallback(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream := newMockRunnerStream()
	defer stream.Close()

	cm.AddConnection(1, "test-node", "test-org", stream)

	// No callback set - should not panic
	event := &runnerv1.ErrorEvent{
		PodKey:  "test-pod",
		Code:    "UNKNOWN",
		Message: "something went wrong",
	}
	cm.HandlePodError(1, event)
}
