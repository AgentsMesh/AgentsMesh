package relay

import (
	"testing"
	"time"
)

// TestRelayManager_RegisterAndSelect registers a relay and verifies
// it can be selected via SelectRelayWithAffinity.
func TestRelayManager_RegisterAndSelect(t *testing.T) {
	store := NewMockStore()
	m := newTestManager(t, WithStore(store), WithHealthCheckInterval(time.Minute))

	info := &RelayInfo{
		ID:       "relay-int-1",
		URL:      "wss://relay1.example.com",
		Region:   "us-east",
		Capacity: 500,
	}

	if err := m.Register(info); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Verify in-memory
	relays := m.GetRelays()
	if len(relays) != 1 {
		t.Fatalf("relays = %d, want 1", len(relays))
	}
	if relays[0].ID != "relay-int-1" {
		t.Errorf("relay ID = %s, want relay-int-1", relays[0].ID)
	}
	if !relays[0].Healthy {
		t.Error("newly registered relay should be healthy")
	}

	// Verify in store
	stored, err := store.GetRelay(nil, "relay-int-1")
	if err != nil {
		t.Fatalf("store.GetRelay: %v", err)
	}
	if stored == nil {
		t.Fatal("relay not persisted to store")
	}
	if stored.URL != "wss://relay1.example.com" {
		t.Errorf("stored URL = %s, want wss://relay1.example.com", stored.URL)
	}

	// Select via affinity — should return the only available relay
	selected := m.SelectRelayWithAffinity("any-org")
	if selected == nil {
		t.Fatal("SelectRelayWithAffinity returned nil")
	}
	if selected.ID != "relay-int-1" {
		t.Errorf("selected ID = %s, want relay-int-1", selected.ID)
	}
}

// TestRelayManager_Unregister registers a relay, unregisters it, and
// verifies it is gone from both memory and store.
func TestRelayManager_Unregister(t *testing.T) {
	store := NewMockStore()
	m := newTestManager(t, WithStore(store), WithHealthCheckInterval(time.Minute))

	if err := m.Register(&RelayInfo{
		ID:       "relay-unreg",
		URL:      "wss://relay-unreg.example.com",
		Capacity: 100,
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Verify present
	if m.GetRelayByID("relay-unreg") == nil {
		t.Fatal("relay should exist before unregister")
	}

	// Force unregister
	m.ForceUnregister("relay-unreg")

	// Verify gone from memory
	if m.GetRelayByID("relay-unreg") != nil {
		t.Error("relay should be gone from memory after ForceUnregister")
	}

	// Verify gone from store
	stored, _ := store.GetRelay(nil, "relay-unreg")
	if stored != nil {
		t.Error("relay should be gone from store after ForceUnregister")
	}

	// Select should return nil (no relays)
	if m.SelectRelayWithAffinity("org") != nil {
		t.Error("SelectRelayWithAffinity should return nil with no relays")
	}
}

// TestRelayManager_StopGraceful starts a manager, registers a relay,
// stops gracefully, and verifies the manager is stopped.
func TestRelayManager_StopGraceful(t *testing.T) {
	store := NewMockStore()
	// Use short health check to verify the goroutine stops
	m := NewManagerWithOptions(
		WithStore(store),
		WithHealthCheckInterval(50*time.Millisecond),
	)

	if err := m.Register(&RelayInfo{
		ID:       "relay-stop",
		URL:      "wss://relay-stop.example.com",
		Capacity: 100,
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Stop
	m.Stop()

	if !m.IsStopped() {
		t.Error("IsStopped should be true after Stop")
	}

	// Double stop should not panic
	m.Stop()

	// Relays remain in memory (Stop does not clear them)
	if len(m.GetRelays()) != 1 {
		t.Errorf("relays after stop = %d, want 1", len(m.GetRelays()))
	}
}
