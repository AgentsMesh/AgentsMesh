package terminal

import (
	"bytes"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// --- Test NewOutputAggregator ---

func TestNewOutputAggregator(t *testing.T) {
	flushed := false
	agg := NewOutputAggregator(func(data []byte) {
		flushed = true
	})

	if agg == nil {
		t.Fatal("NewOutputAggregator returned nil")
	}

	// Check default values
	if agg.maxDelay != 16*time.Millisecond {
		t.Errorf("maxDelay: got %v, want 16ms", agg.maxDelay)
	}

	if agg.maxSize != 16*1024 {
		t.Errorf("maxSize: got %v, want 16384", agg.maxSize)
	}

	if agg.stopped {
		t.Error("stopped should be false initially")
	}

	if flushed {
		t.Error("flush should not be called on creation")
	}
}

func TestNewOutputAggregatorWithOptions(t *testing.T) {
	agg := NewOutputAggregator(
		func(data []byte) {},
		WithMaxDelay(50*time.Millisecond),
		WithMaxSize(8*1024),
	)

	if agg.maxDelay != 50*time.Millisecond {
		t.Errorf("maxDelay: got %v, want 50ms", agg.maxDelay)
	}

	if agg.maxSize != 8*1024 {
		t.Errorf("maxSize: got %v, want 8192", agg.maxSize)
	}
}

func TestNewOutputAggregatorNilOnFlush(t *testing.T) {
	// Should not panic with nil onFlush
	agg := NewOutputAggregator(nil)

	// Write should not panic
	agg.Write([]byte("test"))

	// Flush should not panic
	agg.Flush()

	// Stop should not panic
	agg.Stop()
}

// --- Test Write ---

func TestWriteSingleChunk(t *testing.T) {
	var received []byte
	var mu sync.Mutex
	flushed := make(chan struct{}, 1)

	agg := NewOutputAggregator(func(data []byte) {
		mu.Lock()
		received = append(received, data...)
		mu.Unlock()
		select {
		case flushed <- struct{}{}:
		default:
		}
	}, WithMaxDelay(10*time.Millisecond))

	agg.Write([]byte("hello"))

	// Wait for timer flush
	select {
	case <-flushed:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for flush")
	}

	mu.Lock()
	defer mu.Unlock()
	if !bytes.Equal(received, []byte("hello")) {
		t.Errorf("received: got %q, want %q", received, "hello")
	}
}

func TestWriteMultipleChunks(t *testing.T) {
	var received []byte
	var mu sync.Mutex
	flushed := make(chan struct{}, 1)

	agg := NewOutputAggregator(func(data []byte) {
		mu.Lock()
		received = append(received, data...)
		mu.Unlock()
		select {
		case flushed <- struct{}{}:
		default:
		}
	}, WithMaxDelay(50*time.Millisecond))

	// Write multiple chunks quickly
	agg.Write([]byte("hello"))
	agg.Write([]byte(" "))
	agg.Write([]byte("world"))

	// Wait for timer flush
	select {
	case <-flushed:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for flush")
	}

	mu.Lock()
	defer mu.Unlock()
	if !bytes.Equal(received, []byte("hello world")) {
		t.Errorf("received: got %q, want %q", received, "hello world")
	}
}

func TestWriteExceedsMaxSize(t *testing.T) {
	var flushCount int32
	var received []byte
	var mu sync.Mutex

	agg := NewOutputAggregator(func(data []byte) {
		atomic.AddInt32(&flushCount, 1)
		mu.Lock()
		received = append(received, data...)
		mu.Unlock()
	}, WithMaxSize(10)) // Very small buffer

	// Write data larger than maxSize
	data := []byte("this is a longer string")
	agg.Write(data)

	// Should flush immediately due to exceeding maxSize
	time.Sleep(10 * time.Millisecond)

	if atomic.LoadInt32(&flushCount) < 1 {
		t.Error("expected at least one immediate flush when exceeding maxSize")
	}

	mu.Lock()
	defer mu.Unlock()
	if !bytes.Equal(received, data) {
		t.Errorf("received: got %q, want %q", received, data)
	}
}

func TestWriteAfterStopped(t *testing.T) {
	var flushCount int32

	agg := NewOutputAggregator(func(data []byte) {
		atomic.AddInt32(&flushCount, 1)
	})

	agg.Stop()

	// Write after stop should be ignored
	agg.Write([]byte("ignored"))

	time.Sleep(50 * time.Millisecond)

	// Only the Stop() flush should have been called (with empty buffer)
	// No additional flushes from Write
	if atomic.LoadInt32(&flushCount) > 1 {
		t.Errorf("flushCount: got %d, expected at most 1", atomic.LoadInt32(&flushCount))
	}
}

// --- Test Flush ---

func TestFlushEmptyBuffer(t *testing.T) {
	var flushCount int32

	agg := NewOutputAggregator(func(data []byte) {
		atomic.AddInt32(&flushCount, 1)
	})

	// Flush empty buffer should not call onFlush
	agg.Flush()

	if atomic.LoadInt32(&flushCount) != 0 {
		t.Errorf("flushCount: got %d, want 0", atomic.LoadInt32(&flushCount))
	}
}

func TestFlushWithData(t *testing.T) {
	var received []byte
	flushed := make(chan struct{}, 1)

	agg := NewOutputAggregator(func(data []byte) {
		received = data
		flushed <- struct{}{}
	}, WithMaxDelay(1*time.Hour)) // Long delay to ensure timer doesn't fire

	agg.Write([]byte("test data"))
	agg.Flush()

	select {
	case <-flushed:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for flush")
	}

	if !bytes.Equal(received, []byte("test data")) {
		t.Errorf("received: got %q, want %q", received, "test data")
	}
}

func TestFlushStopsTimer(t *testing.T) {
	var flushCount int32

	agg := NewOutputAggregator(func(data []byte) {
		atomic.AddInt32(&flushCount, 1)
	}, WithMaxDelay(50*time.Millisecond))

	agg.Write([]byte("test"))
	agg.Flush() // Should stop the timer

	time.Sleep(100 * time.Millisecond)

	// Should only have one flush from the explicit Flush() call
	if atomic.LoadInt32(&flushCount) != 1 {
		t.Errorf("flushCount: got %d, want 1", atomic.LoadInt32(&flushCount))
	}
}

func TestFlushMultipleTimes(t *testing.T) {
	var flushCount int32

	agg := NewOutputAggregator(func(data []byte) {
		atomic.AddInt32(&flushCount, 1)
	})

	agg.Write([]byte("data1"))
	agg.Flush()

	agg.Write([]byte("data2"))
	agg.Flush()

	agg.Write([]byte("data3"))
	agg.Flush()

	// Three flushes with data
	if atomic.LoadInt32(&flushCount) != 3 {
		t.Errorf("flushCount: got %d, want 3", atomic.LoadInt32(&flushCount))
	}
}

// --- Test Stop ---

func TestStopFlushesRemainingData(t *testing.T) {
	var received []byte
	flushed := make(chan struct{}, 1)

	agg := NewOutputAggregator(func(data []byte) {
		received = data
		flushed <- struct{}{}
	}, WithMaxDelay(1*time.Hour)) // Long delay

	agg.Write([]byte("remaining"))
	agg.Stop()

	select {
	case <-flushed:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for flush on stop")
	}

	if !bytes.Equal(received, []byte("remaining")) {
		t.Errorf("received: got %q, want %q", received, "remaining")
	}
}

func TestStopMultipleTimes(t *testing.T) {
	var flushCount int32

	agg := NewOutputAggregator(func(data []byte) {
		atomic.AddInt32(&flushCount, 1)
	})

	agg.Write([]byte("data"))

	// Stop multiple times should not panic and only flush once
	agg.Stop()
	agg.Stop()
	agg.Stop()

	if atomic.LoadInt32(&flushCount) != 1 {
		t.Errorf("flushCount: got %d, want 1", atomic.LoadInt32(&flushCount))
	}
}

func TestStopEmptyBuffer(t *testing.T) {
	var flushCount int32

	agg := NewOutputAggregator(func(data []byte) {
		atomic.AddInt32(&flushCount, 1)
	})

	// Stop with empty buffer should not call onFlush
	agg.Stop()

	if atomic.LoadInt32(&flushCount) != 0 {
		t.Errorf("flushCount: got %d, want 0", atomic.LoadInt32(&flushCount))
	}
}

func TestStopStopsTimer(t *testing.T) {
	var flushCount int32

	agg := NewOutputAggregator(func(data []byte) {
		atomic.AddInt32(&flushCount, 1)
	}, WithMaxDelay(50*time.Millisecond))

	agg.Write([]byte("test"))
	agg.Stop() // Should stop the timer

	time.Sleep(100 * time.Millisecond)

	// Should only have one flush from Stop()
	if atomic.LoadInt32(&flushCount) != 1 {
		t.Errorf("flushCount: got %d, want 1", atomic.LoadInt32(&flushCount))
	}
}

// --- Test Timer Behavior ---

func TestTimerResetOnWrite(t *testing.T) {
	var flushCount int32
	var lastFlushTime time.Time
	var mu sync.Mutex

	agg := NewOutputAggregator(func(data []byte) {
		mu.Lock()
		lastFlushTime = time.Now()
		mu.Unlock()
		atomic.AddInt32(&flushCount, 1)
	}, WithMaxDelay(30*time.Millisecond))

	startTime := time.Now()
	agg.Write([]byte("first"))

	// Write again before timer fires - timer should NOT reset (single timer design)
	time.Sleep(10 * time.Millisecond)
	agg.Write([]byte("second"))

	// Wait for flush
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	elapsed := lastFlushTime.Sub(startTime)
	mu.Unlock()

	// Flush should happen around 30ms after first write
	if elapsed < 25*time.Millisecond || elapsed > 50*time.Millisecond {
		t.Errorf("flush timing: got %v, expected ~30ms", elapsed)
	}

	// Should only have one flush (both writes aggregated)
	if atomic.LoadInt32(&flushCount) != 1 {
		t.Errorf("flushCount: got %d, want 1", atomic.LoadInt32(&flushCount))
	}
}

// --- Test Concurrency ---

func TestConcurrentWrites(t *testing.T) {
	var totalReceived int32
	var mu sync.Mutex

	agg := NewOutputAggregator(func(data []byte) {
		mu.Lock()
		totalReceived += int32(len(data))
		mu.Unlock()
	}, WithMaxDelay(10*time.Millisecond))

	var wg sync.WaitGroup
	numGoroutines := 10
	writesPerGoroutine := 100
	dataPerWrite := []byte("x")

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < writesPerGoroutine; j++ {
				agg.Write(dataPerWrite)
			}
		}()
	}

	wg.Wait()
	agg.Stop()

	// Give time for final flush
	time.Sleep(50 * time.Millisecond)

	expected := int32(numGoroutines * writesPerGoroutine)
	mu.Lock()
	received := totalReceived
	mu.Unlock()

	if received != expected {
		t.Errorf("totalReceived: got %d, want %d", received, expected)
	}
}

