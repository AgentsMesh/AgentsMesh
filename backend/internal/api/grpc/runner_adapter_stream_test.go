package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	runnerv1 "github.com/AgentsMesh/AgentsMesh/proto/gen/go/runner/v1"
)

// ==================== grpcStreamAdapter Tests ====================

func TestGrpcStreamAdapter_Send(t *testing.T) {
	t.Run("successful send", func(t *testing.T) {
		done := make(chan struct{})
		sendCh := make(chan *runnerv1.ServerMessage, 10)
		adapter := &grpcStreamAdapter{
			sendCh: sendCh,
			done:   done,
		}

		msg := &runnerv1.ServerMessage{
			Timestamp: 12345,
		}
		err := adapter.Send(msg)
		require.NoError(t, err)

		// Verify message was queued
		select {
		case received := <-sendCh:
			assert.Equal(t, msg, received)
		default:
			t.Fatal("expected message to be queued")
		}
	})

	t.Run("send when closed with full buffer", func(t *testing.T) {
		done := make(chan struct{})
		// Use unbuffered channel - will block on send, triggering select's other cases
		sendCh := make(chan *runnerv1.ServerMessage)
		adapter := &grpcStreamAdapter{
			sendCh: sendCh,
			done:   done,
		}
		close(done)

		// With default case in select:
		// - sendCh is full (unbuffered, no receiver)
		// - done is closed
		// - default case matches first due to non-blocking select
		// So we expect "buffer full" or "connection closed" depending on select order
		msg := &runnerv1.ServerMessage{Timestamp: 12345}
		err := adapter.Send(msg)
		require.Error(t, err)
		// The error could be either "buffer full" or "connection closed" due to select non-determinism
		// but with default present, "buffer full" is most likely
		errMsg := err.Error()
		assert.True(t, errMsg == "rpc error: code = ResourceExhausted desc = send buffer full" ||
			errMsg == "rpc error: code = Canceled desc = connection closed",
			"unexpected error: %s", errMsg)
	})
}

func TestGrpcStreamAdapter_Send_BufferFull(t *testing.T) {
	done := make(chan struct{})
	sendCh := make(chan *runnerv1.ServerMessage) // Unbuffered channel

	adapter := &grpcStreamAdapter{
		sendCh: sendCh,
		done:   done,
	}

	// Send should fail immediately as buffer is full (no receiver)
	msg := &runnerv1.ServerMessage{Timestamp: 12345}
	err := adapter.Send(msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "send buffer full")
}

func TestGrpcStreamAdapter_Recv(t *testing.T) {
	recvCh := make(chan *runnerv1.RunnerMessage, 1)
	mockStream := &mockConnectServer{
		ctx:    context.Background(),
		recvCh: recvCh,
	}

	adapter := &grpcStreamAdapter{
		stream: mockStream,
		sendCh: make(chan *runnerv1.ServerMessage, 10),
		done:   make(chan struct{}),
	}

	// Queue a message
	expectedMsg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_Heartbeat{
			Heartbeat: &runnerv1.HeartbeatData{NodeId: "test"},
		},
	}
	recvCh <- expectedMsg

	// Recv should return the message
	msg, err := adapter.Recv()
	require.NoError(t, err)
	assert.Equal(t, expectedMsg, msg)
}

func TestGrpcStreamAdapter_Context(t *testing.T) {
	ctx := context.WithValue(context.Background(), "key", "value")
	mockStream := &mockConnectServer{
		ctx: ctx,
	}

	adapter := &grpcStreamAdapter{
		stream: mockStream,
		sendCh: make(chan *runnerv1.ServerMessage, 10),
		done:   make(chan struct{}),
	}

	assert.Equal(t, ctx, adapter.Context())
}
