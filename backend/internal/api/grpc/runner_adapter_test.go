package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

func TestNewGRPCRunnerAdapter(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	assert.NotNil(t, adapter)
	assert.Equal(t, connMgr, adapter.connManager)
}

func TestGRPCRunnerAdapter_SendCreatePod(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	// Test sending to non-existent runner
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-pod",
		LaunchCommand: "claude",
	}
	err := adapter.SendCreatePod(999, cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestGRPCRunnerAdapter_SendTerminatePod(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	// Test sending to non-existent runner
	err := adapter.SendTerminatePod(999, "test-pod", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestGRPCRunnerAdapter_SendPodInput(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	// Test sending to non-existent runner
	err := adapter.SendPodInput(999, "test-pod", []byte("hello"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestGRPCRunnerAdapter_SendPrompt(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	// Test sending to non-existent runner
	err := adapter.SendPrompt(999, "test-pod", "Hello, Claude!")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestGRPCRunnerAdapter_IsConnected(t *testing.T) {
	logger := newTestLogger()
	runnerSvc := newMockRunnerService()
	orgSvc := newMockOrgService()
	connMgr := runner.NewRunnerConnectionManager(logger)

	adapter := NewGRPCRunnerAdapter(logger, nil, runnerSvc, orgSvc, nil, nil, connMgr, nil)

	// Initially not connected
	assert.False(t, adapter.IsConnected(1))

	// Add a connection via connManager
	mockStream := &mockRunnerStream{}
	connMgr.AddConnection(1, "test-node", "test-org", mockStream)

	// Now connected
	assert.True(t, adapter.IsConnected(1))
}

// ==================== Mock Definitions ====================

// mockRunnerStream implements runner.RunnerStream for testing with full type safety.
type mockRunnerStream struct {
	sent []*runnerv1.ServerMessage
}

// Compile-time check: mockRunnerStream implements runner.RunnerStream
var _ runner.RunnerStream = (*mockRunnerStream)(nil)

func (m *mockRunnerStream) Send(msg *runnerv1.ServerMessage) error {
	m.sent = append(m.sent, msg)
	return nil
}

func (m *mockRunnerStream) Recv() (*runnerv1.RunnerMessage, error) {
	return nil, context.Canceled
}

func (m *mockRunnerStream) Context() context.Context {
	return context.Background()
}

// mockConnectServer implements runnerv1.RunnerService_ConnectServer for testing
type mockConnectServer struct {
	ctx      context.Context
	messages []*runnerv1.ServerMessage
	recvCh   chan *runnerv1.RunnerMessage
}

func (m *mockConnectServer) Send(msg *runnerv1.ServerMessage) error {
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockConnectServer) Recv() (*runnerv1.RunnerMessage, error) {
	if m.recvCh == nil {
		return nil, context.Canceled
	}
	select {
	case msg := <-m.recvCh:
		return msg, nil
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	}
}

func (m *mockConnectServer) Context() context.Context {
	return m.ctx
}

func (m *mockConnectServer) SetHeader(metadata.MD) error  { return nil }
func (m *mockConnectServer) SendHeader(metadata.MD) error { return nil }
func (m *mockConnectServer) SetTrailer(metadata.MD)       {}
func (m *mockConnectServer) SendMsg(interface{}) error    { return nil }
func (m *mockConnectServer) RecvMsg(interface{}) error    { return nil }
