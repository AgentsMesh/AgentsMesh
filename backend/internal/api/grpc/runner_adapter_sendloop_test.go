package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// ==================== sendLoop Tests ====================
// mockSendStream is defined in runner_adapter_mock_sendstream_test.go

func TestGRPCRunnerAdapter_SendLoop_DoneSignal(t *testing.T) {
	logger := newTestLogger()
	connMgr := runner.NewRunnerConnectionManager(logger)
	defer connMgr.Close()

	adapter := NewGRPCRunnerAdapter(logger, nil, nil, nil, nil, nil, connMgr, nil)

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

	adapter := NewGRPCRunnerAdapter(logger, nil, nil, nil, nil, nil, connMgr, nil)

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

	adapter := NewGRPCRunnerAdapter(logger, nil, nil, nil, nil, nil, connMgr, nil)

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

	adapter := NewGRPCRunnerAdapter(logger, nil, nil, nil, nil, nil, connMgr, nil)

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
