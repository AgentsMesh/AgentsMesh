package runner

import (
	"testing"
	"time"

	runnerv1 "github.com/AgentsMesh/AgentsMesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
)

// Tests for ConnectionManager OSC event handlers

func TestConnectionManager_HandleOSCNotification(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream := newMockRunnerStream()
	defer stream.Close()

	// Add connection
	conn := cm.AddConnection(1, "test-node", "test-org", stream)
	initialPing := conn.GetLastPing()

	// Track callback invocation
	var callbackRunnerID int64
	var callbackData *runnerv1.OSCNotificationEvent
	cm.SetOSCNotificationCallback(func(runnerID int64, data *runnerv1.OSCNotificationEvent) {
		callbackRunnerID = runnerID
		callbackData = data
	})

	time.Sleep(10 * time.Millisecond)

	// Handle OSC notification
	event := &runnerv1.OSCNotificationEvent{
		PodKey:    "test-pod",
		Title:     "Build Complete",
		Body:      "Your project compiled successfully",
		Timestamp: time.Now().UnixMilli(),
	}
	cm.HandleOSCNotification(1, event)

	// Verify last ping was updated (heartbeat)
	assert.True(t, conn.GetLastPing().After(initialPing))

	// Verify callback was called
	assert.Equal(t, int64(1), callbackRunnerID)
	assert.Equal(t, "test-pod", callbackData.PodKey)
	assert.Equal(t, "Build Complete", callbackData.Title)
	assert.Equal(t, "Your project compiled successfully", callbackData.Body)
}

func TestConnectionManager_HandleOSCNotification_NoCallback(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream := newMockRunnerStream()
	defer stream.Close()

	cm.AddConnection(1, "test-node", "test-org", stream)

	// No callback set - should not panic
	event := &runnerv1.OSCNotificationEvent{
		PodKey: "test-pod",
		Title:  "Test",
		Body:   "Test body",
	}

	// This should not panic
	cm.HandleOSCNotification(1, event)
}

func TestConnectionManager_HandleOSCTitle(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream := newMockRunnerStream()
	defer stream.Close()

	// Add connection
	conn := cm.AddConnection(1, "test-node", "test-org", stream)
	initialPing := conn.GetLastPing()

	// Track callback invocation
	var callbackRunnerID int64
	var callbackData *runnerv1.OSCTitleEvent
	cm.SetOSCTitleCallback(func(runnerID int64, data *runnerv1.OSCTitleEvent) {
		callbackRunnerID = runnerID
		callbackData = data
	})

	time.Sleep(10 * time.Millisecond)

	// Handle OSC title
	event := &runnerv1.OSCTitleEvent{
		PodKey: "test-pod",
		Title:  "My Terminal",
	}
	cm.HandleOSCTitle(1, event)

	// Verify last ping was updated (heartbeat)
	assert.True(t, conn.GetLastPing().After(initialPing))

	// Verify callback was called
	assert.Equal(t, int64(1), callbackRunnerID)
	assert.Equal(t, "test-pod", callbackData.PodKey)
	assert.Equal(t, "My Terminal", callbackData.Title)
}

func TestConnectionManager_HandleOSCTitle_NoCallback(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	stream := newMockRunnerStream()
	defer stream.Close()

	cm.AddConnection(1, "test-node", "test-org", stream)

	// No callback set - should not panic
	event := &runnerv1.OSCTitleEvent{
		PodKey: "test-pod",
		Title:  "Test Title",
	}

	// This should not panic
	cm.HandleOSCTitle(1, event)
}

func TestConnectionManager_SetOSCCallbacks(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()

	// Test SetOSCNotificationCallback
	notifyCalled := false
	cm.SetOSCNotificationCallback(func(runnerID int64, data *runnerv1.OSCNotificationEvent) {
		notifyCalled = true
	})
	assert.NotNil(t, cm.onOSCNotification)

	// Test SetOSCTitleCallback
	titleCalled := false
	cm.SetOSCTitleCallback(func(runnerID int64, data *runnerv1.OSCTitleEvent) {
		titleCalled = true
	})
	assert.NotNil(t, cm.onOSCTitle)

	// Invoke callbacks to verify they work
	cm.onOSCNotification(1, &runnerv1.OSCNotificationEvent{})
	cm.onOSCTitle(1, &runnerv1.OSCTitleEvent{})

	assert.True(t, notifyCalled)
	assert.True(t, titleCalled)
}
