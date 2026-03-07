package relay

import (
	"testing"
)

func TestSelectRelayWithAffinity(t *testing.T) {
	m := newTestManager(t)

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
	org2Relay := m.SelectRelayWithAffinity("org-two")
	if org2Relay == nil {
		t.Fatal("SelectRelayWithAffinity returned nil for org-two")
	}
}

func TestSelectRelayWithAffinityFallback(t *testing.T) {
	m := newTestManager(t)

	if err := m.Register(&RelayInfo{ID: "relay-1", URL: "wss://r1.com", Healthy: true, Capacity: 100}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := m.Register(&RelayInfo{ID: "relay-2", URL: "wss://r2.com", Healthy: true, Capacity: 100}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	primary := m.SelectRelayWithAffinity("test-org")
	if primary == nil {
		t.Fatal("SelectRelayWithAffinity returned nil")
	}

	// Mark primary as unhealthy
	m.mu.Lock()
	m.relays[primary.ID].Healthy = false
	m.mu.Unlock()

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

	restored := m.SelectRelayWithAffinity("test-org")
	if restored == nil {
		t.Fatal("restored SelectRelayWithAffinity returned nil")
	}
	if restored.ID != primary.ID {
		t.Errorf("should return to primary when restored: got %q, want %q", restored.ID, primary.ID)
	}
}

func TestSelectRelaySkipsOverloaded(t *testing.T) {
	m := newTestManager(t)

	if err := m.Register(&RelayInfo{ID: "relay-overloaded", URL: "wss://r1.com", Healthy: true, Capacity: 100, CPUUsage: 90}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := m.Register(&RelayInfo{ID: "relay-ok", URL: "wss://r2.com", Healthy: true, Capacity: 100, CPUUsage: 50}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	relay := m.SelectRelayWithAffinity("any-org")
	if relay == nil {
		t.Fatal("SelectRelayWithAffinity returned nil")
	}
	if relay.ID != "relay-ok" {
		t.Errorf("should prefer relay-ok over overloaded: got %q", relay.ID)
	}
}
