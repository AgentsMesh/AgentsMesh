package terminal

import (
	"bytes"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSmartAggregator_BasicAggregation(t *testing.T) {
	var received []byte
	var mu sync.Mutex
	done := make(chan struct{})

	agg := NewSmartAggregator(
		func(data []byte) {
			mu.Lock()
			received = append(received, data...)
			mu.Unlock()
			select {
			case done <- struct{}{}:
			default:
			}
		},
		func() float64 { return 0 }, // No queue pressure
		WithSmartBaseDelay(10*time.Millisecond),
	)

	// Write some data
	agg.Write([]byte("hello"))
	agg.Write([]byte(" world"))

	// Wait for flush
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for flush")
	}

	mu.Lock()
	defer mu.Unlock()
	if string(received) != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", string(received))
	}
}

func TestSmartAggregator_AdaptiveDelay(t *testing.T) {
	agg := NewSmartAggregator(
		func(data []byte) {},
		func() float64 { return 0 },
		WithSmartBaseDelay(16*time.Millisecond),
		WithSmartMaxDelay(200*time.Millisecond),
	)
	defer agg.Stop()

	tests := []struct {
		usage    float64
		expected time.Duration
	}{
		{0.0, 16 * time.Millisecond},   // No load: base delay
		{0.5, 64 * time.Millisecond},   // 50% load: 16 * (1 + 0.25*12) = 16 * 4 = 64
		{0.8, 124 * time.Millisecond},  // 80% load: 16 * (1 + 0.64*12) = 16 * 8.68 ≈ 139 (capped calculation)
		{1.0, 200 * time.Millisecond},  // 100% load: capped at maxDelay
	}

	for _, tc := range tests {
		delay := agg.calculateDelay(tc.usage)
		// Allow 20% tolerance for rounding
		minExpected := time.Duration(float64(tc.expected) * 0.8)
		maxExpected := time.Duration(float64(tc.expected) * 1.2)
		if tc.usage == 1.0 {
			// For max load, should be exactly maxDelay
			if delay != tc.expected {
				t.Errorf("usage=%.1f: expected %v, got %v", tc.usage, tc.expected, delay)
			}
		} else if delay < minExpected || delay > maxExpected {
			t.Errorf("usage=%.1f: expected ~%v, got %v", tc.usage, tc.expected, delay)
		}
	}
}

func TestSmartAggregator_DiscardOldFrames(t *testing.T) {
	var received []byte
	var mu sync.Mutex
	done := make(chan struct{}, 10)

	agg := NewSmartAggregator(
		func(data []byte) {
			mu.Lock()
			received = data
			mu.Unlock()
			done <- struct{}{}
		},
		func() float64 { return 0.3 }, // Moderate pressure - allows flush but triggers frame discard
		WithSmartBaseDelay(10*time.Millisecond),
	)
	defer agg.Stop()

	// Write data with clear screen sequence in the middle
	// old frame + clear screen + new frame
	agg.Write([]byte("old content"))
	agg.Write([]byte("\x1b[2J")) // Clear screen
	agg.Write([]byte("new content"))

	// Wait for flush
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout waiting for flush")
	}

	mu.Lock()
	defer mu.Unlock()

	// Should only contain content from clear screen onwards
	expected := "\x1b[2Jnew content"
	if string(received) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(received))
	}
}

