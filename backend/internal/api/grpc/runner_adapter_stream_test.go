package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// ==================== grpcStreamAdapter Tests ====================

func TestGrpcStreamAdapter_Send(t *testing.T) {
	t.Run("successful send", func(t *testing.T) {
		done := make(chan struct{})
		mockStream := &mockSendStream{}
		adapter := &grpcStreamAdapter{
			stream: mockStream,
			done:   done,
		}

		msg := &runnerv1.ServerMessage{
			Timestamp: 12345,
		}
		err := adapter.Send(msg)
		require.NoError(t, err)

		// Verify message was sent to stream
		msgs := mockStream.getSentMsgs()
		require.Len(t, msgs, 1)
		assert.Equal(t, msg, msgs[0])
	})

	t.Run("send when closed", func(t *testing.T) {
		done := make(chan struct{})
		mockStream := &mockSendStream{}
		adapter := &grpcStreamAdapter{
			stream: mockStream,
			done:   done,
		}
		close(done)

		msg := &runnerv1.ServerMessage{Timestamp: 12345}
		err := adapter.Send(msg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "connection closed")
	})
}

func TestGrpcStreamAdapter_Send_StreamError(t *testing.T) {
	done := make(chan struct{})
	mockStream := &mockSendStream{
		sendErr: context.DeadlineExceeded,
	}

	adapter := &grpcStreamAdapter{
		stream: mockStream,
		done:   done,
	}

	msg := &runnerv1.ServerMessage{Timestamp: 12345}
	err := adapter.Send(msg)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestGrpcStreamAdapter_Recv(t *testing.T) {
	recvCh := make(chan *runnerv1.RunnerMessage, 1)
	mockStream := &mockConnectServer{
		ctx:    context.Background(),
		recvCh: recvCh,
	}

	adapter := &grpcStreamAdapter{
		stream: mockStream,
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
		done:   make(chan struct{}),
	}

	assert.Equal(t, ctx, adapter.Context())
}
