package grpc

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// ==================== sendLoop Tests ====================

// mockSendStream is used for testing sendLoop
type mockSendStream struct {
	mu       sync.Mutex
	sendErr  error
	sentMsgs []*runnerv1.ServerMessage
}

func (m *mockSendStream) Send(msg *runnerv1.ServerMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sentMsgs = append(m.sentMsgs, msg)
	return nil
}

func (m *mockSendStream) getSentMsgs() []*runnerv1.ServerMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Return a copy to avoid race
	result := make([]*runnerv1.ServerMessage, len(m.sentMsgs))
	copy(result, m.sentMsgs)
	return result
}

func (m *mockSendStream) Recv() (*runnerv1.RunnerMessage, error) {
	return nil, nil
}

func (m *mockSendStream) Context() context.Context {
	return context.Background()
}

func (m *mockSendStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockSendStream) SendHeader(metadata.MD) error { return nil }
func (m *mockSendStream) SetTrailer(metadata.MD)       {}
func (m *mockSendStream) SendMsg(interface{}) error    { return nil }
func (m *mockSendStream) RecvMsg(interface{}) error    { return nil }

func TestGRPCRunnerAdapter_SendLoop_DoneSignal(t *testing.T) {
	logger := newTestLogger()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, nil, nil, nil, nil, connMgr)

	// Create connection
	mockStream := &mockRunnerStream{}
	conn := connMgr.AddConnection(1, "test-node", "test-org", mockStream)

	// Create grpcStreamAdapter with done channel
	done := make(chan struct{})
	grpcAdapter := &grpcStreamAdapter{
		stream: &mockSendStream{},
		sendCh: make(chan *runnerv1.ServerMessage, 10),
		done:   done,
	}

	// Start sendLoop in goroutine
	finished := make(chan struct{})
	go func() {
		adapter.sendLoop(1, conn, grpcAdapter)
		close(finished)
	}()

	// Close done channel to signal sendLoop to stop
	close(done)

	// Wait for sendLoop to finish
	select {
	case <-finished:
		// Expected - sendLoop exited on done signal
	case <-time.After(time.Second):
		t.Fatal("sendLoop did not exit on done signal")
	}
}

func TestGRPCRunnerAdapter_SendLoop_ChannelClosed(t *testing.T) {
	logger := newTestLogger()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, nil, nil, nil, nil, connMgr)

	// Create connection
	mockStream := &mockRunnerStream{}
	conn := connMgr.AddConnection(1, "test-node", "test-org", mockStream)

	// Create grpcStreamAdapter
	done := make(chan struct{})
	grpcAdapter := &grpcStreamAdapter{
		stream: &mockSendStream{},
		sendCh: make(chan *runnerv1.ServerMessage, 10),
		done:   done,
	}

	// Start sendLoop in goroutine
	finished := make(chan struct{})
	go func() {
		adapter.sendLoop(1, conn, grpcAdapter)
		close(finished)
	}()

	// Close the connection's Send channel
	conn.Close()

	// Wait for sendLoop to finish
	select {
	case <-finished:
		// Expected - sendLoop exited when channel closed
	case <-time.After(time.Second):
		t.Fatal("sendLoop did not exit when Send channel closed")
	}
}

func TestGRPCRunnerAdapter_SendLoop_SendError(t *testing.T) {
	logger := newTestLogger()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, nil, nil, nil, nil, connMgr)

	// Create connection
	mockStream := &mockRunnerStream{}
	conn := connMgr.AddConnection(1, "test-node", "test-org", mockStream)

	// Create mock stream that returns error on Send
	errorStream := &mockSendStream{
		sendErr: context.DeadlineExceeded,
	}

	// Create grpcStreamAdapter
	done := make(chan struct{})
	grpcAdapter := &grpcStreamAdapter{
		stream: errorStream,
		sendCh: make(chan *runnerv1.ServerMessage, 10),
		done:   done,
	}

	// Start sendLoop in goroutine
	finished := make(chan struct{})
	go func() {
		adapter.sendLoop(1, conn, grpcAdapter)
		close(finished)
	}()

	// Send a message via the connection
	msg := &runnerv1.ServerMessage{Timestamp: 12345}
	conn.SendMessage(msg)

	// Wait for sendLoop to finish (should exit on send error)
	select {
	case <-finished:
		// Expected - sendLoop exited on send error
	case <-time.After(time.Second):
		t.Fatal("sendLoop did not exit on send error")
	}
}

func TestGRPCRunnerAdapter_SendLoop_SuccessfulSend(t *testing.T) {
	logger := newTestLogger()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, nil, nil, nil, nil, connMgr)

	// Create connection
	mockStream := &mockRunnerStream{}
	conn := connMgr.AddConnection(1, "test-node", "test-org", mockStream)

	// Create mock stream that tracks sent messages
	successStream := &mockSendStream{}

	// Create grpcStreamAdapter
	done := make(chan struct{})
	grpcAdapter := &grpcStreamAdapter{
		stream: successStream,
		sendCh: make(chan *runnerv1.ServerMessage, 10),
		done:   done,
	}

	// Start sendLoop in goroutine
	go adapter.sendLoop(1, conn, grpcAdapter)

	// Send a message via the connection
	msg := &runnerv1.ServerMessage{Timestamp: 12345}
	conn.SendMessage(msg)

	// Wait a bit for message to be processed
	time.Sleep(50 * time.Millisecond)

	// Stop sendLoop
	close(done)

	// Verify message was sent (use thread-safe getter)
	msgs := successStream.getSentMsgs()
	assert.Len(t, msgs, 1)
	assert.Equal(t, int64(12345), msgs[0].Timestamp)
}

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

	adapter := NewGRPCRunnerAdapter(logger, nil, nil, nil, nil, nil, connMgr)

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

	adapter := NewGRPCRunnerAdapter(logger, nil, nil, nil, nil, nil, connMgr)

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

	adapter := NewGRPCRunnerAdapter(logger, nil, nil, nil, nil, nil, connMgr)

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

	adapter := NewGRPCRunnerAdapter(logger, nil, nil, nil, nil, nil, connMgr)

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