// TestSmartAggregator_SynchronizedOutputFrameBoundary tests frame boundary detection
// using Synchronized Output sequences (ESC[?2026h / ESC[?2026l) which is used by
// Claude Code and modern TUI frameworks.
func TestSmartAggregator_SynchronizedOutputFrameBoundary(t *testing.T) {
	var received []byte
	var mu sync.Mutex
	done := make(chan struct{}, 10)

	agg := NewSmartAggregator(
		func(data []byte) {
			mu.Lock()
			received = data
			mu.Unlock()
			done <- struct{}{}
		},
		func() float64 { return 0.3 }, // Moderate pressure - triggers frame discard
		WithSmartBaseDelay(10*time.Millisecond),
	)
	defer agg.Stop()

	// Simulate Claude Code output pattern:
	// Frame 1: ESC[?2026h ... content1 ... ESC[?2026l
	// Frame 2: ESC[?2026h ... content2 ... ESC[?2026l
	// Only Frame 2 should be kept (from the last frame START onwards)

	syncStart := "\x1b[?2026h"
	syncEnd := "\x1b[?2026l"

	// Write Frame 1 (old frame - should be discarded)
	agg.Write([]byte(syncStart + "old frame content" + syncEnd))
	// Write Frame 2 (new frame - should be kept)
	agg.Write([]byte(syncStart + "new frame content" + syncEnd))

	// Wait for flush
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout waiting for flush")
	}

	mu.Lock()
	defer mu.Unlock()

	// Should contain the complete Frame 2 (from last ESC[?2026h onwards)
	if !bytes.Contains(received, []byte(syncStart)) {
		t.Errorf("Expected sync output start sequence in result, got '%s'", string(received))
	}
	if !bytes.Contains(received, []byte(syncEnd)) {
		t.Errorf("Expected sync output end sequence in result, got '%s'", string(received))
	}
	if !bytes.Contains(received, []byte("new frame content")) {
		t.Errorf("Expected 'new frame content' in result, got '%s'", string(received))
	}
	// Old frame should be discarded
	if bytes.Contains(received, []byte("old frame content")) {
		t.Errorf("Old frame should be discarded, but got '%s'", string(received))
	}
}

// TestSmartAggregator_SyncOutputPriorityOverClearScreen tests that Synchronized Output
// (ESC[?2026h) takes priority over Clear Screen (ESC[2J) for frame boundary detection.
func TestSmartAggregator_SyncOutputPriorityOverClearScreen(t *testing.T) {
	var received []byte
	var mu sync.Mutex
	done := make(chan struct{}, 10)

	agg := NewSmartAggregator(
		func(data []byte) {
			mu.Lock()
			received = data
			mu.Unlock()
			done <- struct{}{}
		},
		func() float64 { return 0.3 },
		WithSmartBaseDelay(10*time.Millisecond),
	)
	defer agg.Stop()

	// Mix of sync output and clear screen sequences
	// Sync output start should take priority
	syncStart := "\x1b[?2026h"
	syncEnd := "\x1b[?2026l"
	clearScreen := "\x1b[2J"

	// Pattern: clear_screen -> old_content -> sync_start -> new_content -> sync_end
	// Should keep from sync_start onwards (not from clear_screen)
	agg.Write([]byte(clearScreen + "after clear"))
	agg.Write([]byte(syncStart + "sync frame" + syncEnd))

	// Wait for flush
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout waiting for flush")
	}

	mu.Lock()
	defer mu.Unlock()

	// Should contain complete sync frame
	if !bytes.Contains(received, []byte(syncStart)) {
		t.Errorf("Expected sync output start sequence, got '%s'", string(received))
	}
	if !bytes.Contains(received, []byte("sync frame")) {
		t.Errorf("Expected 'sync frame' in result, got '%s'", string(received))
	}
	// Content before sync start should be discarded (sync output takes priority)
	if bytes.Contains(received, []byte("after clear")) {
		t.Errorf("Content before sync start should be discarded, got '%s'", string(received))
	}
}

func TestSmartAggregator_MaxSizeFlush(t *testing.T) {
	var flushCount int32
	done := make(chan struct{}, 10)

	agg := NewSmartAggregator(
		func(data []byte) {
			atomic.AddInt32(&flushCount, 1)
			done <- struct{}{}
		},
		func() float64 { return 0 },
		WithSmartMaxSize(100), // Small max size for testing
		WithSmartBaseDelay(1*time.Second), // Long delay so timer doesn't interfere
	)
	defer agg.Stop()

	// Write data exceeding max size
	data := bytes.Repeat([]byte("x"), 150)
	agg.Write(data)

	// Should flush immediately due to max size
	select {
	case <-done:
	case <-time.After(50 * time.Millisecond):
		t.Fatal("Expected immediate flush on max size exceeded")
	}

	count := atomic.LoadInt32(&flushCount)
	if count < 1 {
		t.Errorf("Expected at least 1 flush, got %d", count)
	}
}

