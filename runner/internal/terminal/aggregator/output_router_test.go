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

func TestOutputRouter_NewOutputRouter(t *testing.T) {
	var received []byte
	or := NewOutputRouter(func(data []byte) {
		received = data
	})

	if or == nil {
		t.Fatal("NewOutputRouter should not return nil")
		return
	}

	// Route should use onFlush when no relay
	or.Route([]byte("test"))
	if string(received) != "test" {
		t.Errorf("Expected 'test', got '%s'", received)
	}
}

func TestOutputRouter_Route_EmptyData(t *testing.T) {
	var callCount int
	or := NewOutputRouter(func(data []byte) {
		callCount++
	})

	or.Route(nil)
	or.Route([]byte{})

	if callCount != 0 {
		t.Errorf("Route should not call callback for empty data, called %d times", callCount)
	}
}

func TestOutputRouter_Route_PrefersRelay(t *testing.T) {
	var grpcData []byte
	or := NewOutputRouter(func(data []byte) {
		grpcData = data
	})

	// Set relay client (connected)
	relay := newMockRelayWriter(true)
	or.SetRelayClient(relay)

	or.Route([]byte("test"))

	// Should use relay, not gRPC
	if grpcData != nil {
		t.Error("gRPC should not be called when relay is connected")
	}
	if string(relay.getData()) != "test" {
		t.Errorf("Relay should receive data, got '%s'", relay.getData())
	}
}

func TestOutputRouter_Route_FallbackToGRPC_WhenDisconnected(t *testing.T) {
	var grpcData []byte
	or := NewOutputRouter(func(data []byte) {
		grpcData = data
	})

	// Set relay client but disconnected
	relay := newMockRelayWriter(false)
	or.SetRelayClient(relay)

	or.Route([]byte("test"))

	// Should fall back to gRPC when relay is disconnected
	if string(grpcData) != "test" {
		t.Errorf("Should fallback to gRPC when relay disconnected, got '%s'", grpcData)
	}
	if len(relay.getData()) > 0 {
		t.Error("Disconnected relay should not receive data")
	}
}

func TestOutputRouter_Route_FallbackToGRPC_WhenCleared(t *testing.T) {
	var grpcData []byte
	or := NewOutputRouter(func(data []byte) {
		grpcData = data
	})

	// Set then clear relay
	relay := newMockRelayWriter(true)
	or.SetRelayClient(relay)
	or.SetRelayClient(nil)

	or.Route([]byte("test"))

	if string(grpcData) != "test" {
		t.Errorf("Should fallback to gRPC, got '%s'", grpcData)
	}
}

func TestOutputRouter_SetRelayClient(t *testing.T) {
	or := NewOutputRouter(nil)

	if or.HasRelayClient() {
		t.Error("Should not have relay initially")
	}

	relay := newMockRelayWriter(true)
	or.SetRelayClient(relay)

	if !or.HasRelayClient() {
		t.Error("Should have relay after SetRelayClient")
	}

	or.SetRelayClient(nil)

	if or.HasRelayClient() {
		t.Error("Should not have relay after setting nil")
	}
}

func TestOutputRouter_HasRelayClient(t *testing.T) {
	or := NewOutputRouter(nil)

	if or.HasRelayClient() {
		t.Error("HasRelayClient should be false initially")
	}

	relay := newMockRelayWriter(true)
	or.SetRelayClient(relay)

	if !or.HasRelayClient() {
		t.Error("HasRelayClient should be true after setting relay")
	}
}

func TestOutputRouter_SetOnFlush(t *testing.T) {
	or := NewOutputRouter(nil)

	var received []byte
	or.SetOnFlush(func(data []byte) {
		received = data
	})

	or.Route([]byte("test"))

	if string(received) != "test" {
		t.Errorf("New onFlush should be used, got '%s'", received)
	}
}
