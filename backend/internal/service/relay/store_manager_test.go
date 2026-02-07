package relay

import (
	"context"
	"testing"
)

func TestManagerWithStore(t *testing.T) {
	store := NewMockStore()
	m := NewManagerWithOptions(WithStore(store))

	if m == nil {
		t.Fatal("NewManagerWithOptions returned nil")
	}
	if m.store != store {
		t.Error("store not set")
	}
}

func TestRegisterPersistsToStore(t *testing.T) {
	store := NewMockStore()
	m := NewManagerWithOptions(WithStore(store))

	relay := &RelayInfo{ID: "relay-1", URL: "wss://r1.com", Region: "us-east"}
	if err := m.Register(relay); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Verify persisted to store
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

func TestUnregisterDeletesFromStore(t *testing.T) {
	store := NewMockStore()
	m := NewManagerWithOptions(WithStore(store))

	relay := &RelayInfo{ID: "relay-1", URL: "wss://r1.com"}
	if err := m.Register(relay); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Unregister
	m.Unregister("relay-1")

	// Verify removed from store
	ctx := context.Background()
	stored, _ := store.GetRelay(ctx, "relay-1")
	if stored != nil {
		t.Error("relay should be deleted from store")
	}
}

func TestForceUnregisterCleansUpStore(t *testing.T) {
	store := NewMockStore()
	m := NewManagerWithOptions(WithStore(store))

	relay := &RelayInfo{ID: "relay-1", URL: "wss://r1.com"}
	if err := m.Register(relay); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Force unregister
	m.ForceUnregister("relay-1")

	// Verify cleaned up in store
	ctx := context.Background()
	if r, _ := store.GetRelay(ctx, "relay-1"); r != nil {
		t.Error("relay should be deleted from store")
	}
}

func TestGracefulUnregisterCleansUpStore(t *testing.T) {
	store := NewMockStore()
	m := NewManagerWithOptions(WithStore(store))

	relay := &RelayInfo{ID: "relay-1", URL: "wss://r1.com"}
	if err := m.Register(relay); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Graceful unregister
	m.GracefulUnregister("relay-1", "shutdown")

	// Verify cleaned up in store
	ctx := context.Background()
	if r, _ := store.GetRelay(ctx, "relay-1"); r != nil {
		t.Error("relay should be deleted from store")
	}
}
