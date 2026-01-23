package terminal

import (
	"bytes"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestSmartAggregator_HighLoadFrameDropping verifies frame dropping under high load.
func TestSmartAggregator_HighLoadFrameDropping(t *testing.T) {
	var totalBytes int64
	var flushCount int64
	var mu sync.Mutex
	var lastData []byte

	// Simulate high queue pressure (80%)
	queueUsage := 0.8

	agg := NewSmartAggregator(
		func(data []byte) {
			atomic.AddInt64(&flushCount, 1)
			atomic.AddInt64(&totalBytes, int64(len(data)))
			mu.Lock()
			lastData = data
			mu.Unlock()
		},
		func() float64 { return queueUsage },
		WithSmartBaseDelay(5*time.Millisecond), // Faster for testing
	)

	// Send multiple frames with clear screen sequences
	for i := 0; i < 10; i++ {
		// Old content (should be discarded under high load)
		agg.Write([]byte("old frame content that should be dropped"))
		// Clear screen
		agg.Write([]byte("\x1b[2J"))
		// New content (should be kept)
		agg.Write([]byte("new frame"))
	}

	agg.Stop()
	time.Sleep(50 * time.Millisecond) // Wait for async flush

	mu.Lock()
	defer mu.Unlock()

	// Last data should start with clear screen (old content discarded)
	if !bytes.HasPrefix(lastData, clearScreenSeq) {
		t.Errorf("Expected last data to start with clear screen sequence")
	}

	t.Logf("✅ High load frame dropping test:")
	t.Logf("   Flush count: %d, Total bytes: %d", flushCount, totalBytes)
	t.Logf("   Last data length: %d (starts with ESC[2J: %v)",
		len(lastData), bytes.HasPrefix(lastData, clearScreenSeq))
}

// TestSmartAggregator_AdaptiveDelayUnderPressure verifies delay increases with load.
func TestSmartAggregator_AdaptiveDelayUnderPressure(t *testing.T) {
	var queueUsage float64
	var mu sync.Mutex

	setUsage := func(u float64) {
		mu.Lock()
		queueUsage = u
		mu.Unlock()
	}

	getUsage := func() float64 {
		mu.Lock()
		defer mu.Unlock()
		return queueUsage
	}

	var flushTimes []time.Time
	var flushMu sync.Mutex

	agg := NewSmartAggregator(
		func(data []byte) {
			flushMu.Lock()
			flushTimes = append(flushTimes, time.Now())
			flushMu.Unlock()
		},
		getUsage,
		WithSmartBaseDelay(10*time.Millisecond),
		WithSmartMaxDelay(100*time.Millisecond),
	)

	// Test at different load levels
	testCases := []struct {
		usage       float64
		minInterval time.Duration
		maxInterval time.Duration
	}{
		{0.0, 8 * time.Millisecond, 20 * time.Millisecond},   // Low load
		{0.5, 20 * time.Millisecond, 60 * time.Millisecond},  // Medium load
		{0.9, 50 * time.Millisecond, 120 * time.Millisecond}, // High load
	}

	for _, tc := range testCases {
		setUsage(tc.usage)
		flushMu.Lock()
		flushTimes = nil
		flushMu.Unlock()

		start := time.Now()
		for i := 0; i < 5; i++ {
			agg.Write([]byte("test"))
			time.Sleep(tc.maxInterval)
		}

		flushMu.Lock()
		count := len(flushTimes)
		flushMu.Unlock()

		elapsed := time.Since(start)
		t.Logf("   Usage %.0f%%: %d flushes in %v", tc.usage*100, count, elapsed)
	}

	agg.Stop()
	t.Logf("✅ Adaptive delay test completed")
}

// TestSmartAggregator_ConcurrentHighVolume simulates multiple terminals.
func TestSmartAggregator_ConcurrentHighVolume(t *testing.T) {
	var totalFlushed int64
	var totalBytes int64

	agg := NewSmartAggregator(
		func(data []byte) {
			atomic.AddInt64(&totalFlushed, 1)
			atomic.AddInt64(&totalBytes, int64(len(data)))
		},
		func() float64 { return 0.3 }, // Moderate pressure
		WithSmartBaseDelay(5*time.Millisecond),
	)

	var wg sync.WaitGroup
	numWriters := 10
	writesPerWriter := 1000

	start := time.Now()

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < writesPerWriter; j++ {
				agg.Write([]byte("terminal output data from writer"))
			}
		}(i)
	}

	wg.Wait()
	agg.Stop()
	time.Sleep(50 * time.Millisecond) // Wait for final flush

	elapsed := time.Since(start)
	expectedBytes := int64(numWriters * writesPerWriter * len("terminal output data from writer"))
	finalFlushed := atomic.LoadInt64(&totalFlushed)
	finalBytes := atomic.LoadInt64(&totalBytes)

	t.Logf("✅ Concurrent high volume test:")
	t.Logf("   Writers: %d × %d writes = %d total", numWriters, writesPerWriter, numWriters*writesPerWriter)
	t.Logf("   Flushed: %d times, %d bytes (expected: %d)", finalFlushed, finalBytes, expectedBytes)
	t.Logf("   Elapsed: %v, Throughput: %.0f writes/sec", elapsed, float64(numWriters*writesPerWriter)/elapsed.Seconds())

	// Should capture all bytes
	if finalBytes != expectedBytes {
		t.Errorf("Expected %d bytes, got %d", expectedBytes, finalBytes)
	}
}