func TestConcurrentWriteAndFlush(t *testing.T) {
	var flushCount int32

	agg := NewOutputAggregator(func(data []byte) {
		atomic.AddInt32(&flushCount, 1)
	}, WithMaxDelay(5*time.Millisecond))

	var wg sync.WaitGroup

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			agg.Write([]byte("data"))
			time.Sleep(time.Millisecond)
		}
	}()

	// Flusher goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			agg.Flush()
			time.Sleep(2 * time.Millisecond)
		}
	}()

	wg.Wait()
	agg.Stop()

	// Should have multiple flushes without panic
	if atomic.LoadInt32(&flushCount) == 0 {
		t.Error("expected at least some flushes")
	}
}

func TestConcurrentWriteAndStop(t *testing.T) {
	agg := NewOutputAggregator(func(data []byte) {
		// Do nothing
	}, WithMaxDelay(10*time.Millisecond))

	var wg sync.WaitGroup

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			agg.Write([]byte("data"))
		}
	}()

	// Stop after some writes
	time.Sleep(5 * time.Millisecond)
	agg.Stop()

	wg.Wait()

	// Should not panic
}

// --- Test Edge Cases ---

func TestWriteEmptyData(t *testing.T) {
	var flushCount int32

	agg := NewOutputAggregator(func(data []byte) {
		atomic.AddInt32(&flushCount, 1)
	}, WithMaxDelay(10*time.Millisecond))

	agg.Write([]byte{})
	agg.Write(nil)

	time.Sleep(50 * time.Millisecond)

	// Empty writes should still trigger flush if timer was started
	// But the flushed data would be empty, so onFlush is not called
	// Actually, bytes.Buffer.Write with empty slice is valid and adds 0 bytes
	// Timer is started on any Write, so it will fire
	// But flushLocked checks buffer.Len() > 0 before calling onFlush
}

