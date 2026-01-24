package terminal

import (
	"bytes"
	"sync"
	"testing"
)

func TestOutputRouter_NewOutputRouter(t *testing.T) {
	var received []byte
	or := NewOutputRouter(func(data []byte) {
		received = data
	})

	if or == nil {
		t.Fatal("NewOutputRouter should not return nil")
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
	var grpcData, relayData []byte
	or := NewOutputRouter(func(data []byte) {
		grpcData = data
	})

	// Set relay output
	or.SetRelayOutput(func(data []byte) {
		relayData = data
	})

	or.Route([]byte("test"))

	// Should use relay, not gRPC
	if grpcData != nil {
		t.Error("gRPC should not be called when relay is set")
	}
	if string(relayData) != "test" {
		t.Errorf("Relay should receive data, got '%s'", relayData)
	}
}

func TestOutputRouter_Route_FallbackToGRPC(t *testing.T) {
	var grpcData []byte
	or := NewOutputRouter(func(data []byte) {
		grpcData = data
	})

	// Set then clear relay
	or.SetRelayOutput(func(data []byte) {})
	or.SetRelayOutput(nil)

	or.Route([]byte("test"))

	if string(grpcData) != "test" {
		t.Errorf("Should fallback to gRPC, got '%s'", grpcData)
	}
}

func TestOutputRouter_SetRelayOutput(t *testing.T) {
	or := NewOutputRouter(nil)

	if or.HasRelayOutput() {
		t.Error("Should not have relay initially")
	}

	or.SetRelayOutput(func(data []byte) {})

	if !or.HasRelayOutput() {
		t.Error("Should have relay after SetRelayOutput")
	}

	or.SetRelayOutput(nil)

	if or.HasRelayOutput() {
		t.Error("Should not have relay after setting nil")
	}
}

func TestOutputRouter_GetRelayOutput(t *testing.T) {
	or := NewOutputRouter(nil)

	if or.GetRelayOutput() != nil {
		t.Error("GetRelayOutput should return nil initially")
	}

	relay := func(data []byte) {}
	or.SetRelayOutput(relay)

	if or.GetRelayOutput() == nil {
		t.Error("GetRelayOutput should return the relay function")
	}
}

func TestOutputRouter_HasRelayOutput(t *testing.T) {
	or := NewOutputRouter(nil)

	if or.HasRelayOutput() {
		t.Error("HasRelayOutput should be false initially")
	}

	or.SetRelayOutput(func(data []byte) {})

	if !or.HasRelayOutput() {
		t.Error("HasRelayOutput should be true after setting relay")
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

func TestOutputRouter_NoCallbacks(t *testing.T) {
	or := NewOutputRouter(nil)

	// Should not panic with no callbacks
	or.Route([]byte("test"))
}

func TestOutputRouter_Concurrent(t *testing.T) {
	var mu sync.Mutex
	var totalBytes int

	or := NewOutputRouter(func(data []byte) {
		mu.Lock()
		totalBytes += len(data)
		mu.Unlock()
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

	// Concurrent relay updates
	go func() {
		for i := 0; i < 50; i++ {
			or.SetRelayOutput(func(data []byte) {
				mu.Lock()
				totalBytes += len(data)
				mu.Unlock()
			})
			or.SetRelayOutput(nil)
		}
	}()

	wg.Wait()

	mu.Lock()
	if totalBytes != 1000 {
		t.Errorf("Expected 1000 bytes routed, got %d", totalBytes)
	}
	mu.Unlock()
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
