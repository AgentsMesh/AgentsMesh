package relay

import (
	"testing"
	"time"
)

func TestForceUnregister(t *testing.T) {
	m := NewManager()
	relay := &RelayInfo{ID: "relay-1", URL: "wss://relay.example.com"}
	if err := m.Register(relay); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Force unregister
	m.ForceUnregister("relay-1")

	// Verify relay is removed
	if m.GetRelayByID("relay-1") != nil {
		t.Error("relay should be removed")
	}
}

func TestForceUnregisterNotFound(t *testing.T) {
	m := NewManager()

	// Should not panic on unknown relay
	m.ForceUnregister("unknown")

	// Verify no side effects
	if len(m.GetRelays()) != 0 {
		t.Error("should have no relays")
	}
}

func TestGracefulUnregister(t *testing.T) {
	m := NewManager()
	relay := &RelayInfo{ID: "relay-1", URL: "wss://relay.example.com"}
	if err := m.Register(relay); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Graceful unregister
	m.GracefulUnregister("relay-1", "shutdown")

	// Verify relay is removed
	if m.GetRelayByID("relay-1") != nil {
		t.Error("relay should be removed")
	}
}

func TestGracefulUnregisterNotFound(t *testing.T) {
	m := NewManager()

	// Should not panic on unknown relay
	m.GracefulUnregister("unknown", "shutdown")

	// Verify no side effects
	if len(m.GetRelays()) != 0 {
		t.Error("should have no relays")
	}
}

func TestManagerStop(t *testing.T) {
	m := NewManager()

	// Should not be stopped initially
	if m.IsStopped() {
		t.Error("manager should not be stopped initially")
	}

	// Stop the manager
	m.Stop()

	// Should be stopped now
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
	m := NewManagerWithOptions(
		WithHealthCheckInterval(50 * time.Millisecond),
	)

	// Register a relay to trigger health check activity
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
	m := NewManager()

	// Register a healthy relay
	relay := &RelayInfo{ID: "relay-1", URL: "wss://r1.com"}
	if err := m.Register(relay); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Verify it's healthy
	r := m.GetRelayByID("relay-1")
	if !r.Healthy {
		t.Error("relay should be healthy after registration")
	}

	// Mark as unhealthy
	m.markRelayUnhealthy("relay-1")

	// Verify it's unhealthy
	r = m.GetRelayByID("relay-1")
	if r.Healthy {
		t.Error("relay should be unhealthy after markRelayUnhealthy")
	}
}

func TestMarkRelayUnhealthyNotFound(t *testing.T) {
	m := NewManager()

	// Should not panic on unknown relay
	m.markRelayUnhealthy("unknown")
}

func TestMarkRelayUnhealthyAlreadyUnhealthy(t *testing.T) {
	m := NewManager()

	// Register and mark unhealthy
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
