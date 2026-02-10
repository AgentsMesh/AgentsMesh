package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// ==================== RequestRelayToken Tests ====================

func TestGRPCRunnerAdapter_HandleProtoMessage_RequestRelayToken(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	// Add a connection
	mockStream := &mockRunnerStream{}
	conn := connMgr.AddConnection(1, "test-node", "test-org", mockStream)

	t.Run("request relay token message", func(t *testing.T) {
		var requestReceived bool
		var receivedRunnerID int64
		var receivedData *runnerv1.RequestRelayTokenEvent

		connMgr.SetRequestRelayTokenCallback(func(runnerID int64, data *runnerv1.RequestRelayTokenEvent) {
			requestReceived = true
			receivedRunnerID = runnerID
			receivedData = data
		})

		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_RequestRelayToken{
				RequestRelayToken: &runnerv1.RequestRelayTokenEvent{
					PodKey:    "test-pod",
					SessionId: "session-123",
					RelayUrl:  "wss://relay.example.com",
				},
			},
		}
		adapter.handleProtoMessage(context.Background(), 1, conn, msg)

		assert.True(t, requestReceived)
		assert.Equal(t, int64(1), receivedRunnerID)
		require.NotNil(t, receivedData)
		assert.Equal(t, "test-pod", receivedData.PodKey)
		assert.Equal(t, "session-123", receivedData.SessionId)
		assert.Equal(t, "wss://relay.example.com", receivedData.RelayUrl)
	})

	t.Run("request relay token message without callback", func(t *testing.T) {
		// Clear callback
		connMgr.SetRequestRelayTokenCallback(nil)

		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_RequestRelayToken{
				RequestRelayToken: &runnerv1.RequestRelayTokenEvent{
					PodKey:    "test-pod",
					SessionId: "session-123",
					RelayUrl:  "wss://relay.example.com",
				},
			},
		}

		// Should not panic
		assert.NotPanics(t, func() {
			adapter.handleProtoMessage(context.Background(), 1, conn, msg)
		})
	})
}
