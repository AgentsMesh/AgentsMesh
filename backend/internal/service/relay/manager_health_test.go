package relay

import (
	"testing"
	"time"
)

func TestForceUnregister(t *testing.T) {
	m := newTestManager(t)
	relay := &RelayInfo{ID: "relay-1", URL: "wss://relay.example.com"}
	if err := m.Register(relay); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	m.ForceUnregister("relay-1")

	if m.GetRelayByID("relay-1") != nil {
		t.Error("relay should be removed")
	}
}

func TestForceUnregisterNotFound(t *testing.T) {
	m := newTestManager(t)

	// Should not panic on unknown relay
	m.ForceUnregister("unknown")

	if len(m.GetRelays()) != 0 {
		t.Error("should have no relays")
	}
}

func TestGracefulUnregister(t *testing.T) {
	m := newTestManager(t)
	relay := &RelayInfo{ID: "relay-1", URL: "wss://relay.example.com"}
	if err := m.Register(relay); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	m.GracefulUnregister("relay-1", "shutdown")

	if m.GetRelayByID("relay-1") != nil {
		t.Error("relay should be removed")
	}
}

func TestGracefulUnregisterNotFound(t *testing.T) {
	m := newTestManager(t)

	// Should not panic on unknown relay
	m.GracefulUnregister("unknown", "shutdown")

	if len(m.GetRelays()) != 0 {
		t.Error("should have no relays")
	}
}

func TestManagerStop(t *testing.T) {
	m := newTestManager(t)

	if m.IsStopped() {
		t.Error("manager should not be stopped initially")
	}

	m.Stop()

	if !m.IsStopped() {
		t.Error("manager should be stopped after Stop()")
	}

	// Stop should be idempotent (no panic on double stop)
	m.Stop()
	if !m.IsStopped() {
		t.Error("manager should remain stopped")
	}
}

func TestManagerStopWithHealthCheck(t *testing.T) {
	m := newTestManager(t,
		WithHealthCheckInterval(50*time.Millisecond),
	)

	if err := m.Register(&RelayInfo{ID: "relay-1", URL: "wss://r1.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Wait for at least one health check cycle
	time.Sleep(100 * time.Millisecond)

	// Stop should complete without blocking
	done := make(chan struct{})
	go func() {
		m.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Good, stop completed
	case <-time.After(500 * time.Millisecond):
		t.Error("Stop() should not block")
	}

	if !m.IsStopped() {
		t.Error("manager should be stopped")
	}
}

func TestMarkRelayUnhealthy(t *testing.T) {
	m := newTestManager(t)

	relay := &RelayInfo{ID: "relay-1", URL: "wss://r1.com"}
	if err := m.Register(relay); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	r := m.GetRelayByID("relay-1")
	if !r.Healthy {
		t.Error("relay should be healthy after registration")
	}

	// Set last heartbeat to past so TOCTOU re-check passes
	m.mu.Lock()
	m.relays["relay-1"].LastHeartbeat = time.Now().Add(-time.Minute)
	m.mu.Unlock()

	m.markRelayUnhealthy("relay-1")

	r = m.GetRelayByID("relay-1")
	if r.Healthy {
		t.Error("relay should be unhealthy after markRelayUnhealthy")
	}
}

func TestMarkRelayUnhealthyNotFound(t *testing.T) {
	m := newTestManager(t)

	// Should not panic on unknown relay
	m.markRelayUnhealthy("unknown")
}

func TestMarkRelayUnhealthyTOCTOUProtection(t *testing.T) {
	m := newTestManager(t)

	relay := &RelayInfo{ID: "relay-1", URL: "wss://r1.com"}
	if err := m.Register(relay); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Relay was just registered (LastHeartbeat ≈ now).
	// markRelayUnhealthy's TOCTOU re-check should detect the recent heartbeat
	// and skip marking it unhealthy.
	m.markRelayUnhealthy("relay-1")

	r := m.GetRelayByID("relay-1")
	if !r.Healthy {
		t.Error("recently heartbeated relay should remain healthy (TOCTOU protection)")
	}
}

func TestMarkRelayUnhealthyAlreadyUnhealthy(t *testing.T) {
	m := newTestManager(t)

	relay := &RelayInfo{ID: "relay-1", URL: "wss://r1.com"}
	if err := m.Register(relay); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	m.mu.Lock()
	m.relays["relay-1"].Healthy = false
	m.mu.Unlock()

	// Should be idempotent
	m.markRelayUnhealthy("relay-1")

	r := m.GetRelayByID("relay-1")
	if r.Healthy {
		t.Error("relay should still be unhealthy")
	}
}

func TestRemoveStaleRelay(t *testing.T) {
	m := newTestManager(t)

	if err := m.Register(&RelayInfo{ID: "relay-stale", URL: "wss://stale.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if m.GetRelayByID("relay-stale") == nil {
		t.Fatal("relay should exist after registration")
	}

	// Set last heartbeat to far past so re-check inside removeStaleRelay passes
	m.mu.Lock()
	m.relays["relay-stale"].LastHeartbeat = time.Now().Add(-time.Hour)
	m.mu.Unlock()

	m.removeStaleRelay("relay-stale")

	if m.GetRelayByID("relay-stale") != nil {
		t.Error("stale relay should be removed")
	}
}

func TestRemoveStaleRelaySkipsReRegistered(t *testing.T) {
	m := newTestManager(t)

	if err := m.Register(&RelayInfo{ID: "relay-1", URL: "wss://r1.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Relay was just registered (LastHeartbeat ≈ now), removeStaleRelay should NOT delete it
	m.removeStaleRelay("relay-1")

	if m.GetRelayByID("relay-1") == nil {
		t.Error("recently registered relay should NOT be removed by removeStaleRelay")
	}
}

func TestRemoveStaleRelayNotFound(t *testing.T) {
	m := newTestManager(t)

	// Should not panic on unknown relay
	m.removeStaleRelay("unknown")
}

func TestDoHealthCheckMarksUnhealthy(t *testing.T) {
	m := newTestManager(t,
		WithHealthCheckInterval(10*time.Millisecond),
	)

	if err := m.Register(&RelayInfo{ID: "relay-1", URL: "wss://r1.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Set last heartbeat to > 1 interval but < staleRelayMultiplier intervals ago
	// This should trigger the unhealthy path (not the stale removal path)
	m.mu.Lock()
	m.relays["relay-1"].LastHeartbeat = time.Now().Add(-50 * time.Millisecond) // 5x interval, < 10x stale
	m.mu.Unlock()

	m.doHealthCheck()

	r := m.GetRelayByID("relay-1")
	if r == nil {
		t.Fatal("relay should still exist (not stale yet)")
	}
	if r.Healthy {
		t.Error("relay should be marked unhealthy by doHealthCheck")
	}
}

func TestDoHealthCheckRemovesStaleRelays(t *testing.T) {
	m := newTestManager(t,
		WithHealthCheckInterval(10*time.Millisecond),
	)

	if err := m.Register(&RelayInfo{ID: "relay-1", URL: "wss://r1.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Set last heartbeat to far past (beyond staleRelayMultiplier * interval)
	m.mu.Lock()
	m.relays["relay-1"].LastHeartbeat = time.Now().Add(-time.Hour)
	m.mu.Unlock()

	// Run one health check cycle
	m.doHealthCheck()

	// Relay should be auto-removed (not just marked unhealthy)
	if m.GetRelayByID("relay-1") != nil {
		t.Error("stale relay should be auto-removed by doHealthCheck")
	}
}
