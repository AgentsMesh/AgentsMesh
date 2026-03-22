package runner

import (
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

// TestOnSubscribePod_PodNotFound tests that OnSubscribePod returns error when pod not found
func TestOnSubscribePod_PodNotFound(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	err := handler.OnSubscribePod(client.SubscribePodRequest{
		PodKey:      "non-existent-pod",
		RelayURL:    "wss://relay.example.com",
		RunnerToken: "token-123",
	})

	if err == nil {
		t.Fatal("expected error for non-existent pod, got nil")
	}

	if !contains(err.Error(), "pod not found") {
		t.Errorf("expected error to contain 'pod not found', got: %s", err.Error())
	}
}

// TestOnSubscribePod_AlreadyConnectedSameRelay tests that when already connected to the same relay URL,
// only the token is updated without disconnecting and reconnecting.
// This is the key test case to prevent regression of the multi-client connection issue.
func TestOnSubscribePod_AlreadyConnectedSameRelay(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Create a pod
	pod := &Pod{PodKey: "pod-1", Status: PodStatusRunning}
	store.Put("pod-1", pod)

	// Create a mock relay client that is already connected
	relayURL := "wss://relay.example.com"
	mockRelayClient := relay.NewMockClient(relayURL)
	mockRelayClient.SetConnected(true)

	pod.SetRelayClient(mockRelayClient)

	// Subscribe with the same relay URL but new token
	err := handler.OnSubscribePod(client.SubscribePodRequest{
		PodKey:      "pod-1",
		RelayURL:    relayURL, // Same URL
		RunnerToken: "new-token",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Key assertion: Stop should NOT be called when already connected to same relay
	if mockRelayClient.StopCalled {
		t.Error("Stop() should NOT be called when already connected to the same relay URL")
	}

	// Verify that token was updated
	if len(mockRelayClient.UpdateTokenCalls) != 1 || mockRelayClient.UpdateTokenCalls[0] != "new-token" {
		t.Errorf("expected UpdateToken to be called with 'new-token', got calls: %v", mockRelayClient.UpdateTokenCalls)
	}

	// Verify that the same relay client is still attached to the pod
	currentClient := pod.GetRelayClient()
	if currentClient != mockRelayClient {
		t.Error("expected the same relay client to remain attached to the pod (no reconnect)")
	}
}

// TestOnSubscribePod_ConnectedToDifferentRelay tests that when connected to a different relay URL,
// the existing connection is disconnected and a new connection attempt is made.
func TestOnSubscribePod_ConnectedToDifferentRelay(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Create a pod with a mock relay client connected to a different URL
	pod := &Pod{PodKey: "pod-1", Status: PodStatusRunning}
	store.Put("pod-1", pod)

	oldMockClient := relay.NewMockClient("wss://old-relay.example.com")
	oldMockClient.SetConnected(true)
	pod.SetRelayClient(oldMockClient)

	// Subscribe with a different relay URL
	// This will fail at Connect() since we're connecting to a real (fake) URL
	_ = handler.OnSubscribePod(client.SubscribePodRequest{
		PodKey:      "pod-1",
		RelayURL:    "wss://new-relay.example.com", // Different URL
		RunnerToken: "new-token",
	})

	// Key assertion: Stop SHOULD be called when switching to a different relay
	if !oldMockClient.StopCalled {
		t.Error("Stop() SHOULD be called when switching to a different relay URL")
	}
}

// TestOnSubscribePod_ExistingClientNotConnected tests that when there's an existing client
// but it's not connected, a new connection is established.
func TestOnSubscribePod_ExistingClientNotConnected(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Create a pod with a relay client that is NOT connected
	pod := &Pod{PodKey: "pod-1", Status: PodStatusRunning}
	store.Put("pod-1", pod)

	oldMockClient := relay.NewMockClient("wss://relay.example.com")
	oldMockClient.SetConnected(false) // Not connected
	pod.SetRelayClient(oldMockClient)

	// Subscribe with the same relay URL
	// Since client is not connected, it should try to reconnect
	_ = handler.OnSubscribePod(client.SubscribePodRequest{
		PodKey:      "pod-1",
		RelayURL:    "wss://relay.example.com",
		RunnerToken: "new-token",
	})

	// When not connected (even to same URL), should call Stop to clean up
	if !oldMockClient.StopCalled {
		t.Error("Stop() SHOULD be called when existing client is not connected")
	}
}

// TestOnSubscribePod_NoExistingClient tests the normal case when there's no existing relay client.
func TestOnSubscribePod_NoExistingClient(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Create a pod without any relay client
	pod := &Pod{PodKey: "pod-1", Status: PodStatusRunning}
	store.Put("pod-1", pod)

	// Verify no existing client
	if pod.GetRelayClient() != nil {
		t.Fatal("pod should not have a relay client initially")
	}

	// Subscribe - will fail to connect but that's OK for this test
	// We just want to verify the flow when there's no existing client doesn't panic
	_ = handler.OnSubscribePod(client.SubscribePodRequest{
		PodKey:      "pod-1",
		RelayURL:    "wss://relay.example.com",
		RunnerToken: "token-123",
	})

	// Test passes if no panic occurs
}

// TestOnUnsubscribePod_PodNotFound tests that OnUnsubscribePod handles non-existent pod gracefully
func TestOnUnsubscribePod_PodNotFound(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Should not return error for non-existent pod
	err := handler.OnUnsubscribePod(client.UnsubscribePodRequest{
		PodKey: "non-existent-pod",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestOnUnsubscribePod_DisconnectsRelay tests that OnUnsubscribePod properly disconnects relay
func TestOnUnsubscribePod_DisconnectsRelay(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Create a pod with a mock relay client
	pod := &Pod{PodKey: "pod-1", Status: PodStatusRunning}
	store.Put("pod-1", pod)

	mockRelayClient := relay.NewMockClient("wss://relay.example.com")
	mockRelayClient.SetConnected(true)
	pod.SetRelayClient(mockRelayClient)

	// Verify client is attached
	if pod.GetRelayClient() == nil {
		t.Fatal("pod should have a relay client before unsubscribe")
	}

	err := handler.OnUnsubscribePod(client.UnsubscribePodRequest{
		PodKey: "pod-1",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify relay was disconnected
	if !mockRelayClient.StopCalled {
		t.Error("Stop() should be called on unsubscribe")
	}

	// Verify RelayClient is nil after DisconnectRelay
	if pod.GetRelayClient() != nil {
		t.Error("relay client should be nil after unsubscribe")
	}
}

// TestOnSubscribePod_MultipleClientsScenario tests the scenario where multiple clients
// (e.g., Web + Mobile) connect to the same pod. The second subscribe should not cause reconnect.
func TestOnSubscribePod_MultipleClientsScenario(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Create a pod
	pod := &Pod{PodKey: "pod-1", Status: PodStatusRunning}
	store.Put("pod-1", pod)

	relayURL := "wss://relay.example.com"

	// Simulate first client (e.g., Web) - create and set a mock relay client
	firstClient := relay.NewMockClient(relayURL)
	firstClient.SetConnected(true)
	pod.SetRelayClient(firstClient)

	// Simulate second client (e.g., Mobile) connecting to the same pod
	// This should NOT cause a disconnect/reconnect
	err := handler.OnSubscribePod(client.SubscribePodRequest{
		PodKey:      "pod-1",
		RelayURL:    relayURL, // Same relay URL
		RunnerToken: "token-v2",
	})

	if err != nil {
		t.Fatalf("second subscribe should succeed: %v", err)
	}

	// The key assertion: the same client instance should still be attached
	currentClient := pod.GetRelayClient()
	if currentClient != firstClient {
		t.Error("second subscribe should NOT create a new relay client when already connected to same URL")
	}

	// Verify Stop was not called
	if firstClient.StopCalled {
		t.Error("Stop() should NOT be called for second client connecting to same relay")
	}

	// Verify token was updated
	if len(firstClient.UpdateTokenCalls) != 1 || firstClient.UpdateTokenCalls[0] != "token-v2" {
		t.Errorf("expected token to be updated to 'token-v2', got: %v", firstClient.UpdateTokenCalls)
	}

	// Simulate third client connecting - should still not reconnect
	err = handler.OnSubscribePod(client.SubscribePodRequest{
		PodKey:      "pod-1",
		RelayURL:    relayURL,
		RunnerToken: "token-v3",
	})

	if err != nil {
		t.Fatalf("third subscribe should succeed: %v", err)
	}

	if pod.GetRelayClient() != firstClient {
		t.Error("third subscribe should NOT create a new relay client")
	}

	// Still should not have called Stop
	if firstClient.StopCalled {
		t.Error("Stop() should NOT be called for third client")
	}

	// Verify all token updates
	expectedTokens := []string{"token-v2", "token-v3"}
	if len(firstClient.UpdateTokenCalls) != 2 {
		t.Errorf("expected 2 token updates, got %d", len(firstClient.UpdateTokenCalls))
	}
	for i, token := range expectedTokens {
		if i < len(firstClient.UpdateTokenCalls) && firstClient.UpdateTokenCalls[i] != token {
			t.Errorf("expected token update %d to be '%s', got '%s'", i, token, firstClient.UpdateTokenCalls[i])
		}
	}
}

// TestOnSubscribePod_ReconnectAfterDisconnect tests that after a client disconnects,
// a new subscription correctly creates a new connection.
func TestOnSubscribePod_ReconnectAfterDisconnect(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	pod := &Pod{PodKey: "pod-1", Status: PodStatusRunning}
	store.Put("pod-1", pod)

	relayURL := "wss://relay.example.com"

	// First client connects
	firstClient := relay.NewMockClient(relayURL)
	firstClient.SetConnected(true)
	pod.SetRelayClient(firstClient)

	// Simulate disconnect (client is no longer connected)
	firstClient.SetConnected(false)

	// New subscription should create a new connection
	_ = handler.OnSubscribePod(client.SubscribePodRequest{
		PodKey:      "pod-1",
		RelayURL:    relayURL,
		RunnerToken: "new-token",
	})

	// Stop should be called to clean up the disconnected client
	if !firstClient.StopCalled {
		t.Error("Stop() should be called to clean up disconnected client")
	}
}