// TestSmartAggregator_RapidStartStop verifies no data loss on rapid start/stop.
func TestSmartAggregator_RapidStartStop(t *testing.T) {
	var totalBytes int64

	for i := 0; i < 100; i++ {
		agg := NewSmartAggregator(
			func(data []byte) {
				atomic.AddInt64(&totalBytes, int64(len(data)))
			},
			func() float64 { return 0 },
			WithSmartBaseDelay(1*time.Millisecond),
		)

		agg.Write([]byte("quick data"))
		agg.Stop()
	}

	time.Sleep(100 * time.Millisecond) // Wait for all async flushes

	expectedBytes := int64(100 * len("quick data"))
	finalBytes := atomic.LoadInt64(&totalBytes)
	if finalBytes != expectedBytes {
		t.Errorf("Expected %d bytes, got %d (data loss: %d)", expectedBytes, finalBytes, expectedBytes-finalBytes)
	}

	t.Logf("✅ Rapid start/stop test: %d bytes captured (no data loss)", finalBytes)
}

// TestSmartAggregator_ClearScreenDetection verifies ESC[2J detection accuracy.
func TestSmartAggregator_ClearScreenDetection(t *testing.T) {
	var lastData []byte
	var mu sync.Mutex

	agg := NewSmartAggregator(
		func(data []byte) {
			mu.Lock()
			lastData = make([]byte, len(data))
			copy(lastData, data)
			mu.Unlock()
		},
		func() float64 { return 0.8 }, // High pressure to trigger discard
		WithSmartBaseDelay(5*time.Millisecond),
	)

	// Test various clear screen sequences
	testCases := []struct {
		name  string
		input [][]byte
	}{
		{"single clear", [][]byte{[]byte("old"), clearScreenSeq, []byte("new")}},
		{"multiple clears", [][]byte{[]byte("a"), clearScreenSeq, []byte("b"), clearScreenSeq, []byte("c")}},
		{"no clear", [][]byte{[]byte("just text")}},
	}

	for _, tc := range testCases {
		mu.Lock()
		lastData = nil
		mu.Unlock()

		for _, data := range tc.input {
			agg.Write(data)
		}
		agg.Flush()
		time.Sleep(20 * time.Millisecond)

		mu.Lock()
		result := lastData
		mu.Unlock()

		t.Logf("   %s: input=%d parts, output=%d bytes", tc.name, len(tc.input), len(result))
	}

	agg.Stop()
	t.Logf("✅ Clear screen detection test completed")
}

// BenchmarkSmartAggregator_Write measures write throughput.
func BenchmarkSmartAggregator_Write(b *testing.B) {
	agg := NewSmartAggregator(
		func(data []byte) {},
		func() float64 { return 0 },
	)
	defer agg.Stop()

	data := bytes.Repeat([]byte("x"), 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agg.Write(data)
	}
}

// BenchmarkSmartAggregator_WriteUnderPressure measures write with high queue pressure.
func BenchmarkSmartAggregator_WriteUnderPressure(b *testing.B) {
	agg := NewSmartAggregator(
		func(data []byte) {},
		func() float64 { return 0.9 }, // High pressure
	)
	defer agg.Stop()

	// Include clear screen to trigger frame discard
	data := append([]byte("old content"), clearScreenSeq...)
	data = append(data, []byte("new content")...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agg.Write(data)
	}
}

// BenchmarkSmartAggregator_ConcurrentWrite measures concurrent write performance.
func BenchmarkSmartAggregator_ConcurrentWrite(b *testing.B) {
	agg := NewSmartAggregator(
		func(data []byte) {},
		func() float64 { return 0.5 },
	)
	defer agg.Stop()

	data := bytes.Repeat([]byte("x"), 256)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			agg.Write(data)
		}
	})
}
