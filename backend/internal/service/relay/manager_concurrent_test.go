package relay

import (
	"fmt"
	"sync"
	"testing"
)

func TestConcurrentRegisterAndHeartbeat(t *testing.T) {
	m := newTestManager(t)

	const numRelays = 20
	const numHeartbeats = 50

	// Pre-register relays
	for i := 0; i < numRelays; i++ {
		id := fmt.Sprintf("relay-%03d", i)
		if err := m.Register(&RelayInfo{ID: id, URL: "wss://r.com"}); err != nil {
			t.Fatalf("Register %s failed: %v", id, err)
		}
	}

	var wg sync.WaitGroup

	// Concurrent heartbeats on existing relays
	for i := 0; i < numHeartbeats; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("relay-%03d", n%numRelays)
			_ = m.Heartbeat(id, n, float64(n%100), float64(n%100))
		}(i)
	}

	// Concurrent re-registrations
	for i := 0; i < numRelays; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("relay-%03d", n)
			_ = m.Register(&RelayInfo{ID: id, URL: "wss://updated.com"})
		}(i)
	}

	wg.Wait()

	// Verify no data corruption
	relays := m.GetRelays()
	if len(relays) != numRelays {
		t.Errorf("expected %d relays, got %d", numRelays, len(relays))
	}
}

func TestConcurrentRegisterAndUnregister(t *testing.T) {
	m := newTestManager(t)

	const numRelays = 20
	var wg sync.WaitGroup

	// Concurrently register
	for i := 0; i < numRelays; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("relay-%03d", n)
			_ = m.Register(&RelayInfo{ID: id, URL: "wss://r.com"})
		}(i)
	}
	wg.Wait()

	// Concurrently unregister half and heartbeat the other half
	for i := 0; i < numRelays; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("relay-%03d", n)
			if n%2 == 0 {
				m.ForceUnregister(id)
			} else {
				_ = m.Heartbeat(id, 10, 20, 30)
			}
		}(i)
	}
	wg.Wait()

	// Should not panic; exact count depends on race ordering
	relays := m.GetRelays()
	if len(relays) > numRelays {
		t.Errorf("more relays than registered: %d", len(relays))
	}
}

func TestConcurrentRegisterWithStore(t *testing.T) {
	store := NewMockStore()
	m := newTestManager(t, WithStore(store))

	const numRelays = 20
	var wg sync.WaitGroup

	for i := 0; i < numRelays; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("relay-%03d", n)
			_ = m.Register(&RelayInfo{ID: id, URL: "wss://r.com"})
		}(i)
	}
	wg.Wait()

	relays := m.GetRelays()
	if len(relays) != numRelays {
		t.Errorf("expected %d relays, got %d", numRelays, len(relays))
	}
}