func TestSmartAggregator_Stop(t *testing.T) {
	var received []byte
	var mu sync.Mutex
	done := make(chan struct{}, 1)

	agg := NewSmartAggregator(
		func(data []byte) {
			mu.Lock()
			received = data
			mu.Unlock()
			select {
			case done <- struct{}{}:
			default:
			}
		},
		func() float64 { return 0 },
		WithSmartBaseDelay(1*time.Second), // Long delay
	)

	// Write data
	agg.Write([]byte("pending data"))

	// Stop should flush immediately
	agg.Stop()

	// Wait a bit for the goroutine
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected flush on stop")
	}

	mu.Lock()
	defer mu.Unlock()
	if string(received) != "pending data" {
		t.Errorf("Expected 'pending data', got '%s'", string(received))
	}

	// Subsequent writes should be ignored
	agg.Write([]byte("ignored"))
	if agg.BufferLen() != 0 {
		t.Error("Buffer should be empty after stop")
	}
}

func TestSmartAggregator_ConcurrentWrites(t *testing.T) {
	var totalBytes int64
	var mu sync.Mutex

	agg := NewSmartAggregator(
		func(data []byte) {
			mu.Lock()
			totalBytes += int64(len(data))
			mu.Unlock()
		},
		func() float64 { return 0 },
		WithSmartBaseDelay(5*time.Millisecond),
	)

	// Concurrent writes
	var wg sync.WaitGroup
	numWriters := 10
	bytesPerWriter := 1000

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < bytesPerWriter; j++ {
				agg.Write([]byte("x"))
			}
		}()
	}

	wg.Wait()
	agg.Stop()

	// Give some time for final flush
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	expected := int64(numWriters * bytesPerWriter)
	if totalBytes != expected {
		t.Errorf("Expected %d bytes, got %d", expected, totalBytes)
	}
}

func TestSmartAggregator_NilQueueUsageFn(t *testing.T) {
	done := make(chan struct{})

	// Test with nil queueUsageFn (should not panic)
	agg := NewSmartAggregator(
		func(data []byte) {
			close(done)
		},
		nil, // nil queue usage function
		WithSmartBaseDelay(10*time.Millisecond),
	)

	agg.Write([]byte("test"))

	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for flush")
	}

	agg.Stop()
}

func TestSmartAggregator_Flush(t *testing.T) {
	var received []byte
	var mu sync.Mutex
	done := make(chan struct{}, 1)

	agg := NewSmartAggregator(
		func(data []byte) {
			mu.Lock()
			received = data
			mu.Unlock()
			select {
			case done <- struct{}{}:
			default:
			}
		},
		func() float64 { return 0 },
		WithSmartBaseDelay(1*time.Second), // Long delay
	)
	defer agg.Stop()

	// Write data
	agg.Write([]byte("data"))

	// Manual flush
	agg.Flush()

	// Should flush immediately
	select {
	case <-done:
	case <-time.After(50 * time.Millisecond):
		t.Fatal("Expected immediate flush")
	}

	mu.Lock()
	defer mu.Unlock()
	if string(received) != "data" {
		t.Errorf("Expected 'data', got '%s'", string(received))
	}
}

func TestSmartAggregator_IsStopped(t *testing.T) {
	agg := NewSmartAggregator(
		func(data []byte) {},
		func() float64 { return 0 },
	)

	if agg.IsStopped() {
		t.Error("Should not be stopped initially")
	}

	agg.Stop()

	if !agg.IsStopped() {
		t.Error("Should be stopped after Stop()")
	}

	// Double stop should not panic
	agg.Stop()
}

