package runner

import (
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

// TestOnListRelayConnections_Empty tests OnListRelayConnections with no pods
func TestOnListRelayConnections_Empty(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	connections := handler.OnListRelayConnections()
	if connections == nil {
		t.Fatal("OnListRelayConnections returned nil, expected empty slice")
	}
	if len(connections) != 0 {
		t.Errorf("expected 0 connections, got %d", len(connections))
	}
}

// TestOnListRelayConnections_PodsWithoutRelay tests OnListRelayConnections with pods that have no relay client
func TestOnListRelayConnections_PodsWithoutRelay(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Add pods without relay clients
	store.Put("pod-1", &Pod{PodKey: "pod-1", Status: PodStatusRunning})
	store.Put("pod-2", &Pod{PodKey: "pod-2", Status: PodStatusRunning})

	connections := handler.OnListRelayConnections()
	if len(connections) != 0 {
		t.Errorf("expected 0 connections for pods without relay, got %d", len(connections))
	}
}

// TestOnListRelayConnections_WithRelayClients tests OnListRelayConnections with pods that have relay clients
func TestOnListRelayConnections_WithRelayClients(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Create relay clients (not connected, just for testing the data retrieval)
	relayClient1 := relay.NewClient("wss://relay1.example.com", "pod-1", "session-1", "token-1", nil)
	relayClient2 := relay.NewClient("wss://relay2.example.com", "pod-2", "session-2", "token-2", nil)

	// Create pods with relay clients
	pod1 := &Pod{PodKey: "pod-1", Status: PodStatusRunning}
	pod1.SetRelayClient(relayClient1)

	pod2 := &Pod{PodKey: "pod-2", Status: PodStatusRunning}
	pod2.SetRelayClient(relayClient2)

	store.Put("pod-1", pod1)
	store.Put("pod-2", pod2)

	connections := handler.OnListRelayConnections()
	if len(connections) != 2 {
		t.Errorf("expected 2 connections, got %d", len(connections))
	}

	// Verify connection info
	connMap := make(map[string]client.RelayConnectionInfo)
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
		// Not connected since we didn't call Connect()
		if conn.Connected {
			t.Error("pod-1 should not be connected")
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

// TestOnListRelayConnections_MixedPods tests OnListRelayConnections with a mix of pods with and without relay clients
func TestOnListRelayConnections_MixedPods(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Pod with relay client
	relayClient := relay.NewClient("wss://relay.example.com", "pod-1", "session-1", "token-1", nil)
	pod1 := &Pod{PodKey: "pod-1", Status: PodStatusRunning}
	pod1.SetRelayClient(relayClient)

	// Pod without relay client
	pod2 := &Pod{PodKey: "pod-2", Status: PodStatusRunning}

	// Pod with nil relay client explicitly
	pod3 := &Pod{PodKey: "pod-3", Status: PodStatusRunning}
	pod3.SetRelayClient(nil)

	store.Put("pod-1", pod1)
	store.Put("pod-2", pod2)
	store.Put("pod-3", pod3)

	connections := handler.OnListRelayConnections()
	if len(connections) != 1 {
		t.Errorf("expected 1 connection, got %d", len(connections))
	}

	if len(connections) > 0 && connections[0].PodKey != "pod-1" {
		t.Errorf("expected pod-1, got %s", connections[0].PodKey)
	}
}

// TestOnListRelayConnections_ConnectedAt tests that ConnectedAt is correctly retrieved
func TestOnListRelayConnections_ConnectedAt(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Create relay client (not connected)
	relayClient := relay.NewClient("wss://relay.example.com", "pod-1", "session-1", "token-1", nil)

	pod := &Pod{PodKey: "pod-1", Status: PodStatusRunning}
	pod.SetRelayClient(relayClient)
	store.Put("pod-1", pod)

	connections := handler.OnListRelayConnections()
	if len(connections) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(connections))
	}

	// ConnectedAt should be 0 since we never connected
	if connections[0].ConnectedAt != 0 {
		t.Errorf("expected ConnectedAt to be 0 for unconnected client, got %d", connections[0].ConnectedAt)
	}
}

// TestRelayClient_GetConnectedAt tests the relay client's GetConnectedAt method
func TestRelayClient_GetConnectedAt(t *testing.T) {
	relayClient := relay.NewClient("wss://relay.example.com", "pod-1", "session-1", "token-1", nil)

	// Before connection, should be 0
	if relayClient.GetConnectedAt() != 0 {
		t.Errorf("expected 0 before connection, got %d", relayClient.GetConnectedAt())
	}

	// Note: To test non-zero ConnectedAt, we would need to actually connect to a server
	// which is covered in client_test.go integration tests
}

// TestRelayClient_GetSessionID tests the relay client's GetSessionID method
func TestRelayClient_GetSessionID(t *testing.T) {
	relayClient := relay.NewClient("wss://relay.example.com", "pod-1", "test-session-id", "token-1", nil)

	sessionID := relayClient.GetSessionID()
	if sessionID != "test-session-id" {
		t.Errorf("expected test-session-id, got %s", sessionID)
	}
}

// TestRelayClient_GetRelayURL tests the relay client's GetRelayURL method
func TestRelayClient_GetRelayURL(t *testing.T) {
	relayClient := relay.NewClient("wss://relay.example.com", "pod-1", "session-1", "token-1", nil)

	url := relayClient.GetRelayURL()
	if url != "wss://relay.example.com" {
		t.Errorf("expected wss://relay.example.com, got %s", url)
	}
}

// TestOnListRelayConnections_ConcurrentAccess tests concurrent access to OnListRelayConnections
func TestOnListRelayConnections_ConcurrentAccess(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Add some pods with relay clients
	for i := 0; i < 10; i++ {
		relayClient := relay.NewClient("wss://relay.example.com", "pod", "session", "token", nil)
		pod := &Pod{PodKey: "pod-" + string(rune('0'+i)), Status: PodStatusRunning}
		pod.SetRelayClient(relayClient)
		store.Put(pod.PodKey, pod)
	}

	// Concurrent reads
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				connections := handler.OnListRelayConnections()
				if connections == nil {
					t.Error("OnListRelayConnections returned nil")
				}
			}
			done <- struct{}{}
		}()
	}

	// Wait for all goroutines with timeout
	timeout := time.After(5 * time.Second)
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-timeout:
			t.Fatal("timeout waiting for concurrent reads")
		}
	}
}
