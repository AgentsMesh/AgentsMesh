package aggregator

import (
	"bytes"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// NOTE: compressSpaces was removed because it doesn't work for TUI apps.
// CSI CUF (\x1b[nC) only moves the cursor - it does NOT overwrite existing content.
// TUI apps rely on spaces to clear old content during redraws.
// The correct solution requires a full VirtualTerminal implementation.
// Now we use serialize mode with VirtualTerminal.Serialize() for proper space compression.

// TestSmartAggregator_SerializeMode tests the serialize mode functionality
// where Write() only marks pending data and flushLocked() calls the serialize callback
func TestSmartAggregator_SerializeMode(t *testing.T) {
	// Simulated VirtualTerminal serialized output
	serializedOutput := []byte("Hello\x1b[5CWorld")
	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
		func() float64 { return 0.0 },
		WithSmartBaseDelay(10*time.Millisecond),
		// Serialize callback returns compressed data
		WithSerializeCallback(func() []byte {
			return serializedOutput
		}),
	)
	agg.SetRelayClient(relay)

	// Write with nil data - in serialize mode, data is ignored
	agg.Write(nil)

	// Wait for flush
	time.Sleep(50 * time.Millisecond)

	result := relay.getData()

	// Verify serialized output was sent
	if !bytes.Equal(result, serializedOutput) {
		t.Errorf("Expected serialized output %q, got %q", serializedOutput, result)
	}

	agg.Stop()
	t.Logf("Serialize mode test: correctly sent serialized output")
}

// TestSmartAggregator_SerializeModeNoPendingData tests that flush is skipped when no pending data
func TestSmartAggregator_SerializeModeNoPendingData(t *testing.T) {
	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
		func() float64 { return 0.0 },
		WithSmartBaseDelay(5*time.Millisecond),
		WithSerializeCallback(func() []byte {
			return []byte("should not be called if no pending data")
		}),
	)
	agg.SetRelayClient(relay)

	// Force flush without any Write() - should not flush
	agg.Flush()
	time.Sleep(20 * time.Millisecond)

	if len(relay.getData()) != 0 {
		t.Errorf("Expected 0 bytes flushed when no pending data, got %d", len(relay.getData()))
	}

	agg.Stop()
	t.Logf("Serialize mode no-pending-data test: correctly skipped flush")
}

// TestSmartAggregator_SerializeModeMultipleWrites tests aggregation with multiple writes
func TestSmartAggregator_SerializeModeMultipleWrites(t *testing.T) {
	var mu sync.Mutex
	var callbackCount int

	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
		func() float64 { return 0.0 },
		WithSmartBaseDelay(50*time.Millisecond),
		WithSerializeCallback(func() []byte {
			mu.Lock()
			callbackCount++
			mu.Unlock()
			return []byte("serialized")
		}),
	)
	agg.SetRelayClient(relay)

	// Multiple rapid writes should be aggregated
	for i := 0; i < 10; i++ {
		agg.Write(nil)
		time.Sleep(5 * time.Millisecond)
	}

	// Wait for flush
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	cc := callbackCount
	mu.Unlock()

	// Should have aggregated multiple writes into fewer flushes
	if cc == 0 {
		t.Errorf("Expected at least 1 callback, got 0")
	}
	if cc > 3 {
		t.Errorf("Expected aggregation, but got %d callbacks for 10 writes", cc)
	}

	agg.Stop()
	t.Logf("Serialize mode aggregation test: %d callbacks for 10 writes", cc)
}

// TestSmartAggregator_SerializeModeEmptyCallback tests handling of empty callback result
func TestSmartAggregator_SerializeModeEmptyCallback(t *testing.T) {
	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
		func() float64 { return 0.0 },
		WithSmartBaseDelay(5*time.Millisecond),
		// Callback returns empty data
		WithSerializeCallback(func() []byte {
			return nil
		}),
	)
	agg.SetRelayClient(relay)

	agg.Write(nil)
	time.Sleep(20 * time.Millisecond)

	// Should not send data when callback returns nil
	if len(relay.getData()) != 0 {
		t.Errorf("Expected 0 bytes when callback returns nil, got %d", len(relay.getData()))
	}

	agg.Stop()
	t.Logf("Serialize mode empty callback test: correctly skipped empty data")
}

// TestSmartAggregator_SerializeModeCriticalLoad tests serialize mode under critical load
func TestSmartAggregator_SerializeModeCriticalLoad(t *testing.T) {
	var usage atomic.Int64
	usage.Store(60) // 0.6 * 100 = 60, Critical load (stored as int to avoid float64 atomic issues)
	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
		func() float64 { return float64(usage.Load()) / 100.0 },
		WithSmartBaseDelay(10*time.Millisecond),
		WithSmartMaxDelay(50*time.Millisecond),
		WithSerializeCallback(func() []byte {
			return []byte("data")
		}),
	)
	agg.SetRelayClient(relay)

	// Write under critical load
	agg.Write(nil)

	// Wait for maxDelay
	time.Sleep(100 * time.Millisecond)

	// Lower usage
	usage.Store(0)

	// Wait for flush
	time.Sleep(100 * time.Millisecond)

	agg.Stop()
	t.Logf("Serialize mode critical load test completed, relay data: %d bytes", len(relay.getData()))
}
