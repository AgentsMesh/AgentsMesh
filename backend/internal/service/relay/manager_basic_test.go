package relay

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.relays == nil {
		t.Error("relays map not initialized")
	}
}

func TestRegister(t *testing.T) {
	m := NewManager()
	info := &RelayInfo{
		ID:       "relay-1",
		URL:      "wss://relay.example.com",
		Region:   "us-east",
		Capacity: 1000,
	}

	if err := m.Register(info); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	relays := m.GetRelays()
	if len(relays) != 1 {
		t.Fatalf("expected 1 relay, got %d", len(relays))
	}
	if relays[0].ID != "relay-1" {
		t.Errorf("id: got %q, want %q", relays[0].ID, "relay-1")
	}
	if !relays[0].Healthy {
		t.Error("newly registered relay should be healthy")
	}
}

func TestHeartbeat(t *testing.T) {
	m := NewManager()
	if err := m.Register(&RelayInfo{ID: "relay-1", URL: "wss://relay.example.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err := m.Heartbeat("relay-1", 50, 25.5, 60.0)
	if err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}

	relays := m.GetRelays()
	if relays[0].CurrentConnections != 50 {
		t.Errorf("connections: got %d, want 50", relays[0].CurrentConnections)
	}
	if relays[0].CPUUsage != 25.5 {
		t.Errorf("cpu: got %f, want 25.5", relays[0].CPUUsage)
	}
}

func TestHeartbeatNotFound(t *testing.T) {
	m := NewManager()
	err := m.Heartbeat("unknown", 0, 0, 0)
	if err == nil {
		t.Error("expected error for unknown relay")
	}
}

func TestHeartbeatWithLatency(t *testing.T) {
	m := NewManager()
	if err := m.Register(&RelayInfo{ID: "relay-1", URL: "wss://relay.example.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// First heartbeat with latency
	err := m.HeartbeatWithLatency("relay-1", 50, 25.5, 60.0, 100)
	if err != nil {
		t.Fatalf("HeartbeatWithLatency: %v", err)
	}

	relays := m.GetRelays()
	if relays[0].AvgLatencyMs != 100 {
		t.Errorf("initial latency: got %d, want 100", relays[0].AvgLatencyMs)
	}

	// Second heartbeat should apply EMA smoothing
	err = m.HeartbeatWithLatency("relay-1", 50, 25.5, 60.0, 200)
	if err != nil {
		t.Fatalf("HeartbeatWithLatency: %v", err)
	}

	relays = m.GetRelays()
	// EMA: 100 * 0.7 + 200 * 0.3 = 70 + 60 = 130
	if relays[0].AvgLatencyMs != 130 {
		t.Errorf("smoothed latency: got %d, want 130", relays[0].AvgLatencyMs)
	}
}

func TestHeartbeatWithLatencyNotFound(t *testing.T) {
	m := NewManager()
	err := m.HeartbeatWithLatency("unknown", 0, 0, 0, 100)
	if err == nil {
		t.Error("expected error for unknown relay")
	}
}

func TestUnregister(t *testing.T) {
	m := NewManager()
	if err := m.Register(&RelayInfo{ID: "relay-1", URL: "wss://relay.example.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	m.Unregister("relay-1")

	if len(m.GetRelays()) != 0 {
		t.Error("relay should be removed")
	}
}

func TestGetRelayByID(t *testing.T) {
	m := NewManager()
	if err := m.Register(&RelayInfo{ID: "relay-1", URL: "wss://relay.example.com", Region: "us-east"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Found
	relay := m.GetRelayByID("relay-1")
	if relay == nil {
		t.Fatal("GetRelayByID returned nil")
	}
	if relay.ID != "relay-1" || relay.Region != "us-east" {
		t.Error("relay data mismatch")
	}

	// Not found
	if m.GetRelayByID("unknown") != nil {
		t.Error("should return nil for unknown relay")
	}
}

func TestRelayInfoGetRunnerURL(t *testing.T) {
	// With internal URL
	r1 := &RelayInfo{URL: "wss://public.com", InternalURL: "ws://internal:8090"}
	if r1.GetRunnerURL() != "ws://internal:8090" {
		t.Errorf("should return internal URL: %s", r1.GetRunnerURL())
	}

	// Without internal URL
	r2 := &RelayInfo{URL: "wss://public.com"}
	if r2.GetRunnerURL() != "wss://public.com" {
		t.Errorf("should fallback to URL: %s", r2.GetRunnerURL())
	}
}

func TestGetHealthyRelayCount(t *testing.T) {
	m := NewManager()
	if err := m.Register(&RelayInfo{ID: "relay-1", URL: "wss://r1.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := m.Register(&RelayInfo{ID: "relay-2", URL: "wss://r2.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := m.Register(&RelayInfo{ID: "relay-3", URL: "wss://r3.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Mark relay-3 as unhealthy directly
	m.mu.Lock()
	m.relays["relay-3"].Healthy = false
	m.mu.Unlock()

	if m.GetHealthyRelayCount() != 2 {
		t.Errorf("healthy count: got %d, want 2", m.GetHealthyRelayCount())
	}
}

func TestHasHealthyRelays(t *testing.T) {
	m := NewManager()
	if m.HasHealthyRelays() {
		t.Error("should be false with no relays")
	}

	if err := m.Register(&RelayInfo{ID: "relay-1", URL: "wss://r1.com", Healthy: true}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if !m.HasHealthyRelays() {
		t.Error("should be true with healthy relay")
	}
}

func TestGetStats(t *testing.T) {
	m := NewManager()
	if err := m.Register(&RelayInfo{ID: "relay-1", URL: "wss://r1.com", CurrentConnections: 10}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := m.Register(&RelayInfo{ID: "relay-2", URL: "wss://r2.com", CurrentConnections: 5}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Mark relay-2 as unhealthy directly
	m.mu.Lock()
	m.relays["relay-2"].Healthy = false
	m.mu.Unlock()

	stats := m.GetStats()
	if stats.TotalRelays != 2 {
		t.Errorf("TotalRelays: got %d, want 2", stats.TotalRelays)
	}
	if stats.HealthyRelays != 1 {
		t.Errorf("HealthyRelays: got %d, want 1", stats.HealthyRelays)
	}
	if stats.TotalConnections != 15 {
		t.Errorf("TotalConnections: got %d, want 15", stats.TotalConnections)
	}
}
