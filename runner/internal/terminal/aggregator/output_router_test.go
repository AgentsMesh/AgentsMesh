package aggregator

import (
	"bytes"
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

func TestOutputRouter_NoCallbacks_BuffersEarlyOutput(t *testing.T) {
	or := NewOutputRouter(nil)

	// Should buffer data when no callbacks are set
	or.Route([]byte("error: invalid argument\n"))

	buf := or.DrainEarlyBuffer()
	if string(buf) != "error: invalid argument\n" {
		t.Errorf("Expected buffered early output, got '%s'", buf)
	}

	// After drain, buffer should be empty and done
	buf2 := or.DrainEarlyBuffer()
	if buf2 != nil {
		t.Errorf("Expected nil after second drain, got '%s'", buf2)
	}

	// Further routes should not buffer (earlyDone=true)
	or.Route([]byte("more data"))
	buf3 := or.DrainEarlyBuffer()
	if buf3 != nil {
		t.Errorf("Expected nil for post-drain route, got '%s'", buf3)
	}
}

func TestOutputRouter_EarlyBuffer_ReplayOnRelayConnect(t *testing.T) {
	or := NewOutputRouter(nil)

	// Buffer some early output
	or.Route([]byte("startup "))
	or.Route([]byte("output"))

	// When relay connects, buffered data should be replayed
	relay := newMockRelayWriter(true)
	or.SetRelayClient(relay)

	if string(relay.getData()) != "startup output" {
		t.Errorf("Expected replayed 'startup output', got '%s'", relay.getData())
	}

	// Subsequent routes go directly through relay
	or.Route([]byte(" live"))
	if string(relay.getData()) != "startup output live" {
		t.Errorf("Expected 'startup output live', got '%s'", relay.getData())
	}
}

func TestOutputRouter_EarlyBuffer_ReplayOnFlushSet(t *testing.T) {
	or := NewOutputRouter(nil)

	// Buffer some early output
	or.Route([]byte("buffered data"))

	// When onFlush is set, buffered data should be replayed
	var flushReceived []byte
	or.SetOnFlush(func(data []byte) {
		flushReceived = append(flushReceived, data...)
	})

	if string(flushReceived) != "buffered data" {
		t.Errorf("Expected replayed 'buffered data', got '%s'", flushReceived)
	}
}

func TestOutputRouter_EarlyBuffer_MaxSize(t *testing.T) {
	or := NewOutputRouter(nil)

	// Fill beyond max buffer size
	bigData := bytes.Repeat([]byte("x"), earlyBufferMaxSize+1000)
	or.Route(bigData)

	buf := or.DrainEarlyBuffer()
	if len(buf) != earlyBufferMaxSize {
		t.Errorf("Expected buffer capped at %d, got %d", earlyBufferMaxSize, len(buf))
	}
}

func TestOutputRouter_EarlyBuffer_NotUsedWhenCallbackSet(t *testing.T) {
	var received []byte
	or := NewOutputRouter(func(data []byte) {
		received = append(received, data...)
	})

	// With onFlush set, data should go directly, not buffer
	or.Route([]byte("direct"))

	if string(received) != "direct" {
		t.Errorf("Expected 'direct', got '%s'", received)
	}

	// Early buffer should be empty
	buf := or.DrainEarlyBuffer()
	if buf != nil {
		t.Errorf("Expected nil early buffer when callback is set, got '%s'", buf)
	}
}

func TestOutputRouter_Concurrent(t *testing.T) {
	// Track total bytes across ALL destinations (gRPC + relay).
	// When a connected relay is set, data flows to relay.SendOutput instead of gRPC callback,
	// so we must count both to verify no data is lost.
	var totalBytes atomic.Int64

	or := NewOutputRouter(func(data []byte) {
		totalBytes.Add(int64(len(data)))
	})

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

	// Concurrent relay updates — relay also counts bytes via SendOutput
	go func() {
		for i := 0; i < 50; i++ {
			relay := &countingRelayWriter{totalBytes: &totalBytes}
			relay.connected.Store(true)
			or.SetRelayClient(relay)
			or.SetRelayClient(nil)
		}
	}()

	wg.Wait()

	got := totalBytes.Load()
	if got != 1000 {
		t.Errorf("Expected 1000 bytes routed, got %d", got)
	}
}

// countingRelayWriter is a RelayWriter that adds byte counts to a shared counter.
type countingRelayWriter struct {
	connected  atomic.Bool
	totalBytes *atomic.Int64
}

func (c *countingRelayWriter) SendOutput(data []byte) error {
	c.totalBytes.Add(int64(len(data)))
	return nil
}

func (c *countingRelayWriter) IsConnected() bool {
	return c.connected.Load()
}

func TestOutputRouter_LargeData(t *testing.T) {
	var received []byte
	or := NewOutputRouter(func(data []byte) {
		received = data
	})

	// Test with large data
	largeData := bytes.Repeat([]byte("x"), 1024*1024) // 1MB
	or.Route(largeData)

	if len(received) != len(largeData) {
		t.Errorf("Expected %d bytes, got %d", len(largeData), len(received))
	}
}

func TestOutputRouter_RelayDisconnectAutoFallback(t *testing.T) {
	// Core test for the deadlock fix: when relay disconnects,
	// output should automatically fall back to gRPC
	var grpcData []byte
	or := NewOutputRouter(func(data []byte) {
		grpcData = append(grpcData, data...)
	})

	relay := newMockRelayWriter(true)
	or.SetRelayClient(relay)

	// Send while connected — should go to relay
	or.Route([]byte("connected"))
	if string(relay.getData()) != "connected" {
		t.Errorf("Expected relay to receive 'connected', got '%s'", relay.getData())
	}
	if grpcData != nil {
		t.Error("gRPC should not receive data while relay is connected")
	}

	// Simulate disconnect (relay client stays registered but disconnected)
	relay.connected.Store(false)

	// Send while disconnected — should fall back to gRPC
	or.Route([]byte("disconnected"))
	if string(grpcData) != "disconnected" {
		t.Errorf("Expected gRPC fallback to receive 'disconnected', got '%s'", grpcData)
	}

	// Simulate reconnect
	relay.connected.Store(true)

	grpcData = nil
	or.Route([]byte("reconnected"))
	if string(relay.getData()) != "connectedreconnected" {
		t.Errorf("Expected relay to receive 'reconnected', got '%s'", relay.getData())
	}
	if grpcData != nil {
		t.Error("gRPC should not receive data after relay reconnects")
	}
}

func TestOutputRouter_StaleClientCannotIntercept(t *testing.T) {
	// Simulates the original bug scenario: old client's callback should not
	// intercept output after a new client is set
	var grpcData []byte
	or := NewOutputRouter(func(data []byte) {
		grpcData = append(grpcData, data...)
	})

	// Old client
	oldRelay := newMockRelayWriter(true)
	or.SetRelayClient(oldRelay)

	// Replace with new client
	newRelay := newMockRelayWriter(true)
	or.SetRelayClient(newRelay)

	// Old client is stopped (disconnected)
	oldRelay.connected.Store(false)

	// Output should go to new client, not old
	or.Route([]byte("test"))
	if len(oldRelay.getData()) > 0 {
		// Old relay might have data from before replacement, but not "test"
		// After SetRelayClient(newRelay), the router holds newRelay reference
	}
	if string(newRelay.getData()) != "test" {
		t.Errorf("Expected new relay to receive 'test', got '%s'", newRelay.getData())
	}
}
