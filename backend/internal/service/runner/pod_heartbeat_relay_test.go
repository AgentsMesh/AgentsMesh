package runner

import (
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

func TestHandleHeartbeat_RelayConnections(t *testing.T) {
	pc, _, _, db := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "relay-heartbeat-node",
		Status:         "online",
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Send heartbeat with relay connections (using Proto type)
	now := time.Now().UnixMilli()
	data := &runnerv1.HeartbeatData{
		NodeId: r.NodeID,
		Pods:   []*runnerv1.PodInfo{},
		RelayConnections: []*runnerv1.RelayConnectionInfo{
			{
				PodKey:      "pod-1",
				RelayUrl:    "wss://relay1.example.com",
				SessionId:   "session-1",
				Connected:   true,
				ConnectedAt: now,
			},
			{
				PodKey:      "pod-2",
				RelayUrl:    "wss://relay2.example.com",
				SessionId:   "session-2",
				Connected:   true,
				ConnectedAt: now - 60000, // 1 minute ago
			},
		},
	}

	pc.handleHeartbeat(r.ID, data)

	// Verify relay connections were cached
	connections := pc.GetRelayConnections(r.ID)
	if len(connections) != 2 {
		t.Fatalf("expected 2 relay connections, got %d", len(connections))
	}

	// Verify connection details
	connMap := make(map[string]RelayConnectionInfo)
	for _, conn := range connections {
		connMap[conn.PodKey] = conn
	}

	if conn, ok := connMap["pod-1"]; ok {
		if conn.RelayURL != "wss://relay1.example.com" {
			t.Errorf("pod-1 relay URL: expected wss://relay1.example.com, got %s", conn.RelayURL)
		}
		if conn.SessionID != "session-1" {
			t.Errorf("pod-1 session ID: expected session-1, got %s", conn.SessionID)
		}
		if !conn.Connected {
			t.Error("pod-1 should be connected")
		}
		if conn.ConnectedAt.UnixMilli() != now {
			t.Errorf("pod-1 connected_at: expected %d, got %d", now, conn.ConnectedAt.UnixMilli())
		}
	} else {
		t.Error("pod-1 connection not found")
	}

	if conn, ok := connMap["pod-2"]; ok {
		if conn.RelayURL != "wss://relay2.example.com" {
			t.Errorf("pod-2 relay URL: expected wss://relay2.example.com, got %s", conn.RelayURL)
		}
		if conn.SessionID != "session-2" {
			t.Errorf("pod-2 session ID: expected session-2, got %s", conn.SessionID)
		}
	} else {
		t.Error("pod-2 connection not found")
	}
}

func TestHandleHeartbeat_RelayConnectionsUpdate(t *testing.T) {
	pc, _, _, db := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "relay-update-node",
		Status:         "online",
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// First heartbeat with 2 connections
	data1 := &runnerv1.HeartbeatData{
		NodeId: r.NodeID,
		Pods:   []*runnerv1.PodInfo{},
		RelayConnections: []*runnerv1.RelayConnectionInfo{
			{PodKey: "pod-1", RelayUrl: "wss://relay.example.com", SessionId: "session-1", Connected: true},
			{PodKey: "pod-2", RelayUrl: "wss://relay.example.com", SessionId: "session-2", Connected: true},
		},
	}

	pc.handleHeartbeat(r.ID, data1)

	connections := pc.GetRelayConnections(r.ID)
	if len(connections) != 2 {
		t.Fatalf("expected 2 connections after first heartbeat, got %d", len(connections))
	}

	// Second heartbeat with only 1 connection (pod-2 disconnected)
	data2 := &runnerv1.HeartbeatData{
		NodeId: r.NodeID,
		Pods:   []*runnerv1.PodInfo{},
		RelayConnections: []*runnerv1.RelayConnectionInfo{
			{PodKey: "pod-1", RelayUrl: "wss://relay.example.com", SessionId: "session-1", Connected: true},
		},
	}

	pc.handleHeartbeat(r.ID, data2)

	connections = pc.GetRelayConnections(r.ID)
	if len(connections) != 1 {
		t.Fatalf("expected 1 connection after second heartbeat, got %d", len(connections))
	}

	if connections[0].PodKey != "pod-1" {
		t.Errorf("expected pod-1, got %s", connections[0].PodKey)
	}
}

func TestHandleHeartbeat_RelayConnectionsEmpty(t *testing.T) {
	pc, _, _, db := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "relay-empty-node",
		Status:         "online",
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// First heartbeat with connections
	data1 := &runnerv1.HeartbeatData{
		NodeId: r.NodeID,
		Pods:   []*runnerv1.PodInfo{},
		RelayConnections: []*runnerv1.RelayConnectionInfo{
			{PodKey: "pod-1", RelayUrl: "wss://relay.example.com", SessionId: "session-1", Connected: true},
		},
	}

	pc.handleHeartbeat(r.ID, data1)

	connections := pc.GetRelayConnections(r.ID)
	if len(connections) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(connections))
	}

	// Second heartbeat with empty connections (all disconnected)
	data2 := &runnerv1.HeartbeatData{
		NodeId:           r.NodeID,
		Pods:             []*runnerv1.PodInfo{},
		RelayConnections: []*runnerv1.RelayConnectionInfo{}, // Empty slice
	}

	pc.handleHeartbeat(r.ID, data2)

	// Cache should be cleared
	connections = pc.GetRelayConnections(r.ID)
	if len(connections) != 0 {
		t.Fatalf("expected 0 connections after empty heartbeat, got %d", len(connections))
	}
}