func TestLargeDataWrite(t *testing.T) {
	var received []byte
	var mu sync.Mutex

	agg := NewOutputAggregator(func(data []byte) {
		mu.Lock()
		received = append(received, data...)
		mu.Unlock()
	}, WithMaxSize(1024))

	// Write data larger than maxSize
	largeData := make([]byte, 2048)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	agg.Write(largeData)
	agg.Stop()

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if !bytes.Equal(received, largeData) {
		t.Errorf("received length: got %d, want %d", len(received), len(largeData))
	}
}

func TestExactMaxSizeWrite(t *testing.T) {
	var flushCount int32
	maxSize := 100

	agg := NewOutputAggregator(func(data []byte) {
		atomic.AddInt32(&flushCount, 1)
	}, WithMaxSize(maxSize), WithMaxDelay(1*time.Hour))

	// Write exactly maxSize bytes
	data := make([]byte, maxSize)
	agg.Write(data)

	// Should flush immediately when >= maxSize
	time.Sleep(10 * time.Millisecond)

	if atomic.LoadInt32(&flushCount) != 1 {
		t.Errorf("flushCount: got %d, want 1", atomic.LoadInt32(&flushCount))
	}
}

// --- Test Options ---

func TestWithMaxDelayOption(t *testing.T) {
	delays := []time.Duration{
		1 * time.Millisecond,
		100 * time.Millisecond,
		1 * time.Second,
	}

	for _, delay := range delays {
		agg := NewOutputAggregator(nil, WithMaxDelay(delay))
		if agg.maxDelay != delay {
			t.Errorf("WithMaxDelay(%v): got %v", delay, agg.maxDelay)
		}
	}
}