func TestSmartAggregator_BufferLimitEnforced(t *testing.T) {
	// Test that buffer never exceeds maxSize even without clear screen signals
	// This simulates `find /` style output (no ESC[2J)
	maxSize := 1000
	var totalFlushed int64
	var mu sync.Mutex

	agg := NewSmartAggregator(
		func(data []byte) {
			mu.Lock()
			totalFlushed += int64(len(data))
			mu.Unlock()
		},
		func() float64 { return 0.9 }, // High pressure - delays flush
		WithSmartMaxSize(maxSize),
		WithSmartBaseDelay(50*time.Millisecond),
	)

	// Write much more data than maxSize without any clear screen
	totalWritten := 0
	for i := 0; i < 100; i++ {
		chunk := bytes.Repeat([]byte("x"), 200) // 200 bytes per write
		agg.Write(chunk)
		totalWritten += len(chunk)

		// Buffer should never exceed maxSize
		if agg.BufferLen() > maxSize {
			t.Errorf("Buffer exceeded maxSize: %d > %d", agg.BufferLen(), maxSize)
		}
	}

	agg.Stop()

	t.Logf("✅ Buffer limit test: wrote %d bytes, buffer never exceeded %d",
		totalWritten, maxSize)
}

func TestSmartAggregator_BufferLimitWithClearScreen(t *testing.T) {
	// Test that clear screen detection still works with buffer limit
	maxSize := 500
	var lastFlush []byte
	var mu sync.Mutex

	agg := NewSmartAggregator(
		func(data []byte) {
			mu.Lock()
			lastFlush = make([]byte, len(data))
			copy(lastFlush, data)
			mu.Unlock()
		},
		func() float64 { return 0.5 }, // Medium pressure
		WithSmartMaxSize(maxSize),
		WithSmartBaseDelay(10*time.Millisecond),
	)

	// Write old frame + clear screen + new frame
	agg.Write(bytes.Repeat([]byte("old"), 100)) // 300 bytes
	agg.Write([]byte("\x1b[2J"))                 // Clear screen
	agg.Write([]byte("new frame content"))       // New frame

	// Wait for flush
	time.Sleep(50 * time.Millisecond)
	agg.Stop()

	mu.Lock()
	defer mu.Unlock()

	// Should contain the clear screen and new content
	if !bytes.Contains(lastFlush, []byte("\x1b[2J")) {
		t.Error("Clear screen should be preserved")
	}
	if !bytes.Contains(lastFlush, []byte("new frame content")) {
		t.Error("New frame content should be preserved")
	}
	// Old content should be discarded
	if bytes.Contains(lastFlush, []byte("oldoldold")) {
		t.Error("Old frame content should be discarded")
	}
}

func TestSmartAggregator_LargeChunkExceedsMaxSize(t *testing.T) {
	// Test behavior when a single write exceeds maxSize
	maxSize := 100
	var flushed []byte
	var mu sync.Mutex

	agg := NewSmartAggregator(
		func(data []byte) {
			mu.Lock()
			flushed = append(flushed, data...)
			mu.Unlock()
		},
		func() float64 { return 0 },
		WithSmartMaxSize(maxSize),
		WithSmartBaseDelay(10*time.Millisecond),
	)

	// First write some normal data
	agg.Write([]byte("prefix"))

	// Then write a chunk larger than maxSize
	largeChunk := bytes.Repeat([]byte("L"), 200)
	agg.Write(largeChunk)

	// Buffer should still be capped at maxSize
	if agg.BufferLen() > maxSize {
		t.Errorf("Buffer exceeded maxSize after large write: %d > %d",
			agg.BufferLen(), maxSize)
	}

	agg.Stop()
	time.Sleep(20 * time.Millisecond)

	t.Logf("✅ Large chunk test: buffer stayed within %d limit", maxSize)
}

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
