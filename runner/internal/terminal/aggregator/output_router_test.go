package aggregator

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockRelayWriter implements RelayWriter for testing.
type mockRelayWriter struct {
	mu        sync.Mutex
	data      []byte
	connected atomic.Bool
	sendErr   error
	calls     atomic.Int32
}

func newMockRelayWriter(connected bool) *mockRelayWriter {
	m := &mockRelayWriter{}
	m.connected.Store(connected)
	return m
}

func (m *mockRelayWriter) SendOutput(data []byte) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.calls.Add(1)
	m.mu.Lock()
	m.data = append(m.data, data...)
	m.mu.Unlock()
	return nil
}

func (m *mockRelayWriter) FlushOutput(context.Context) error { return nil }

func (m *mockRelayWriter) IsConnected() bool {
	return m.connected.Load()
}

func (m *mockRelayWriter) getData() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]byte(nil), m.data...)
}

func (m *mockRelayWriter) sendCount() int32 {
	return m.calls.Load()
}

func waitForRouter(t *testing.T, router *OutputRouter) {
	t.Helper()
	if !router.barrier().wait(time.Second) {
		t.Fatal("output router did not drain")
	}
}

func TestOutputRouter_New(t *testing.T) {
	or := NewOutputRouter()
	if or == nil {
		t.Fatal("NewOutputRouter should not return nil")
	}
	if len(or.Health()) != 0 {
		t.Error("should not have relay initially")
	}
}

func TestOutputRouter_Route_EmptyData(t *testing.T) {
	or := NewOutputRouter()
	relay := newMockRelayWriter(true)
	or.SetDestination(OutputDestinationCloud, relay)

	or.enqueue(nil)
	or.enqueue([]byte{})

	if len(relay.getData()) != 0 {
		t.Error("Route should not send empty data")
	}
}

func TestOutputRouter_Route_SendsToRelay(t *testing.T) {
	or := NewOutputRouter()
	relay := newMockRelayWriter(true)
	or.SetDestination(OutputDestinationCloud, relay)

	or.enqueue([]byte("hello"))
	waitForRouter(t, or)
	if string(relay.getData()) != "hello" {
		t.Errorf("Expected 'hello', got '%s'", relay.getData())
	}
}

func TestOutputRouter_Route_DropsWhenNoRelay(t *testing.T) {
	or := NewOutputRouter()
	// No relay set — data should be silently dropped (no panic)
	or.enqueue([]byte("dropped"))
}

func TestOutputRouter_Route_DropsWhenDisconnected(t *testing.T) {
	or := NewOutputRouter()
	relay := newMockRelayWriter(false) // disconnected
	or.SetDestination(OutputDestinationCloud, relay)

	or.enqueue([]byte("dropped"))
	waitForRouter(t, or)
	if len(relay.getData()) > 0 {
		t.Error("disconnected relay should not receive data")
	}
	if health := or.Health(); len(health) != 1 || health[0].Desynced {
		t.Fatalf("inactive destination should remain healthy: %+v", health)
	}
}

func TestOutputRouter_SetDestination(t *testing.T) {
	or := NewOutputRouter()

	relay := newMockRelayWriter(true)
	or.SetDestination(OutputDestinationCloud, relay)
	if len(or.Health()) == 0 {
		t.Error("should have relay after SetDestination")
	}

	or.RemoveDestination(OutputDestinationCloud)
	if len(or.Health()) != 0 {
		t.Error("should not have relay after clearing")
	}
}

func TestOutputRouter_RelayDisconnectAndReconnect(t *testing.T) {
	or := NewOutputRouter()
	relay := newMockRelayWriter(true)
	or.SetDestination(OutputDestinationCloud, relay)

	// Connected — data goes to relay
	or.enqueue([]byte("a"))
	waitForRouter(t, or)
	if string(relay.getData()) != "a" {
		t.Errorf("Expected 'a', got '%s'", relay.getData())
	}

	// Disconnect — data is dropped
	relay.connected.Store(false)
	or.enqueue([]byte("dropped"))
	waitForRouter(t, or)
	if string(relay.getData()) != "a" {
		t.Error("disconnected relay should not receive new data")
	}

	// Reconnect installs a fresh generation before deltas flow again.
	relay.connected.Store(true)
	or.SetDestination(OutputDestinationCloud, relay)
	or.enqueue([]byte("b"))
	waitForRouter(t, or)
	if string(relay.getData()) != "ab" {
		t.Errorf("Expected 'ab', got '%s'", relay.getData())
	}
}

func TestOutputRouter_StaleClientReplacement(t *testing.T) {
	or := NewOutputRouter()

	oldRelay := newMockRelayWriter(true)
	or.SetDestination(OutputDestinationCloud, oldRelay)

	newRelay := newMockRelayWriter(true)
	or.SetDestination(OutputDestinationCloud, newRelay)

	oldRelay.connected.Store(false)

	or.enqueue([]byte("test"))
	waitForRouter(t, or)
	if string(newRelay.getData()) != "test" {
		t.Errorf("Expected new relay to receive 'test', got '%s'", newRelay.getData())
	}
}

func TestOutputRouter_Concurrent(t *testing.T) {
	or := NewOutputRouter()
	relay := newMockRelayWriter(true)
	or.SetDestination(OutputDestinationCloud, relay)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				or.enqueue([]byte("x"))
			}
		}()
	}

	// Concurrent relay swaps
	go func() {
		for i := 0; i < 50; i++ {
			r := newMockRelayWriter(true)
			or.SetDestination(OutputDestinationCloud, r)
		}
		or.SetDestination(OutputDestinationCloud, relay) // restore original
	}()

	wg.Wait()
	// No race/panic is the success criterion
}