func TestWithMaxSizeOption(t *testing.T) {
	sizes := []int{
		1,
		1024,
		1024 * 1024,
	}

	for _, size := range sizes {
		agg := NewOutputAggregator(nil, WithMaxSize(size))
		if agg.maxSize != size {
			t.Errorf("WithMaxSize(%d): got %d", size, agg.maxSize)
		}
	}
}

func TestMultipleOptions(t *testing.T) {
	agg := NewOutputAggregator(
		nil,
		WithMaxDelay(100*time.Millisecond),
		WithMaxSize(2048),
	)

	if agg.maxDelay != 100*time.Millisecond {
		t.Errorf("maxDelay: got %v, want 100ms", agg.maxDelay)
	}

	if agg.maxSize != 2048 {
		t.Errorf("maxSize: got %d, want 2048", agg.maxSize)
	}
}

// --- Test Data Integrity ---

func TestDataIntegrity(t *testing.T) {
	var allReceived []byte
	var mu sync.Mutex

	agg := NewOutputAggregator(func(data []byte) {
		mu.Lock()
		allReceived = append(allReceived, data...)
		mu.Unlock()
	}, WithMaxDelay(5*time.Millisecond), WithMaxSize(50))

	// Write various data patterns
	patterns := [][]byte{
		[]byte("hello"),
		[]byte(" "),
		[]byte("world"),
		[]byte("!"),
		make([]byte, 100), // Large chunk that exceeds maxSize
		[]byte("end"),
	}

	var expected []byte
	for _, p := range patterns {
		agg.Write(p)
		expected = append(expected, p...)
	}

	agg.Stop()
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if !bytes.Equal(allReceived, expected) {
		t.Errorf("data integrity check failed: received %d bytes, expected %d bytes",
			len(allReceived), len(expected))
	}
}

// --- Benchmark Tests ---

func BenchmarkWrite(b *testing.B) {
	agg := NewOutputAggregator(func(data []byte) {
		// Do nothing
	})
	defer agg.Stop()

	data := []byte("benchmark data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agg.Write(data)
	}
}

func BenchmarkWriteParallel(b *testing.B) {
	agg := NewOutputAggregator(func(data []byte) {
		// Do nothing
	})
	defer agg.Stop()

	data := []byte("benchmark data")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			agg.Write(data)
		}
	})
}

func BenchmarkFlush(b *testing.B) {
	agg := NewOutputAggregator(func(data []byte) {
		// Do nothing
	}, WithMaxDelay(1*time.Hour))
	defer agg.Stop()

	data := []byte("benchmark data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agg.Write(data)
		agg.Flush()
	}
}
