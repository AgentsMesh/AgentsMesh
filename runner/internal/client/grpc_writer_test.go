package client

import (
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
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
	if heartbeat == nil {
		t.Fatal("expected heartbeat message")
	}

	// Verify pods
	if len(heartbeat.Pods) != 1 {
		t.Errorf("expected 1 pod, got %d", len(heartbeat.Pods))
	}
	if heartbeat.Pods[0].PodKey != "pod-1" {
		t.Errorf("expected pod-1, got %s", heartbeat.Pods[0].PodKey)
	}

	// Verify relay connections
	if len(heartbeat.RelayConnections) != 1 {
		t.Errorf("expected 1 relay connection, got %d", len(heartbeat.RelayConnections))
	}
	rc := heartbeat.RelayConnections[0]
	if rc.PodKey != "pod-1" {
		t.Errorf("relay connection pod_key: expected pod-1, got %s", rc.PodKey)
	}
	if rc.RelayUrl != "wss://relay.example.com" {
		t.Errorf("relay connection relay_url: expected wss://relay.example.com, got %s", rc.RelayUrl)
	}
	if !rc.Connected {
		t.Error("relay connection should be connected")
	}
	if rc.ConnectedAt != now {
		t.Errorf("relay connection connected_at: expected %d, got %d", now, rc.ConnectedAt)
	}
}

// TestBuildHeartbeatMessage_EmptyRelayConnections verifies heartbeat message with empty relay connections
func TestBuildHeartbeatMessage_EmptyRelayConnections(t *testing.T) {
	handler := &mockHandlerWithRelayConnections{
		pods:             []PodInfo{},
		relayConnections: []RelayConnectionInfo{}, // Empty
	}

	msg := buildHeartbeatMessage("test-node", handler)
	heartbeat := msg.GetHeartbeat()
	if heartbeat == nil {
		t.Fatal("expected heartbeat message")
	}

	if len(heartbeat.Pods) != 0 {
		t.Errorf("expected 0 pods, got %d", len(heartbeat.Pods))
	}
	if len(heartbeat.RelayConnections) != 0 {
		t.Errorf("expected 0 relay connections, got %d", len(heartbeat.RelayConnections))
	}
}

// TestBuildHeartbeatMessage_NilHandler verifies heartbeat message with nil handler
func TestBuildHeartbeatMessage_NilHandler(t *testing.T) {
	msg := buildHeartbeatMessage("test-node", nil)
	heartbeat := msg.GetHeartbeat()
	if heartbeat == nil {
		t.Fatal("expected heartbeat message")
	}

	// With nil handler, pods and relay connections should be nil/empty
	if len(heartbeat.Pods) != 0 {
		t.Errorf("expected 0 pods with nil handler, got %d", len(heartbeat.Pods))
	}
	if len(heartbeat.RelayConnections) != 0 {
		t.Errorf("expected 0 relay connections with nil handler, got %d", len(heartbeat.RelayConnections))
	}
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
	if heartbeat == nil {
		t.Fatal("expected heartbeat message")
	}

	if len(heartbeat.Pods) != 3 {
		t.Errorf("expected 3 pods, got %d", len(heartbeat.Pods))
	}
	if len(heartbeat.RelayConnections) != 3 {
		t.Errorf("expected 3 relay connections, got %d", len(heartbeat.RelayConnections))
	}

	// Verify mixed connected states
	connectedCount := 0
	for _, rc := range heartbeat.RelayConnections {
		if rc.Connected {
			connectedCount++
		}
	}
	if connectedCount != 2 {
		t.Errorf("expected 2 connected relay connections, got %d", connectedCount)
	}
}

// TestBuildHeartbeatMessage_NodeIdIncluded verifies heartbeat message includes correct node_id
func TestBuildHeartbeatMessage_NodeIdIncluded(t *testing.T) {
	handler := &mockHandlerWithRelayConnections{
		pods:             []PodInfo{},
		relayConnections: []RelayConnectionInfo{},
	}

	msg := buildHeartbeatMessage("my-test-node", handler)
	heartbeat := msg.GetHeartbeat()
	if heartbeat == nil {
		t.Fatal("expected heartbeat message")
	}

	if heartbeat.NodeId != "my-test-node" {
		t.Errorf("expected node_id 'my-test-node', got '%s'", heartbeat.NodeId)
	}
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
	if heartbeat == nil {
		t.Fatal("expected heartbeat message")
	}

	if len(heartbeat.Pods) != 1 {
		t.Fatalf("expected 1 pod, got %d", len(heartbeat.Pods))
	}

	pod := heartbeat.Pods[0]
	if pod.PodKey != "pod-1" {
		t.Errorf("pod_key: expected pod-1, got %s", pod.PodKey)
	}
	if pod.Status != "running" {
		t.Errorf("status: expected running, got %s", pod.Status)
	}
	if pod.AgentStatus != "executing" {
		t.Errorf("agent_status: expected executing, got %s", pod.AgentStatus)
	}
}
