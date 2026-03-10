package runner

import (
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

// TestDisconnectRelay_NoDeadlockWithCloseHandler verifies that DisconnectRelay
// does not deadlock when Stop() triggers a close handler that accesses relayMu.
// The original bug: relayMu held → Stop() waits for readLoop → readLoop's close
// handler calls SetRelayClient → tries to acquire relayMu → deadlock.
// The fix: DisconnectRelay extracts the pointer under lock, then calls Stop() outside.
func TestDisconnectRelay_NoDeadlockWithCloseHandler(t *testing.T) {
	pod := &Pod{PodKey: "pod-disconnect-1", Status: PodStatusRunning}

	mc := relay.NewMockClient("wss://relay.example.com")
	mc.SetConnected(true)

	// Simulate real behavior: Stop() takes time (readLoop exit) and triggers close handler.
	mc.StopDelay = 50 * time.Millisecond
	mc.OnStopHook = func() {
		// This mimics the close handler calling SetRelayClient(nil).
		// If relayMu is still held by DisconnectRelay, this will deadlock.
		pod.SetRelayClient(nil)
	}

	pod.SetRelayClient(mc)

	done := make(chan struct{})
	go func() {
		pod.DisconnectRelay()
		close(done)
	}()

	select {
	case <-done:
		// Success — no deadlock.
	case <-time.After(3 * time.Second):
		t.Fatal("deadlock detected: DisconnectRelay blocked for 3s")
	}

	// Verify cleanup.
	if pod.GetRelayClient() != nil {
		t.Error("expected relay client to be nil after DisconnectRelay")
	}
	if !mc.StopCalled {
		t.Error("expected Stop() to be called")
	}
}

// TestDisconnectRelay_ConcurrentWithSubscribe verifies that DisconnectRelay
// and OnSubscribeTerminal can run concurrently without deadlock.
func TestDisconnectRelay_ConcurrentWithSubscribe(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, mockConn)

	handler.relayClientFactory = func(url, podKey, token string, logger *slog.Logger) relay.RelayClient {
		mc := relay.NewMockClient(url)
		return mc
	}

	pod := &Pod{PodKey: "pod-concurrent-ds", Status: PodStatusRunning}
	store.Put(pod.PodKey, pod)

	// Set an initial relay client.
	initialClient := relay.NewMockClient("wss://relay.example.com")
	initialClient.SetConnected(true)
	pod.SetRelayClient(initialClient)

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine 1: Disconnect.
	go func() {
		defer wg.Done()
		pod.DisconnectRelay()
	}()

	// Goroutine 2: Subscribe (may win or lose against disconnect).
	go func() {
		defer wg.Done()
		_ = handler.OnSubscribeTerminal(client.SubscribeTerminalRequest{
			PodKey:      pod.PodKey,
			RelayURL:    "wss://relay2.example.com",
			RunnerToken: "token-new",
		})
	}()

	allDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(allDone)
	}()

	select {
	case <-allDone:
		// Success — no deadlock.
	case <-time.After(5 * time.Second):
		t.Fatal("deadlock detected: concurrent DisconnectRelay + OnSubscribeTerminal blocked for 5s")
	}

	// Final state should be consistent: client is either nil or the new one.
	rc := pod.GetRelayClient()
	if rc != nil && rc == initialClient {
		t.Error("initial client should have been replaced or cleared")
	}
}

// TestDisconnectRelay_Idempotent verifies that calling DisconnectRelay multiple
// times is safe and idempotent.
func TestDisconnectRelay_Idempotent(t *testing.T) {
	pod := &Pod{PodKey: "pod-idempotent", Status: PodStatusRunning}

	mc := relay.NewMockClient("wss://relay.example.com")
	mc.SetConnected(true)
	pod.SetRelayClient(mc)

	// First disconnect.
	pod.DisconnectRelay()
	if pod.GetRelayClient() != nil {
		t.Error("expected nil after first DisconnectRelay")
	}

	// Second disconnect should be a no-op.
	pod.DisconnectRelay()
	if pod.GetRelayClient() != nil {
		t.Error("expected nil after second DisconnectRelay")
	}
}
