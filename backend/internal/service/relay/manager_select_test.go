package relay

import (
	"testing"
)

func TestSelectRelay(t *testing.T) {
	m := NewManager()
	if err := m.Register(&RelayInfo{ID: "relay-us", URL: "wss://us.relay.com", Region: "us-east", Capacity: 100, Healthy: true}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := m.Register(&RelayInfo{ID: "relay-eu", URL: "wss://eu.relay.com", Region: "eu-west", Capacity: 100, Healthy: true}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Same region should be preferred
	relay := m.SelectRelay("us-east")
	if relay == nil {
		t.Fatal("SelectRelay returned nil")
	}
	if relay.ID != "relay-us" {
		t.Errorf("should prefer same region: got %q", relay.ID)
	}

	// Different region
	relay = m.SelectRelay("eu-west")
	if relay.ID != "relay-eu" {
		t.Errorf("should prefer same region: got %q", relay.ID)
	}
}

func TestSelectRelayPreferLowLoad(t *testing.T) {
	m := NewManager()
	if err := m.Register(&RelayInfo{ID: "relay-1", URL: "wss://r1.com", Region: "us", Capacity: 100, CurrentConnections: 90}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := m.Register(&RelayInfo{ID: "relay-2", URL: "wss://r2.com", Region: "us", Capacity: 100, CurrentConnections: 10}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	relay := m.SelectRelay("us")
	if relay == nil {
		t.Fatal("SelectRelay returned nil")
	}
	if relay.ID != "relay-2" {
		t.Errorf("should prefer lower load: got %q", relay.ID)
	}
}

func TestSelectRelaySkipsUnhealthy(t *testing.T) {
	m := NewManager()
	// Register both relays (both become healthy automatically)
	if err := m.Register(&RelayInfo{ID: "relay-1", URL: "wss://r1.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := m.Register(&RelayInfo{ID: "relay-2", URL: "wss://r2.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Mark relay-1 as unhealthy directly
	m.mu.Lock()
	m.relays["relay-1"].Healthy = false
	m.mu.Unlock()

	relay := m.SelectRelay("us")
	if relay == nil {
		t.Fatal("SelectRelay returned nil")
	}
	if relay.ID != "relay-2" {
		t.Errorf("should skip unhealthy: got %q", relay.ID)
	}
}

func TestSelectRelayNoHealthy(t *testing.T) {
	m := NewManager()
	if err := m.Register(&RelayInfo{ID: "relay-1", URL: "wss://r1.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Mark as unhealthy directly
	m.mu.Lock()
	m.relays["relay-1"].Healthy = false
	m.mu.Unlock()

	relay := m.SelectRelay("us")
	if relay != nil {
		t.Error("should return nil when no healthy relays")
	}
}

func TestSelectRelayForPod(t *testing.T) {
	m := NewManager()
	if err := m.Register(&RelayInfo{ID: "relay-1", URL: "wss://relay1.example.com", Healthy: true}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := m.Register(&RelayInfo{ID: "relay-2", URL: "wss://relay2.example.com", Healthy: true}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Same org should always get the same relay (affinity)
	r1 := m.SelectRelayForPod("org-alpha")
	if r1 == nil {
		t.Fatal("SelectRelayForPod returned nil relay")
	}

	r2 := m.SelectRelayForPod("org-alpha")
	if r2 == nil {
		t.Fatal("SelectRelayForPod returned nil relay")
	}

	// Same org should select same relay (affinity)
	if r1.ID != r2.ID {
		t.Errorf("same org should select same relay: got %q and %q", r1.ID, r2.ID)
	}
}

func TestSelectRelayWithAffinity(t *testing.T) {
	m := NewManager()

	// Register multiple relays
	if err := m.Register(&RelayInfo{ID: "relay-a", URL: "wss://a.relay.com", Healthy: true, Capacity: 100}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := m.Register(&RelayInfo{ID: "relay-b", URL: "wss://b.relay.com", Healthy: true, Capacity: 100}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := m.Register(&RelayInfo{ID: "relay-c", URL: "wss://c.relay.com", Healthy: true, Capacity: 100}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Same org should consistently get the same relay
	org1Relay1 := m.SelectRelayWithAffinity("org-one")
	org1Relay2 := m.SelectRelayWithAffinity("org-one")
	if org1Relay1.ID != org1Relay2.ID {
		t.Errorf("same org should select same relay: got %q and %q", org1Relay1.ID, org1Relay2.ID)
	}

	// Different orgs may get different relays (load distribution)
	// Note: due to hash distribution, they might still be the same,
	// but the algorithm should provide stable selection
	org2Relay := m.SelectRelayWithAffinity("org-two")
	if org2Relay == nil {
		t.Fatal("SelectRelayWithAffinity returned nil for org-two")
	}
}

func TestSelectRelayWithAffinityFallback(t *testing.T) {
	m := NewManager()

	if err := m.Register(&RelayInfo{ID: "relay-1", URL: "wss://r1.com", Healthy: true, Capacity: 100}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := m.Register(&RelayInfo{ID: "relay-2", URL: "wss://r2.com", Healthy: true, Capacity: 100}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Get primary relay for org
	primary := m.SelectRelayWithAffinity("test-org")
	if primary == nil {
		t.Fatal("SelectRelayWithAffinity returned nil")
	}

	// Mark primary as unhealthy
	m.mu.Lock()
	m.relays[primary.ID].Healthy = false
	m.mu.Unlock()

	// Should fallback to secondary
	fallback := m.SelectRelayWithAffinity("test-org")
	if fallback == nil {
		t.Fatal("fallback SelectRelayWithAffinity returned nil")
	}
	if fallback.ID == primary.ID {
		t.Error("should fallback to different relay when primary is unhealthy")
	}

	// Mark primary as healthy again
	m.mu.Lock()
	m.relays[primary.ID].Healthy = true
	m.mu.Unlock()

	// Should return to primary
	restored := m.SelectRelayWithAffinity("test-org")
	if restored == nil {
		t.Fatal("restored SelectRelayWithAffinity returned nil")
	}
	if restored.ID != primary.ID {
		t.Errorf("should return to primary when restored: got %q, want %q", restored.ID, primary.ID)
	}
}

func TestSelectRelaySkipsOverloaded(t *testing.T) {
	m := NewManager()

	if err := m.Register(&RelayInfo{ID: "relay-overloaded", URL: "wss://r1.com", Healthy: true, Capacity: 100, CPUUsage: 90}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := m.Register(&RelayInfo{ID: "relay-ok", URL: "wss://r2.com", Healthy: true, Capacity: 100, CPUUsage: 50}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Find an org that would normally select the overloaded relay first
	// but should skip it due to high CPU
	relay := m.SelectRelayWithAffinity("any-org")
	if relay == nil {
		t.Fatal("SelectRelayWithAffinity returned nil")
	}
	if relay.ID == "relay-overloaded" {
		t.Error("should skip relay with CPU > 80%")
	}
}

func TestSelectRelayWithOptions(t *testing.T) {
	m := NewManager()
	if err := m.Register(&RelayInfo{ID: "relay-1", URL: "wss://r1.com", Region: "us-east", Capacity: 100, Healthy: true}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := m.Register(&RelayInfo{ID: "relay-2", URL: "wss://r2.com", Region: "us-east", Capacity: 100, Healthy: true}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Select without exclusion
	relay := m.SelectRelayWithOptions(SelectRelayOptions{Region: "us-east"})
	if relay == nil {
		t.Fatal("SelectRelayWithOptions returned nil")
	}

	// Select with exclusion
	relay = m.SelectRelayWithOptions(SelectRelayOptions{
		Region:          "us-east",
		ExcludeRelayIDs: []string{"relay-1"},
	})
	if relay == nil {
		t.Fatal("SelectRelayWithOptions returned nil")
	}
	if relay.ID != "relay-2" {
		t.Errorf("should exclude relay-1: got %q", relay.ID)
	}

	// Exclude all
	relay = m.SelectRelayWithOptions(SelectRelayOptions{
		ExcludeRelayIDs: []string{"relay-1", "relay-2"},
	})
	if relay != nil {
		t.Error("should return nil when all excluded")
	}
}

func TestSelectRelayWeightedScoring(t *testing.T) {
	m := NewManager()

	// Register relays with different characteristics
	if err := m.Register(&RelayInfo{
		ID:                 "relay-high-load",
		URL:                "wss://r1.com",
		Region:             "us-east",
		Capacity:           100,
		CurrentConnections: 90,
		CPUUsage:           80,
		MemoryUsage:        70,
	}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := m.Register(&RelayInfo{
		ID:                 "relay-low-load",
		URL:                "wss://r2.com",
		Region:             "us-east",
		Capacity:           100,
		CurrentConnections: 10,
		CPUUsage:           20,
		MemoryUsage:        30,
	}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	relay := m.SelectRelay("us-east")
	if relay == nil {
		t.Fatal("SelectRelay returned nil")
	}
	if relay.ID != "relay-low-load" {
		t.Errorf("should prefer low load: got %q", relay.ID)
	}
}

func TestSelectRelayLatencyFactor(t *testing.T) {
	m := NewManager()

	// Register relays with different latencies (same other metrics)
	if err := m.Register(&RelayInfo{
		ID:           "relay-high-latency",
		URL:          "wss://r1.com",
		Region:       "us-east",
		Capacity:     100,
		AvgLatencyMs: 300,
	}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := m.Register(&RelayInfo{
		ID:           "relay-low-latency",
		URL:          "wss://r2.com",
		Region:       "us-east",
		Capacity:     100,
		AvgLatencyMs: 50,
	}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	relay := m.SelectRelay("us-east")
	if relay == nil {
		t.Fatal("SelectRelay returned nil")
	}
	if relay.ID != "relay-low-latency" {
		t.Errorf("should prefer low latency: got %q", relay.ID)
	}
}

func TestSelectRelayAtCapacity(t *testing.T) {
	m := NewManager()

	if err := m.Register(&RelayInfo{
		ID:                 "relay-full",
		URL:                "wss://r1.com",
		Region:             "us-east",
		Capacity:           100,
		CurrentConnections: 100, // At capacity
	}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := m.Register(&RelayInfo{
		ID:                 "relay-available",
		URL:                "wss://r2.com",
		Region:             "us-east",
		Capacity:           100,
		CurrentConnections: 50,
	}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	relay := m.SelectRelay("us-east")
	if relay == nil {
		t.Fatal("SelectRelay returned nil")
	}
	if relay.ID != "relay-available" {
		t.Errorf("should skip at-capacity relay: got %q", relay.ID)
	}
}
