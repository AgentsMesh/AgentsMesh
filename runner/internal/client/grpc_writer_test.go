package client

import (
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHandlerWithRelayConnections is a mock handler that returns relay connections.
type mockHandlerWithRelayConnections struct {
	pods             []PodInfo
	relayConnections []RelayConnectionInfo
}

func (h *mockHandlerWithRelayConnections) OnCreatePod(cmd *runnerv1.CreatePodCommand) error {
	return nil
}

func (h *mockHandlerWithRelayConnections) OnTerminatePod(req TerminatePodRequest) error {
	return nil
}

func (h *mockHandlerWithRelayConnections) OnListPods() []PodInfo {
	return h.pods
}

func (h *mockHandlerWithRelayConnections) OnListRelayConnections() []RelayConnectionInfo {
	return h.relayConnections
}

func (h *mockHandlerWithRelayConnections) OnTerminalInput(req TerminalInputRequest) error {
	return nil
}

func (h *mockHandlerWithRelayConnections) OnTerminalResize(req TerminalResizeRequest) error {
	return nil
}

func (h *mockHandlerWithRelayConnections) OnTerminalRedraw(req TerminalRedrawRequest) error {
	return nil
}

func (h *mockHandlerWithRelayConnections) OnSubscribeTerminal(req SubscribeTerminalRequest) error {
	return nil
}

func (h *mockHandlerWithRelayConnections) OnUnsubscribeTerminal(req UnsubscribeTerminalRequest) error {
	return nil
}

func (h *mockHandlerWithRelayConnections) OnQuerySandboxes(req QuerySandboxesRequest) error {
	return nil
}

func (h *mockHandlerWithRelayConnections) OnCreateAutopilot(cmd *runnerv1.CreateAutopilotCommand) error {
	return nil
}

func (h *mockHandlerWithRelayConnections) OnAutopilotControl(cmd *runnerv1.AutopilotControlCommand) error {
	return nil
}

// buildHeartbeatMessage builds a heartbeat message from handler data.
// This is the core logic tested - extracted for testing without needing stream connection.
func buildHeartbeatMessage(nodeID string, handler MessageHandler) *runnerv1.RunnerMessage {
	var pods []*runnerv1.PodInfo
	var relayConnections []*runnerv1.RelayConnectionInfo

	if handler != nil {
		internalPods := handler.OnListPods()
		for _, p := range internalPods {
			pods = append(pods, &runnerv1.PodInfo{
				PodKey:      p.PodKey,
				Status:      p.Status,
				AgentStatus: p.AgentStatus,
			})
		}

		internalRelayConns := handler.OnListRelayConnections()
		for _, rc := range internalRelayConns {
			relayConnections = append(relayConnections, &runnerv1.RelayConnectionInfo{
				PodKey:      rc.PodKey,
				RelayUrl:    rc.RelayURL,
				Connected:   rc.Connected,
				ConnectedAt: rc.ConnectedAt,
			})
		}
	}

	return &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_Heartbeat{
			Heartbeat: &runnerv1.HeartbeatData{
				NodeId:           nodeID,
				Pods:             pods,
				RelayConnections: relayConnections,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
}

// TestBuildHeartbeatMessage_CollectsRelayConnections verifies heartbeat message building with relay connections
func TestBuildHeartbeatMessage_CollectsRelayConnections(t *testing.T) {
	now := time.Now().UnixMilli()
	handler := &mockHandlerWithRelayConnections{
		pods: []PodInfo{
			{PodKey: "pod-1", Status: "running", Pid: 1234},
		},
		relayConnections: []RelayConnectionInfo{
			{
				PodKey:      "pod-1",
				RelayURL:    "wss://relay.example.com",
				Connected:   true,
				ConnectedAt: now,
			},
		},
	}

	msg := buildHeartbeatMessage("test-node", handler)
	heartbeat := msg.GetHeartbeat()
	require.NotNil(t, heartbeat, "expected heartbeat message")

	// Verify pods
	require.Len(t, heartbeat.Pods, 1)
	assert.Equal(t, "pod-1", heartbeat.Pods[0].PodKey)

	// Verify relay connections
	require.Len(t, heartbeat.RelayConnections, 1)
	rc := heartbeat.RelayConnections[0]
	assert.Equal(t, "pod-1", rc.PodKey)
	assert.Equal(t, "wss://relay.example.com", rc.RelayUrl)
	assert.True(t, rc.Connected)
	assert.Equal(t, now, rc.ConnectedAt)
}

// TestBuildHeartbeatMessage_EmptyRelayConnections verifies heartbeat message with empty relay connections
func TestBuildHeartbeatMessage_EmptyRelayConnections(t *testing.T) {
	handler := &mockHandlerWithRelayConnections{
		pods:             []PodInfo{},
		relayConnections: []RelayConnectionInfo{},
	}

	msg := buildHeartbeatMessage("test-node", handler)
	heartbeat := msg.GetHeartbeat()
	require.NotNil(t, heartbeat, "expected heartbeat message")
	assert.Empty(t, heartbeat.Pods)
	assert.Empty(t, heartbeat.RelayConnections)
}

// TestBuildHeartbeatMessage_NilHandler verifies heartbeat message with nil handler
func TestBuildHeartbeatMessage_NilHandler(t *testing.T) {
	msg := buildHeartbeatMessage("test-node", nil)
	heartbeat := msg.GetHeartbeat()
	require.NotNil(t, heartbeat, "expected heartbeat message")

	// With nil handler, pods and relay connections should be nil/empty
	assert.Empty(t, heartbeat.Pods)
	assert.Empty(t, heartbeat.RelayConnections)
}

// TestBuildHeartbeatMessage_MultipleRelayConnections verifies heartbeat message with multiple relay connections
func TestBuildHeartbeatMessage_MultipleRelayConnections(t *testing.T) {
	now := time.Now().UnixMilli()
	handler := &mockHandlerWithRelayConnections{
		pods: []PodInfo{
			{PodKey: "pod-1", Status: "running"},
			{PodKey: "pod-2", Status: "running"},
			{PodKey: "pod-3", Status: "running"},
		},
		relayConnections: []RelayConnectionInfo{
			{PodKey: "pod-1", RelayURL: "wss://relay1.example.com", Connected: true, ConnectedAt: now},
			{PodKey: "pod-2", RelayURL: "wss://relay2.example.com", Connected: true, ConnectedAt: now - 1000},
			{PodKey: "pod-3", RelayURL: "wss://relay1.example.com", Connected: false, ConnectedAt: 0},
		},
	}

	msg := buildHeartbeatMessage("test-node", handler)
	heartbeat := msg.GetHeartbeat()
	require.NotNil(t, heartbeat, "expected heartbeat message")

	assert.Len(t, heartbeat.Pods, 3)
	assert.Len(t, heartbeat.RelayConnections, 3)

	// Verify mixed connected states
	connectedCount := 0
	for _, rc := range heartbeat.RelayConnections {
		if rc.Connected {
			connectedCount++
		}
	}
	assert.Equal(t, 2, connectedCount, "expected 2 connected relay connections")
}

// TestBuildHeartbeatMessage_NodeIdIncluded verifies heartbeat message includes correct node_id
func TestBuildHeartbeatMessage_NodeIdIncluded(t *testing.T) {
	handler := &mockHandlerWithRelayConnections{
		pods:             []PodInfo{},
		relayConnections: []RelayConnectionInfo{},
	}

	msg := buildHeartbeatMessage("my-test-node", handler)
	heartbeat := msg.GetHeartbeat()
	require.NotNil(t, heartbeat, "expected heartbeat message")
	assert.Equal(t, "my-test-node", heartbeat.NodeId)
}

func TestSendHeartbeat_NoHandler(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)
	conn.handler = nil
	conn.agentProbe = NewAgentProbe()

	conn.sendHeartbeat()

	select {
	case msg := <-conn.controlCh:
		hb := msg.GetHeartbeat()
		require.NotNil(t, hb, "expected heartbeat payload")
		assert.Empty(t, hb.Pods)
	default:
		t.Fatal("expected heartbeat in control channel")
	}
}

func TestSendHeartbeat_WithHandler(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)
	conn.agentProbe = NewAgentProbe()

	handler := &mockHandlerWithRelayConnections{
		pods: []PodInfo{
			{PodKey: "pod-1", Status: "running", AgentStatus: "idle"},
		},
		relayConnections: []RelayConnectionInfo{
			{PodKey: "pod-1", RelayURL: "wss://relay.example.com", Connected: true},
		},
	}
	conn.handler = handler

	conn.sendHeartbeat()

	select {
	case msg := <-conn.controlCh:
		hb := msg.GetHeartbeat()
		require.NotNil(t, hb, "expected heartbeat payload")
		assert.Len(t, hb.Pods, 1)
		assert.Len(t, hb.RelayConnections, 1)
	default:
		t.Fatal("expected heartbeat in control channel")
	}
}

func TestSendAndRecord_NilStream(t *testing.T) {
	conn := newTestConnection()
	// stream is nil, should not panic
	conn.sendAndRecord(&runnerv1.RunnerMessage{})
}

func TestSendAndRecord_Success(t *testing.T) {
	conn := newTestConnection()
	setFakeStream(conn)

	before := time.Now().UnixNano()
	conn.sendAndRecord(&runnerv1.RunnerMessage{})

	after := time.Now().UnixNano()
	lastSend := conn.lastSendTime.Load()
	assert.GreaterOrEqual(t, lastSend, before, "lastSendTime should be >= before")
	assert.LessOrEqual(t, lastSend, after, "lastSendTime should be <= after")
}

// TestBuildHeartbeatMessage_PodFieldsMapping verifies pod fields are correctly mapped
func TestBuildHeartbeatMessage_PodFieldsMapping(t *testing.T) {
	handler := &mockHandlerWithRelayConnections{
		pods: []PodInfo{
			{PodKey: "pod-1", Status: "running", AgentStatus: "executing", Pid: 1234},
		},
		relayConnections: []RelayConnectionInfo{},
	}

	msg := buildHeartbeatMessage("test-node", handler)
	heartbeat := msg.GetHeartbeat()
	require.NotNil(t, heartbeat, "expected heartbeat message")
	require.Len(t, heartbeat.Pods, 1)

	pod := heartbeat.Pods[0]
	assert.Equal(t, "pod-1", pod.PodKey)
	assert.Equal(t, "running", pod.Status)
	assert.Equal(t, "executing", pod.AgentStatus)
}
