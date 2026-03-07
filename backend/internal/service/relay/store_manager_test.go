package relay

import (
	"context"
	"testing"
	"time"
)

func TestManagerWithStore(t *testing.T) {
	store := NewMockStore()
	m := newTestManager(t, WithStore(store))

	if m == nil {
		t.Fatal("NewManagerWithOptions returned nil")
	}
	if m.store != store {
		t.Error("store not set")
	}
}

func TestRegisterPersistsToStore(t *testing.T) {
	store := NewMockStore()
	m := newTestManager(t, WithStore(store))

	relay := &RelayInfo{ID: "relay-1", URL: "wss://r1.com", Region: "us-east"}
	if err := m.Register(relay); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ctx := context.Background()
	stored, err := store.GetRelay(ctx, "relay-1")
	if err != nil {
		t.Fatalf("GetRelay: %v", err)
	}
	if stored == nil {
		t.Fatal("relay not persisted to store")
	}
	if stored.ID != "relay-1" || stored.Region != "us-east" {
		t.Error("relay data mismatch")
	}
}

func TestForceUnregisterDeletesFromStore(t *testing.T) {
	store := NewMockStore()
	m := newTestManager(t, WithStore(store))

	relay := &RelayInfo{ID: "relay-1", URL: "wss://r1.com"}
	if err := m.Register(relay); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	m.ForceUnregister("relay-1")

	ctx := context.Background()
	stored, _ := store.GetRelay(ctx, "relay-1")
	if stored != nil {
		t.Error("relay should be deleted from store")
	}
}

func TestForceUnregisterSkipsStoreWhenNotInMemory(t *testing.T) {
	store := NewMockStore()
	m := newTestManager(t, WithStore(store))

	// ForceUnregister a relay that was never registered
	store.mu.Lock()
	deleteBefore := store.deleteCalled
	store.mu.Unlock()

	m.ForceUnregister("non-existent")

	store.mu.Lock()
	deleteAfter := store.deleteCalled
	store.mu.Unlock()

	if deleteAfter != deleteBefore {
		t.Errorf("store.DeleteRelay should not be called for non-existent relay: calls before=%d, after=%d",
			deleteBefore, deleteAfter)
	}
}

func TestGracefulUnregisterCleansUpStore(t *testing.T) {
	store := NewMockStore()
	m := newTestManager(t, WithStore(store))

	relay := &RelayInfo{ID: "relay-1", URL: "wss://r1.com"}
	if err := m.Register(relay); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	m.GracefulUnregister("relay-1", "shutdown")

	ctx := context.Background()
	if r, _ := store.GetRelay(ctx, "relay-1"); r != nil {
		t.Error("relay should be deleted from store")
	}
}

func TestStaleRelayRemovalDeletesFromStore(t *testing.T) {
	store := NewMockStore()
	m := newTestManager(t,
		WithStore(store),
		WithHealthCheckInterval(10*time.Millisecond),
	)

	if err := m.Register(&RelayInfo{ID: "relay-stale", URL: "wss://stale.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Verify relay is in store
	ctx := context.Background()
	if r, _ := store.GetRelay(ctx, "relay-stale"); r == nil {
		t.Fatal("relay should be in store after registration")
	}

	// Set last heartbeat to far past
	m.mu.Lock()
	m.relays["relay-stale"].LastHeartbeat = time.Now().Add(-time.Hour)
	m.mu.Unlock()

	// Run health check to trigger auto-removal
	m.doHealthCheck()

	// Verify relay is removed from memory
	if m.GetRelayByID("relay-stale") != nil {
		t.Error("stale relay should be removed from memory")
	}

	// Verify relay is removed from store
	if r, _ := store.GetRelay(ctx, "relay-stale"); r != nil {
		t.Error("stale relay should be deleted from store")
	}
}

func TestLoadFromStoreSuccess(t *testing.T) {
	store := NewMockStore()
	// Pre-populate store with a relay (LastHeartbeat must be recent to survive doHealthCheck)
	store.relays["pre-existing"] = &RelayInfo{
		ID: "pre-existing", URL: "wss://pre.com", Healthy: true, Capacity: 100,
		LastHeartbeat: time.Now(),
	}

	m := newTestManager(t, WithStore(store))

	// Manager should have loaded the relay from store
	r := m.GetRelayByID("pre-existing")
	if r == nil {
		t.Fatal("relay should be loaded from store on startup")
	}
	if r.URL != "wss://pre.com" {
		t.Errorf("relay URL mismatch: got %q, want %q", r.URL, "wss://pre.com")
	}
}

func TestLoadFromStoreError(t *testing.T) {
	failStore := &FailingLoadStore{}
	// Should not panic or fail — just logs warning
	m := newTestManager(t, WithStore(failStore))

	// Manager should still be usable with empty relay map
	if len(m.GetRelays()) != 0 {
		t.Error("should have no relays when store load fails")
	}
}

func TestHeartbeatSyncsToStore(t *testing.T) {
	store := NewMockStore()
	m := newTestManager(t, WithStore(store))

	if err := m.Register(&RelayInfo{ID: "relay-1", URL: "wss://r1.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Heartbeat should sync to store
	if err := m.Heartbeat("relay-1", 50, 25.0, 30.0); err != nil {
		t.Fatalf("Heartbeat failed: %v", err)
	}

	ctx := context.Background()
	stored, _ := store.GetRelay(ctx, "relay-1")
	if stored == nil {
		t.Fatal("relay should be in store")
	}
	if stored.LastHeartbeat.IsZero() {
		t.Error("store should have updated heartbeat time")
	}
}

func TestHeartbeatStoreFailureDoesNotBreakInMemory(t *testing.T) {
	failStore := &FailingHeartbeatStore{}
	failStore.relays = make(map[string]*RelayInfo)
	m := newTestManager(t, WithStore(failStore))

	if err := m.Register(&RelayInfo{ID: "relay-1", URL: "wss://r1.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Heartbeat should succeed in memory even though store fails
	if err := m.Heartbeat("relay-1", 50, 25.0, 30.0); err != nil {
		t.Fatalf("Heartbeat should not fail: %v", err)
	}

	r := m.GetRelayByID("relay-1")
	if r == nil || r.CurrentConnections != 50 {
		t.Error("in-memory state should still be updated")
	}
}
