package aggregator

import (
	"bytes"
	"testing"
	"time"
)

// Tests for max size handling and buffer limits

func TestSmartAggregator_MaxSizeFlush(t *testing.T) {
	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
		func() float64 { return 0 },
		WithSmartMaxSize(100),
		WithSmartBaseDelay(1*time.Second),
	)
	agg.SetRelayClient(relay)
	defer agg.Stop()

	data := bytes.Repeat([]byte("x"), 150)
	agg.Write(data)

	// Wait for immediate flush on max size exceeded
	time.Sleep(50 * time.Millisecond)

	if len(relay.getData()) < 1 {
		t.Fatal("Expected immediate flush on max size exceeded")
	}
}

func TestSmartAggregator_BufferLimitEnforced(t *testing.T) {
	maxSize := 1000
	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
		func() float64 { return 0.9 },
		WithSmartMaxSize(maxSize),
		WithSmartBaseDelay(50*time.Millisecond),
	)
	agg.SetRelayClient(relay)

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
	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
		func() float64 { return 0.5 },
		WithSmartMaxSize(maxSize),
		WithSmartBaseDelay(10*time.Millisecond),
	)
	agg.SetRelayClient(relay)

	agg.Write(bytes.Repeat([]byte("old"), 100))
	agg.Write([]byte("\x1b[2J"))
	agg.Write([]byte("new frame content"))

	time.Sleep(100 * time.Millisecond)
	agg.Stop()

	lastFlush := relay.getData()

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