func TestHandleHeartbeat_RelayConnectionsNil(t *testing.T) {
	pc, _, _, db := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "relay-nil-node",
		Status:         "online",
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// First heartbeat with connections
	data1 := &runnerv1.HeartbeatData{
		NodeId: r.NodeID,
		Pods:   []*runnerv1.PodInfo{},
		RelayConnections: []*runnerv1.RelayConnectionInfo{
			{PodKey: "pod-1", RelayUrl: "wss://relay.example.com", SessionId: "session-1", Connected: true},
		},
	}

	pc.handleHeartbeat(r.ID, data1)

	connections := pc.GetRelayConnections(r.ID)
	if len(connections) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(connections))
	}

	// Second heartbeat with nil RelayConnections (field not set - old client?)
	data2 := &runnerv1.HeartbeatData{
		NodeId:           r.NodeID,
		Pods:             []*runnerv1.PodInfo{},
		RelayConnections: nil, // Nil - should preserve existing cache
	}

	pc.handleHeartbeat(r.ID, data2)

	// Cache should be preserved (nil means no update, not clear)
	connections = pc.GetRelayConnections(r.ID)
	if len(connections) != 1 {
		t.Fatalf("expected 1 connection to be preserved when RelayConnections is nil, got %d", len(connections))
	}
}

func TestHandleHeartbeat_RelayConnectionsDisconnectedState(t *testing.T) {
	pc, _, _, db := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "relay-disconnected-node",
		Status:         "online",
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Heartbeat with a disconnected relay connection
	data := &runnerv1.HeartbeatData{
		NodeId: r.NodeID,
		Pods:   []*runnerv1.PodInfo{},
		RelayConnections: []*runnerv1.RelayConnectionInfo{
			{
				PodKey:      "pod-1",
				RelayUrl:    "wss://relay.example.com",
				SessionId:   "session-1",
				Connected:   false, // Not connected
				ConnectedAt: 0,     // Never connected
			},
		},
	}

	pc.handleHeartbeat(r.ID, data)

	connections := pc.GetRelayConnections(r.ID)
	if len(connections) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(connections))
	}

	if connections[0].Connected {
		t.Error("connection should be marked as disconnected")
	}
	// Note: time.UnixMilli(0) returns Unix epoch (1970-01-01), not Go's zero time
	// So we check if it equals Unix epoch which represents "never connected"
	expectedTime := time.UnixMilli(0)
	if !connections[0].ConnectedAt.Equal(expectedTime) {
		t.Errorf("connected_at should be Unix epoch for ConnectedAt=0, got %v", connections[0].ConnectedAt)
	}
}

