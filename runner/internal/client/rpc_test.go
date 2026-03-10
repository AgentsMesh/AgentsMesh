package client

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSender implements ConnectionSender for RPCClient testing.
type mockSender struct {
	mu      sync.Mutex
	lastMsg *runnerv1.RunnerMessage
	sendErr error
}

func (m *mockSender) SendMessage(msg *runnerv1.RunnerMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastMsg = msg
	return m.sendErr
}

func (m *mockSender) getLastMsg() *runnerv1.RunnerMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastMsg
}

// stubSender stubs SendMessage and also provides the ConnectionSender interface methods.
// Only SendMessage is used by RPCClient.
func (m *mockSender) SendPodCreated(string, int32, string, string) error { return nil }
func (m *mockSender) SendPodTerminated(string, int32, string) error      { return nil }
func (m *mockSender) SendPodInitProgress(string, string, int32, string) error {
	return nil
}
func (m *mockSender) SendError(string, string, string) error              { return nil }
func (m *mockSender) SendAgentStatus(string, string) error                { return nil }
func (m *mockSender) SendPtyResized(string, int32, int32) error           { return nil }
func (m *mockSender) SendOSCNotification(string, string, string) error    { return nil }
func (m *mockSender) SendOSCTitle(string, string) error                   { return nil }
func (m *mockSender) SendRequestRelayToken(string, string) error          { return nil }
func (m *mockSender) SendSandboxesStatus(string, []*SandboxStatusInfo) error { return nil }
func (m *mockSender) QueueLength() int                                    { return 0 }
func (m *mockSender) QueueCapacity() int                                  { return 100 }
func (m *mockSender) QueueUsage() float64                                 { return 0 }

func TestRPCClient_HandleResponse(t *testing.T) {
	sender := &mockSender{}
	rpc := NewRPCClient(sender)
	defer rpc.Stop()

	// Simulate a response for an unknown request (should not panic)
	rpc.HandleResponse(&runnerv1.McpResponse{
		RequestId: "unknown-id",
		Success:   true,
	})
}

func TestRPCClient_HandleResponse_Nil(t *testing.T) {
	sender := &mockSender{}
	rpc := NewRPCClient(sender)
	defer rpc.Stop()

	// Nil response should not panic
	rpc.HandleResponse(nil)
}

func TestRPCClient_CallAndResponse(t *testing.T) {
	sender := &mockSender{}
	rpc := NewRPCClient(sender)
	defer rpc.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start Call in background
	resultCh := make(chan struct {
		data []byte
		err  error
	}, 1)

	go func() {
		data, err := rpc.Call(ctx, "pod-1", "test/method", map[string]string{"key": "val"})
		resultCh <- struct {
			data []byte
			err  error
		}{data, err}
	}()

	// Wait for the message to be sent
	require.Eventually(t, func() bool {
		return sender.getLastMsg() != nil
	}, 2*time.Second, 5*time.Millisecond, "expected message to be sent")

	// Extract request ID from the sent message
	mcpReq := sender.getLastMsg().GetMcpRequest()
	require.NotNil(t, mcpReq)
	assert.Equal(t, "pod-1", mcpReq.PodKey)
	assert.Equal(t, "test/method", mcpReq.Method)

	// Deliver response
	rpc.HandleResponse(&runnerv1.McpResponse{
		RequestId: mcpReq.RequestId,
		Success:   true,
		Payload:   []byte(`{"result":"ok"}`),
	})

	result := <-resultCh
	require.NoError(t, result.err)
	assert.Equal(t, []byte(`{"result":"ok"}`), result.data)
}

func TestRPCClient_CallError(t *testing.T) {
	sender := &mockSender{}
	rpc := NewRPCClient(sender)
	defer rpc.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resultCh := make(chan error, 1)
	go func() {
		_, err := rpc.Call(ctx, "pod-1", "test/method", nil)
		resultCh <- err
	}()

	require.Eventually(t, func() bool {
		return sender.getLastMsg() != nil
	}, 2*time.Second, 5*time.Millisecond, "expected message to be sent")

	mcpReq := sender.getLastMsg().GetMcpRequest()
	require.NotNil(t, mcpReq)

	// Deliver error response
	rpc.HandleResponse(&runnerv1.McpResponse{
		RequestId: mcpReq.RequestId,
		Success:   false,
		Error: &runnerv1.McpError{
			Code:    404,
			Message: "not found",
		},
	})

	err := <-resultCh
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRPCClient_CallSendFailure(t *testing.T) {
	sender := &mockSender{sendErr: fmt.Errorf("connection lost")}
	rpc := NewRPCClient(sender)
	defer rpc.Stop()

	ctx := context.Background()
	_, err := rpc.Call(ctx, "pod-1", "test/method", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send MCP request")
}

func TestRPCClient_CallContextCancelled(t *testing.T) {
	sender := &mockSender{}
	rpc := NewRPCClient(sender)
	defer rpc.Stop()

	ctx, cancel := context.WithCancel(context.Background())

	resultCh := make(chan error, 1)
	go func() {
		_, err := rpc.Call(ctx, "pod-1", "test/method", nil)
		resultCh <- err
	}()

	require.Eventually(t, func() bool {
		return sender.getLastMsg() != nil
	}, 2*time.Second, 5*time.Millisecond, "expected message to be sent")
	cancel()

	err := <-resultCh
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRPCClient_Stop(t *testing.T) {
	sender := &mockSender{}
	rpc := NewRPCClient(sender)

	// Start a call in background
	resultCh := make(chan error, 1)
	go func() {
		_, err := rpc.Call(context.Background(), "pod-1", "test/method", nil)
		resultCh <- err
	}()

	require.Eventually(t, func() bool {
		return sender.getLastMsg() != nil
	}, 2*time.Second, 5*time.Millisecond, "expected message to be sent")

	// Stop should abort pending calls
	rpc.Stop()

	err := <-resultCh
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RPCClient stopped")
}

func TestRPCClient_StopIdempotent(t *testing.T) {
	sender := &mockSender{}
	rpc := NewRPCClient(sender)

	// Multiple stops should not panic
	rpc.Stop()
	rpc.Stop()
}
