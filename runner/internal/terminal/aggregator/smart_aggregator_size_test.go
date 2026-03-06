package aggregator

import (
	"bytes"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Tests for max size handling and buffer limits

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
	t.Logf("Buffer limit test: wrote %d bytes, buffer never exceeded %d",
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

	time.Sleep(100 * time.Millisecond)
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

