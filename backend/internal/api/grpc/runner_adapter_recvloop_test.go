package grpc

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// ==================== receiveLoop Tests ====================

// mockRecvStream is used for testing receiveLoop
type mockRecvStream struct {
	msgs    []*runnerv1.RunnerMessage
	recvIdx int
	recvErr error
	ctx     context.Context
}

func (m *mockRecvStream) Send(msg *runnerv1.ServerMessage) error {
	return nil
}

func (m *mockRecvStream) Recv() (*runnerv1.RunnerMessage, error) {
	if m.recvErr != nil {
		return nil, m.recvErr
	}
	if m.recvIdx >= len(m.msgs) {
		return nil, io.EOF
	}
	msg := m.msgs[m.recvIdx]
	m.recvIdx++
	return msg, nil
}

func (m *mockRecvStream) Context() context.Context {
	if m.ctx != nil {
		return m.ctx
	}
	return context.Background()
}

func (m *mockRecvStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockRecvStream) SendHeader(metadata.MD) error { return nil }
func (m *mockRecvStream) SetTrailer(metadata.MD)       {}
func (m *mockRecvStream) SendMsg(interface{}) error    { return nil }
func (m *mockRecvStream) RecvMsg(interface{}) error    { return nil }

func TestGRPCRunnerAdapter_ReceiveLoop_EOF(t *testing.T) {
	logger := newTestLogger()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, nil, nil, nil, nil, connMgr, nil)

	// Create connection
	mockStream := &mockRunnerStream{}
	conn := connMgr.AddConnection(1, "test-node", "test-org", mockStream)

	// Create mock stream that returns EOF immediately
	recvStream := &mockRecvStream{
		msgs: []*runnerv1.RunnerMessage{}, // Empty, will return EOF
	}

	// Run receiveLoop
	err := adapter.receiveLoop(context.Background(), 1, conn, recvStream)

	// Should return nil on EOF (graceful disconnect)
	assert.NoError(t, err)
}

func TestGRPCRunnerAdapter_ReceiveLoop_Canceled(t *testing.T) {
	logger := newTestLogger()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, nil, nil, nil, nil, connMgr, nil)

	// Create connection
	mockStream := &mockRunnerStream{}
	conn := connMgr.AddConnection(1, "test-node", "test-org", mockStream)

	// Create mock stream that returns gRPC Canceled status error
	// Note: must use gRPC status.Error, not context.Canceled
	recvStream := &mockRecvStream{
		recvErr: status.Error(codes.Canceled, "context canceled"),
	}

	// Run receiveLoop
	err := adapter.receiveLoop(context.Background(), 1, conn, recvStream)

	// Should return nil on Canceled (graceful disconnect)
	assert.NoError(t, err)
}

func TestGRPCRunnerAdapter_ReceiveLoop_OtherError(t *testing.T) {
	logger := newTestLogger()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, nil, nil, nil, nil, connMgr, nil)

	// Create connection
	mockStream := &mockRunnerStream{}
	conn := connMgr.AddConnection(1, "test-node", "test-org", mockStream)

	// Create mock stream that returns an unexpected error
	recvStream := &mockRecvStream{
		recvErr: context.DeadlineExceeded, // Not EOF or Canceled
	}

	// Run receiveLoop
	err := adapter.receiveLoop(context.Background(), 1, conn, recvStream)

	// Should return the error
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestGRPCRunnerAdapter_ReceiveLoop_ProcessMessages(t *testing.T) {
	logger := newTestLogger()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, nil, nil, nil, nil, connMgr, nil)

	// Create connection
	mockStream := &mockRunnerStream{}
	conn := connMgr.AddConnection(1, "test-node", "test-org", mockStream)

	// Track received heartbeats
	var heartbeatCount int
	connMgr.SetHeartbeatCallback(func(runnerID int64, data *runnerv1.HeartbeatData) {
		heartbeatCount++
	})

	// Create mock stream with messages
	recvStream := &mockRecvStream{
		msgs: []*runnerv1.RunnerMessage{
			{
				Payload: &runnerv1.RunnerMessage_Heartbeat{
					Heartbeat: &runnerv1.HeartbeatData{NodeId: "test"},
				},
			},
			{
				Payload: &runnerv1.RunnerMessage_Heartbeat{
					Heartbeat: &runnerv1.HeartbeatData{NodeId: "test"},
				},
			},
		},
	}

	// Run receiveLoop
	err := adapter.receiveLoop(context.Background(), 1, conn, recvStream)

	// Should return nil after processing all messages and hitting EOF
	assert.NoError(t, err)
	assert.Equal(t, 2, heartbeatCount)
}
