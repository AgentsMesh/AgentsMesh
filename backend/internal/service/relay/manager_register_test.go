package relay

import (
	"errors"
	"fmt"
	"testing"
)

func TestRegisterCopiesInput(t *testing.T) {
	m := newTestManager(t)

	info := &RelayInfo{ID: "relay-1", URL: "wss://r1.com", Region: "us-east", Capacity: 100}
	if err := m.Register(info); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Mutating the original should NOT affect the stored relay
	info.Region = "mutated"
	info.Capacity = 999

	stored := m.GetRelayByID("relay-1")
	if stored == nil {
		t.Fatal("relay not found")
	}
	if stored.Region != "us-east" {
		t.Errorf("stored relay region mutated: got %q, want %q", stored.Region, "us-east")
	}
	if stored.Capacity != 100 {
		t.Errorf("stored relay capacity mutated: got %d, want 100", stored.Capacity)
	}
}

func TestRegisterCapacityLimit(t *testing.T) {
	m := newTestManager(t)

	// Fill to capacity
	for i := 0; i < maxRelayCount; i++ {
		id := fmt.Sprintf("relay-%04d", i)
		err := m.Register(&RelayInfo{ID: id, URL: "wss://r.com"})
		if err != nil {
			t.Fatalf("Register %d failed: %v", i, err)
		}
	}

	// One more should fail
	err := m.Register(&RelayInfo{ID: "relay-overflow", URL: "wss://overflow.com"})
	if err == nil {
		t.Error("Register should fail when capacity limit is reached")
	}
	if !errors.Is(err, ErrCapacityLimitReached) {
		t.Errorf("expected ErrCapacityLimitReached, got: %v", err)
	}
}

func TestRegisterCapacityLimitAllowsReRegistration(t *testing.T) {
	m := newTestManager(t)

	// Register one relay
	if err := m.Register(&RelayInfo{ID: "relay-1", URL: "wss://r.com"}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Re-registration of same ID should always succeed (even at capacity)
	if err := m.Register(&RelayInfo{ID: "relay-1", URL: "wss://r-updated.com"}); err != nil {
		t.Errorf("re-registration should succeed: %v", err)
	}

	// Verify update was applied
	r := m.GetRelayByID("relay-1")
	if r == nil || r.URL != "wss://r-updated.com" {
		t.Error("re-registration should update relay data")
	}
}
