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
		{0.0, 16 * time.Millisecond},  // No load: base delay
		{0.5, 64 * time.Millisecond},  // 50% load: 16 * (1 + 0.25*12) = 16 * 4 = 64
		{0.8, 124 * time.Millisecond}, // 80% load: 16 * (1 + 0.64*12) = 16 * 8.68 ≈ 139
		{1.0, 200 * time.Millisecond}, // 100% load: capped at maxDelay
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

func TestSmartAggregator_PreservesIncrementalFrames(t *testing.T) {
	var received []byte
	var mu sync.Mutex
	done := make(chan struct{}, 10)

	agg := NewSmartAggregator(
		func(data []byte) {
			mu.Lock()
			received = append(received, data...)
			mu.Unlock()
			done <- struct{}{}
		},
		func() float64 { return 0.3 }, // Moderate pressure
		WithSmartBaseDelay(10*time.Millisecond),
	)
	defer agg.Stop()

	// Write incremental sync frames - all should be preserved
	// Small sync frames without clear screen are incremental updates
	syncStart := "\x1b[?2026h"
	syncEnd := "\x1b[?2026l"
	agg.Write([]byte(syncStart + "frame 1" + syncEnd))
	agg.Write([]byte(syncStart + "frame 2" + syncEnd))
	agg.Write([]byte(syncStart + "frame 3" + syncEnd))

	// Wait for flush
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout waiting for flush")
	}

	mu.Lock()
	defer mu.Unlock()

	// All incremental frames should be preserved (content-aware discard)
	if !bytes.Contains(received, []byte("frame 1")) {
		t.Error("Frame 1 should be preserved for incremental updates")
	}
	if !bytes.Contains(received, []byte("frame 2")) {
		t.Error("Frame 2 should be preserved for incremental updates")
	}
	if !bytes.Contains(received, []byte("frame 3")) {
		t.Error("Frame 3 should be preserved for incremental updates")
	}
}

// TestSmartAggregator_SynchronizedOutputFrameBoundary tests frame boundary detection
// with content-aware discard: old frames are only discarded when a full redraw frame arrives
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
		func() float64 { return 0.3 },
		WithSmartBaseDelay(10*time.Millisecond),
	)
	defer agg.Stop()

	syncStart := "\x1b[?2026h"
	syncEnd := "\x1b[?2026l"
	clearScreen := "\x1b[2J" // ESC[2J marks a full redraw frame

	// Write Frame 1 (old incremental frame - will be discarded when full redraw arrives)
	agg.Write([]byte(syncStart + "old frame content" + syncEnd))
	// Write Frame 2 (full redraw frame with ESC[2J - triggers discard of old frame)
	agg.Write([]byte(syncStart + clearScreen + "new frame content" + syncEnd))

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout waiting for flush")
	}

	mu.Lock()
	defer mu.Unlock()

	if !bytes.Contains(received, []byte(syncStart)) {
		t.Errorf("Expected sync output start sequence in result")
	}
	if !bytes.Contains(received, []byte(syncEnd)) {
		t.Errorf("Expected sync output end sequence in result")
	}
	if !bytes.Contains(received, []byte("new frame content")) {
		t.Errorf("Expected 'new frame content' in result")
	}
	// Old frame should be discarded because a full redraw frame was written
	if bytes.Contains(received, []byte("old frame content")) {
		t.Errorf("Old frame should be discarded when full redraw frame arrives")
	}
}

// TestSmartAggregator_SyncOutputPriorityOverClearScreen tests that full redraw frame
// discards content before it (content-aware discard)
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

	syncStart := "\x1b[?2026h"
	syncEnd := "\x1b[?2026l"
	clearScreen := "\x1b[2J"

	// Write content before sync frame
	agg.Write([]byte(clearScreen + "after clear"))
	// Write a full redraw sync frame (contains ESC[2J inside)
	agg.Write([]byte(syncStart + clearScreen + "sync frame" + syncEnd))

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout waiting for flush")
	}

	mu.Lock()
	defer mu.Unlock()

	if !bytes.Contains(received, []byte(syncStart)) {
		t.Errorf("Expected sync output start sequence")
	}
	if !bytes.Contains(received, []byte("sync frame")) {
		t.Errorf("Expected 'sync frame' in result")
	}
	// Content before the full redraw sync frame should be discarded
	if bytes.Contains(received, []byte("after clear")) {
		t.Errorf("Content before full redraw sync frame should be discarded")
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
		WithSmartMaxSize(100),
		WithSmartBaseDelay(1*time.Second),
	)
	defer agg.Stop()

	data := bytes.Repeat([]byte("x"), 150)
	agg.Write(data)

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
		WithSmartBaseDelay(1*time.Second),
	)

	agg.Write([]byte("pending data"))
	agg.Stop()

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

	agg := NewSmartAggregator(
		func(data []byte) {
			close(done)
		},
		nil,
		WithSmartBaseDelay(10*time.Millisecond),
	)

	agg.Write([]byte("test"))

	select {
	case <-done:
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
		WithSmartBaseDelay(1*time.Second),
	)
	defer agg.Stop()

	agg.Write([]byte("data"))
	agg.Flush()

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
	maxSize := 1000
	var totalFlushed int64
	var mu sync.Mutex

	agg := NewSmartAggregator(
		func(data []byte) {
			mu.Lock()
			totalFlushed += int64(len(data))
			mu.Unlock()
		},
		func() float64 { return 0.9 },
		WithSmartMaxSize(maxSize),
		WithSmartBaseDelay(50*time.Millisecond),
	)

	totalWritten := 0
	for i := 0; i < 100; i++ {
		chunk := bytes.Repeat([]byte("x"), 200)
		agg.Write(chunk)
		totalWritten += len(chunk)

		if agg.BufferLen() > maxSize {
			t.Errorf("Buffer exceeded maxSize: %d > %d", agg.BufferLen(), maxSize)
		}
	}

	agg.Stop()
	t.Logf("✅ Buffer limit test: wrote %d bytes, buffer never exceeded %d",
		totalWritten, maxSize)
}

func TestSmartAggregator_BufferLimitWithClearScreen(t *testing.T) {
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
		func() float64 { return 0.5 },
		WithSmartMaxSize(maxSize),
		WithSmartBaseDelay(10*time.Millisecond),
	)

	agg.Write(bytes.Repeat([]byte("old"), 100))
	agg.Write([]byte("\x1b[2J"))
	agg.Write([]byte("new frame content"))

	time.Sleep(50 * time.Millisecond)
	agg.Stop()

	mu.Lock()
	defer mu.Unlock()

	if !bytes.Contains(lastFlush, []byte("\x1b[2J")) {
		t.Error("Clear screen should be preserved")
	}
	if !bytes.Contains(lastFlush, []byte("new frame content")) {
		t.Error("New frame content should be preserved")
	}
	if bytes.Contains(lastFlush, []byte("oldoldold")) {
		t.Error("Old frame content should be discarded")
	}
}

func TestSmartAggregator_LargeChunkExceedsMaxSize(t *testing.T) {
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

	agg.Write([]byte("prefix"))
	largeChunk := bytes.Repeat([]byte("L"), 200)
	agg.Write(largeChunk)

	if agg.BufferLen() > maxSize {
		t.Errorf("Buffer exceeded maxSize after large write: %d > %d",
			agg.BufferLen(), maxSize)
	}

	agg.Stop()
	time.Sleep(20 * time.Millisecond)

	t.Logf("✅ Large chunk test: buffer stayed within %d limit", maxSize)
}
