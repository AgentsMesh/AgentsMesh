package client

import (
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestConnection creates a GRPCConnection with test-friendly defaults.
func newTestConnection() *GRPCConnection {
	return &GRPCConnection{
		controlCh:         make(chan *runnerv1.RunnerMessage, 100),
		terminalCh:        make(chan *runnerv1.RunnerMessage, 100),
		stopCh:            make(chan struct{}),
		reconnectCh:       make(chan struct{}, 1),
		initResultCh:      make(chan *runnerv1.InitializeResult, 1),
		heartbeatInterval: 30 * time.Second,
		podQueue:          NewPodCommandQueue(),
	}
}

func TestHandleServerMessage_CreatePod(t *testing.T) {
	conn := newTestConnection()
	handler := &mockHandler{}
	conn.handler = handler

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_CreatePod{
			CreatePod: &runnerv1.CreatePodCommand{
				PodKey:        "test-pod",
				LaunchCommand: "echo",
			},
		},
	}

	conn.handleServerMessage(msg)
	// Async dispatch — wait briefly
	time.Sleep(50 * time.Millisecond)

	handler.mu.Lock()
	assert.True(t, handler.createPodCalled)
	assert.Equal(t, "test-pod", handler.lastCreateCmd.PodKey)
	handler.mu.Unlock()
}

func TestHandleServerMessage_TerminatePod(t *testing.T) {
	conn := newTestConnection()
	handler := &mockHandler{}
	conn.handler = handler

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_TerminatePod{
			TerminatePod: &runnerv1.TerminatePodCommand{
				PodKey: "test-pod",
				Force:  true,
			},
		},
	}

	conn.handleServerMessage(msg)
	time.Sleep(50 * time.Millisecond)

	handler.mu.Lock()
	assert.True(t, handler.terminatePodCalled)
	assert.Equal(t, "test-pod", handler.lastTerminateReq.PodKey)
	handler.mu.Unlock()
}

func TestHandleServerMessage_PodInput(t *testing.T) {
	conn := newTestConnection()
	handler := &mockHandler{}
	conn.handler = handler

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_PodInput{
			PodInput: &runnerv1.PodInputCommand{
				PodKey: "test-pod",
				Data:   []byte("hello"),
			},
		},
	}

	conn.handleServerMessage(msg) // Synchronous

	handler.mu.Lock()
	assert.True(t, handler.terminalInputCalled)
	assert.Equal(t, "test-pod", handler.lastInputReq.PodKey)
	assert.Equal(t, []byte("hello"), handler.lastInputReq.Data)
	handler.mu.Unlock()
}

func TestHandleServerMessage_SendPrompt(t *testing.T) {
	conn := newTestConnection()
	handler := &mockHandler{}
	conn.handler = handler

	msg := &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_SendPrompt{
			SendPrompt: &runnerv1.SendPromptCommand{
				PodKey: "test-pod",
				Prompt: "write hello world",
			},
		},
	}

	conn.handleServerMessage(msg)

	handler.mu.Lock()
	assert.True(t, handler.sendPromptCalled)
	assert.Equal(t, "test-pod", handler.lastSendPromptCmd.PodKey)
	assert.Equal(t, "write hello world", handler.lastSendPromptCmd.Prompt)
	handler.mu.Unlock()
}

func TestHandleServerMessage_NilHandler(t *testing.T) {
	conn := newTestConnection()
	// handler is nil

	// All message types should not panic with nil handler
	messages := [](*runnerv1.ServerMessage){
		{Payload: &runnerv1.ServerMessage_CreatePod{CreatePod: &runnerv1.CreatePodCommand{PodKey: "p"}}},
		{Payload: &runnerv1.ServerMessage_TerminatePod{TerminatePod: &runnerv1.TerminatePodCommand{PodKey: "p"}}},
		{Payload: &runnerv1.ServerMessage_PodInput{PodInput: &runnerv1.PodInputCommand{PodKey: "p"}}},
		{Payload: &runnerv1.ServerMessage_SendPrompt{SendPrompt: &runnerv1.SendPromptCommand{PodKey: "p"}}},
		{Payload: &runnerv1.ServerMessage_SubscribePod{SubscribePod: &runnerv1.SubscribePodCommand{PodKey: "p"}}},
		{Payload: &runnerv1.ServerMessage_UnsubscribePod{UnsubscribePod: &runnerv1.UnsubscribePodCommand{PodKey: "p"}}},
		{Payload: &runnerv1.ServerMessage_QuerySandboxes{QuerySandboxes: &runnerv1.QuerySandboxesCommand{RequestId: "r"}}},
		{Payload: &runnerv1.ServerMessage_CreateAutopilot{CreateAutopilot: &runnerv1.CreateAutopilotCommand{AutopilotKey: "a"}}},
		{Payload: &runnerv1.ServerMessage_AutopilotControl{AutopilotControl: &runnerv1.AutopilotControlCommand{AutopilotKey: "a"}}},
	}

	for _, msg := range messages {
		conn.handleServerMessage(msg)
	}
	// Wait for async handlers
	time.Sleep(50 * time.Millisecond)
	// No panic = pass
}

func TestHandleInitializeResult(t *testing.T) {
	conn := newTestConnection()

	result := &runnerv1.InitializeResult{
		ServerInfo: &runnerv1.ServerInfo{
			Version: "1.0.0",
		},
	}

	conn.handleInitializeResult(result)

	select {
	case received := <-conn.initResultCh:
		require.NotNil(t, received)
		assert.Equal(t, "1.0.0", received.ServerInfo.Version)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for init result")
	}
}

func TestHandleInitializeResult_ChannelFull(t *testing.T) {
	conn := newTestConnection()
	// Fill the channel
	conn.initResultCh <- &runnerv1.InitializeResult{
		ServerInfo: &runnerv1.ServerInfo{Version: "old"},
	}

	// Should not panic or block
	conn.handleInitializeResult(&runnerv1.InitializeResult{
		ServerInfo: &runnerv1.ServerInfo{Version: "new"},
	})
}

func TestHandleHeartbeatAck(t *testing.T) {
	conn := newTestConnection()
	monitor := NewHeartbeatMonitor(3, func() {})
	conn.heartbeatMonitor = monitor

	// Simulate sent heartbeat (increments missed count)
	monitor.OnSent()
	assert.Equal(t, int32(1), monitor.MissedCount())

	// HeartbeatAck should reset missed count
	ack := &runnerv1.HeartbeatAck{
		HeartbeatTimestamp: time.Now().UnixMilli(),
	}
	conn.handleHeartbeatAck(ack)
	assert.Equal(t, int32(0), monitor.MissedCount())
}

func TestHandleMcpResponse_NilRPCClient(t *testing.T) {
	conn := newTestConnection()
	conn.rpcClient = nil

	// Should not panic
	conn.handleMcpResponse(&runnerv1.McpResponse{
		RequestId: "test-req",
		Success:   true,
	})
}

func TestSetGetRPCClient(t *testing.T) {
	conn := newTestConnection()
	assert.Nil(t, conn.GetRPCClient())

	rpc := &RPCClient{}
	conn.SetRPCClient(rpc)
	assert.Equal(t, rpc, conn.GetRPCClient())
}