func TestHandleRunnerDisconnect_ClearsRelayCache(t *testing.T) {
	pc, _, _, db := setupPodEventHandlerDeps(t)

	// Create a runner
	r := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "relay-disconnect-node",
		Status:         "online",
	}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Add relay connections
	data := &runnerv1.HeartbeatData{
		NodeId: r.NodeID,
		Pods:   []*runnerv1.PodInfo{},
		RelayConnections: []*runnerv1.RelayConnectionInfo{
			{PodKey: "pod-1", RelayUrl: "wss://relay.example.com", SessionId: "session-1", Connected: true},
		},
	}

	pc.handleHeartbeat(r.ID, data)

	// Verify connections exist
	connections := pc.GetRelayConnections(r.ID)
	if len(connections) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(connections))
	}

	// Simulate runner disconnect
	pc.handleRunnerDisconnect(r.ID)

	// Verify cache is cleared
	connections = pc.GetRelayConnections(r.ID)
	if len(connections) != 0 {
		t.Fatalf("expected 0 connections after disconnect, got %d", len(connections))
	}
}

func TestGetRelayConnections_NonExistentRunner(t *testing.T) {
	pc, _, _, _ := setupPodEventHandlerDeps(t)

	// Query connections for non-existent runner
	connections := pc.GetRelayConnections(9999)
	if connections != nil {
		t.Errorf("expected nil for non-existent runner, got %v", connections)
	}
}

func TestHandleHeartbeat_MultipleRunners(t *testing.T) {
	pc, _, _, db := setupPodEventHandlerDeps(t)

	// Create two runners
	r1 := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "relay-multi-node-1",
		Status:         "online",
	}
	r2 := &runner.Runner{
		OrganizationID: 1,
		NodeID:         "relay-multi-node-2",
		Status:         "online",
	}
	if err := db.Create(r1).Error; err != nil {
		t.Fatalf("failed to create runner 1: %v", err)
	}
	if err := db.Create(r2).Error; err != nil {
		t.Fatalf("failed to create runner 2: %v", err)
	}

	// Heartbeat from runner 1
	data1 := &runnerv1.HeartbeatData{
		NodeId: r1.NodeID,
		Pods:   []*runnerv1.PodInfo{},
		RelayConnections: []*runnerv1.RelayConnectionInfo{
			{PodKey: "r1-pod-1", RelayUrl: "wss://relay1.example.com", SessionId: "session-1", Connected: true},
		},
	}
	pc.handleHeartbeat(r1.ID, data1)

	// Heartbeat from runner 2
	data2 := &runnerv1.HeartbeatData{
		NodeId: r2.NodeID,
		Pods:   []*runnerv1.PodInfo{},
		RelayConnections: []*runnerv1.RelayConnectionInfo{
			{PodKey: "r2-pod-1", RelayUrl: "wss://relay2.example.com", SessionId: "session-2", Connected: true},
			{PodKey: "r2-pod-2", RelayUrl: "wss://relay2.example.com", SessionId: "session-3", Connected: true},
		},
	}
	pc.handleHeartbeat(r2.ID, data2)

	// Verify each runner has its own connections
	conn1 := pc.GetRelayConnections(r1.ID)
	conn2 := pc.GetRelayConnections(r2.ID)

	if len(conn1) != 1 {
		t.Errorf("runner 1 should have 1 connection, got %d", len(conn1))
	}
	if len(conn2) != 2 {
		t.Errorf("runner 2 should have 2 connections, got %d", len(conn2))
	}

	// Verify isolation - runner 1's connections shouldn't affect runner 2
	if conn1[0].PodKey != "r1-pod-1" {
		t.Errorf("runner 1 should have r1-pod-1, got %s", conn1[0].PodKey)
	}
}
