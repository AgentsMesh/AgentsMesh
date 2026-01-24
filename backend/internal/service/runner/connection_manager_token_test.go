package runner

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetRequestRelayTokenCallback(t *testing.T) {
	logger := slog.Default()
	cm := NewRunnerConnectionManager(logger)
	defer cm.Close()

	// Initially nil
	assert.Nil(t, cm.onRequestRelayToken)

	// Set callback
	var callbackCalled atomic.Bool
	var receivedRunnerID int64
	var receivedData *runnerv1.RequestRelayTokenEvent

	cm.SetRequestRelayTokenCallback(func(runnerID int64, data *runnerv1.RequestRelayTokenEvent) {
		callbackCalled.Store(true)
		receivedRunnerID = runnerID
		receivedData = data
	})

	// Verify callback is set
	assert.NotNil(t, cm.onRequestRelayToken)

	// Test calling the callback
	testData := &runnerv1.RequestRelayTokenEvent{
		PodKey:    "test-pod",
		SessionId: "test-session",
		RelayUrl:  "wss://relay.example.com",
	}
	cm.onRequestRelayToken(123, testData)

	assert.True(t, callbackCalled.Load())
	assert.Equal(t, int64(123), receivedRunnerID)
	assert.Equal(t, "test-pod", receivedData.PodKey)
	assert.Equal(t, "test-session", receivedData.SessionId)
	assert.Equal(t, "wss://relay.example.com", receivedData.RelayUrl)
}

func TestHandleRequestRelayToken(t *testing.T) {
	logger := slog.Default()
	cm := NewRunnerConnectionManager(logger)
	defer cm.Close()

	// Add a connection first
	mockStream := &mockRunnerStream{}
	conn := cm.AddConnection(123, "node-123", "test-org", mockStream)
	require.NotNil(t, conn)

	// Track callback invocations
	var callbackCalled atomic.Bool
	var receivedRunnerID int64
	var receivedData *runnerv1.RequestRelayTokenEvent

	cm.SetRequestRelayTokenCallback(func(runnerID int64, data *runnerv1.RequestRelayTokenEvent) {
		callbackCalled.Store(true)
		receivedRunnerID = runnerID
		receivedData = data
	})

	// Record last ping time before
	lastPingBefore := conn.GetLastPing()

	// Small delay to ensure time difference
	time.Sleep(5 * time.Millisecond)

	// Handle request
	testData := &runnerv1.RequestRelayTokenEvent{
		PodKey:    "test-pod",
		SessionId: "test-session",
		RelayUrl:  "wss://relay.example.com",
	}
	cm.HandleRequestRelayToken(123, testData)

	// Verify callback was called
	assert.True(t, callbackCalled.Load())
	assert.Equal(t, int64(123), receivedRunnerID)
	assert.Equal(t, "test-pod", receivedData.PodKey)

	// Verify heartbeat was updated
	lastPingAfter := conn.GetLastPing()
	assert.True(t, lastPingAfter.After(lastPingBefore))
}

func TestHandleRequestRelayToken_NoCallback(t *testing.T) {
	logger := slog.Default()
	cm := NewRunnerConnectionManager(logger)
	defer cm.Close()

	// Add a connection
	mockStream := &mockRunnerStream{}
	cm.AddConnection(123, "node-123", "test-org", mockStream)

	// Don't set callback - should not panic
	testData := &runnerv1.RequestRelayTokenEvent{
		PodKey:    "test-pod",
		SessionId: "test-session",
		RelayUrl:  "wss://relay.example.com",
	}

	// Should not panic
	assert.NotPanics(t, func() {
		cm.HandleRequestRelayToken(123, testData)
	})
}

func TestHandleRequestRelayToken_NoConnection(t *testing.T) {
	logger := slog.Default()
	cm := NewRunnerConnectionManager(logger)
	defer cm.Close()

	var callbackCalled atomic.Bool
	cm.SetRequestRelayTokenCallback(func(runnerID int64, data *runnerv1.RequestRelayTokenEvent) {
		callbackCalled.Store(true)
	})

	// Handle request for non-existent connection
	testData := &runnerv1.RequestRelayTokenEvent{
		PodKey:    "test-pod",
		SessionId: "test-session",
		RelayUrl:  "wss://relay.example.com",
	}
	cm.HandleRequestRelayToken(999, testData)

	// Callback should still be called (it just won't update heartbeat for non-existent connection)
	assert.True(t, callbackCalled.Load())
}

// mockRunnerStream implements RunnerStream for testing
type mockRunnerStream struct {
	sendCh chan *runnerv1.ServerMessage
}

func (m *mockRunnerStream) Send(msg *runnerv1.ServerMessage) error {
	if m.sendCh != nil {
		select {
		case m.sendCh <- msg:
		default:
		}
	}
	return nil
}

func (m *mockRunnerStream) Recv() (*runnerv1.RunnerMessage, error) {
	// Block forever for testing
	select {}
}

func (m *mockRunnerStream) Context() context.Context {
	return context.Background()
}
