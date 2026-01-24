package terminal

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
	var mu sync.Mutex
	var flushedData []byte
	var flushCount int

	// Simulated VirtualTerminal serialized output
	serializedOutput := []byte("Hello\x1b[5CWorld")

	agg := NewSmartAggregator(
		func(data []byte) {
			mu.Lock()
			flushedData = append(flushedData, data...)
			flushCount++
			mu.Unlock()
		},
		func() float64 { return 0.0 },
		WithSmartBaseDelay(10*time.Millisecond),
		// Serialize callback returns compressed data
		WithSerializeCallback(func() []byte {
			return serializedOutput
		}),
	)

	// Write with nil data - in serialize mode, data is ignored
	agg.Write(nil)

	// Wait for flush
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	result := flushedData
	count := flushCount
	mu.Unlock()

	// Verify serialized output was sent
	if !bytes.Equal(result, serializedOutput) {
		t.Errorf("Expected serialized output %q, got %q", serializedOutput, result)
	}

	if count != 1 {
		t.Errorf("Expected 1 flush, got %d", count)
	}

	agg.Stop()
	t.Logf("✅ Serialize mode test: correctly sent serialized output")
}

// TestSmartAggregator_SerializeModeNoPendingData tests that flush is skipped when no pending data
func TestSmartAggregator_SerializeModeNoPendingData(t *testing.T) {
	var flushCount int

	agg := NewSmartAggregator(
		func(data []byte) {
			flushCount++
		},
		func() float64 { return 0.0 },
		WithSmartBaseDelay(5*time.Millisecond),
		WithSerializeCallback(func() []byte {
			return []byte("should not be called if no pending data")
		}),
	)

	// Force flush without any Write() - should not flush
	agg.Flush()

	if flushCount != 0 {
		t.Errorf("Expected 0 flushes when no pending data, got %d", flushCount)
	}

	agg.Stop()
	t.Logf("✅ Serialize mode no-pending-data test: correctly skipped flush")
}

// TestSmartAggregator_SerializeModeMultipleWrites tests aggregation with multiple writes
func TestSmartAggregator_SerializeModeMultipleWrites(t *testing.T) {
	var mu sync.Mutex
	var flushCount int
	var callbackCount int

	agg := NewSmartAggregator(
		func(data []byte) {
			mu.Lock()
			flushCount++
			mu.Unlock()
		},
		func() float64 { return 0.0 },
		WithSmartBaseDelay(50*time.Millisecond),
		WithSerializeCallback(func() []byte {
			mu.Lock()
			callbackCount++
			mu.Unlock()
			return []byte("serialized")
		}),
	)

	// Multiple rapid writes should be aggregated
	for i := 0; i < 10; i++ {
		agg.Write(nil)
		time.Sleep(5 * time.Millisecond)
	}

	// Wait for flush
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	fc := flushCount
	cc := callbackCount
	mu.Unlock()

	// Should have aggregated multiple writes into fewer flushes
	if fc == 0 {
		t.Errorf("Expected at least 1 flush, got 0")
	}
	if fc > 3 {
		t.Errorf("Expected aggregation, but got %d flushes for 10 writes", fc)
	}
	if cc != fc {
		t.Errorf("Callback count (%d) should match flush count (%d)", cc, fc)
	}

	agg.Stop()
	t.Logf("✅ Serialize mode aggregation test: %d flushes for 10 writes", fc)
}

// TestSmartAggregator_SerializeModeEmptyCallback tests handling of empty callback result
func TestSmartAggregator_SerializeModeEmptyCallback(t *testing.T) {
	var flushCount int

	agg := NewSmartAggregator(
		func(data []byte) {
			flushCount++
		},
		func() float64 { return 0.0 },
		WithSmartBaseDelay(5*time.Millisecond),
		// Callback returns empty data
		WithSerializeCallback(func() []byte {
			return nil
		}),
	)

	agg.Write(nil)
	time.Sleep(20 * time.Millisecond)

	// Should not call onFlush when callback returns empty data
	if flushCount != 0 {
		t.Errorf("Expected 0 flushes when callback returns nil, got %d", flushCount)
	}

	agg.Stop()
	t.Logf("✅ Serialize mode empty callback test: correctly skipped empty data")
}

// TestSmartAggregator_SerializeModeCriticalLoad tests serialize mode under critical load
func TestSmartAggregator_SerializeModeCriticalLoad(t *testing.T) {
	var usage atomic.Int64
	usage.Store(60) // 0.6 * 100 = 60, Critical load (stored as int to avoid float64 atomic issues)
	var flushCount atomic.Int32

	agg := NewSmartAggregator(
		func(data []byte) {
			flushCount.Add(1)
		},
		func() float64 { return float64(usage.Load()) / 100.0 },
		WithSmartBaseDelay(10*time.Millisecond),
		WithSmartMaxDelay(50*time.Millisecond),
		WithSerializeCallback(func() []byte {
			return []byte("data")
		}),
	)

	// Write under critical load
	agg.Write(nil)

	// Wait for maxDelay
	time.Sleep(100 * time.Millisecond)

	// Lower usage
	usage.Store(0)

	// Wait for flush
	time.Sleep(100 * time.Millisecond)

	agg.Stop()
	t.Logf("✅ Serialize mode critical load test completed, flushes: %d", flushCount.Load())
}
