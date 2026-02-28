package aggregator

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
	var relayReceived []byte
	or.SetRelayOutput(func(data []byte) {
		relayReceived = append(relayReceived, data...)
	})

	if string(relayReceived) != "startup output" {
		t.Errorf("Expected replayed 'startup output', got '%s'", relayReceived)
	}

	// Subsequent routes go directly through relay
	or.Route([]byte(" live"))
	if string(relayReceived) != "startup output live" {
		t.Errorf("Expected 'startup output live', got '%s'", relayReceived)
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
