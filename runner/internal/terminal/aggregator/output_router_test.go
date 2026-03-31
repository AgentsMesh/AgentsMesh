package aggregator

import (
	"sync"
	"sync/atomic"
	"testing"
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

func TestOutputRouter_New(t *testing.T) {
	or := NewOutputRouter()
	if or == nil {
		t.Fatal("NewOutputRouter should not return nil")
	}
	if or.HasRelayClient() {
		t.Error("should not have relay initially")
	}
}

func TestOutputRouter_Route_EmptyData(t *testing.T) {
	or := NewOutputRouter()
	relay := newMockRelayWriter(true)
	or.SetRelayClient(relay)

	or.Route(nil)
	or.Route([]byte{})

	if len(relay.getData()) != 0 {
		t.Error("Route should not send empty data")
	}
}

func TestOutputRouter_Route_SendsToRelay(t *testing.T) {
	or := NewOutputRouter()
	relay := newMockRelayWriter(true)
	or.SetRelayClient(relay)

	or.Route([]byte("hello"))
	if string(relay.getData()) != "hello" {
		t.Errorf("Expected 'hello', got '%s'", relay.getData())
	}
}

func TestOutputRouter_Route_DropsWhenNoRelay(t *testing.T) {
	or := NewOutputRouter()
	// No relay set — data should be silently dropped (no panic)
	or.Route([]byte("dropped"))
}

func TestOutputRouter_Route_DropsWhenDisconnected(t *testing.T) {
	or := NewOutputRouter()
	relay := newMockRelayWriter(false) // disconnected
	or.SetRelayClient(relay)

	or.Route([]byte("dropped"))
	if len(relay.getData()) > 0 {
		t.Error("disconnected relay should not receive data")
	}
}

func TestOutputRouter_SetRelayClient(t *testing.T) {
	or := NewOutputRouter()

	relay := newMockRelayWriter(true)
	or.SetRelayClient(relay)
	if !or.HasRelayClient() {
		t.Error("should have relay after SetRelayClient")
	}

	or.SetRelayClient(nil)
	if or.HasRelayClient() {
		t.Error("should not have relay after clearing")
	}
}

func TestOutputRouter_RelayDisconnectAndReconnect(t *testing.T) {
	or := NewOutputRouter()
	relay := newMockRelayWriter(true)
	or.SetRelayClient(relay)

	// Connected — data goes to relay
	or.Route([]byte("a"))
	if string(relay.getData()) != "a" {
		t.Errorf("Expected 'a', got '%s'", relay.getData())
	}

	// Disconnect — data is dropped
	relay.connected.Store(false)
	or.Route([]byte("dropped"))
	if string(relay.getData()) != "a" {
		t.Error("disconnected relay should not receive new data")
	}

	// Reconnect — data flows again
	relay.connected.Store(true)
	or.Route([]byte("b"))
	if string(relay.getData()) != "ab" {
		t.Errorf("Expected 'ab', got '%s'", relay.getData())
	}
}

func TestOutputRouter_StaleClientReplacement(t *testing.T) {
	or := NewOutputRouter()

	oldRelay := newMockRelayWriter(true)
	or.SetRelayClient(oldRelay)

	newRelay := newMockRelayWriter(true)
	or.SetRelayClient(newRelay)

	oldRelay.connected.Store(false)

	or.Route([]byte("test"))
	if string(newRelay.getData()) != "test" {
		t.Errorf("Expected new relay to receive 'test', got '%s'", newRelay.getData())
	}
}

func TestOutputRouter_Concurrent(t *testing.T) {
	or := NewOutputRouter()
	relay := newMockRelayWriter(true)
	or.SetRelayClient(relay)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				or.Route([]byte("x"))
			}
		}()
	}

	// Concurrent relay swaps
	go func() {
		for i := 0; i < 50; i++ {
			r := newMockRelayWriter(true)
			or.SetRelayClient(r)
		}
		or.SetRelayClient(relay) // restore original
	}()

	wg.Wait()
	// No race/panic is the success criterion
}
