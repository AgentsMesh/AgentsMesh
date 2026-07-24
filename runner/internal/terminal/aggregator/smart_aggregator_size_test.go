package aggregator

import (
	"bytes"
	"testing"
	"time"
)

// Tests for max size handling and buffer limits

func TestSmartAggregator_BufferOverflowDesynchronizesOutput(t *testing.T) {
	relay := newMockRelayWriter(true)
	health := make(chan OutputHealth, 1)

	agg := NewSmartAggregator(
		func() float64 { return 0 },
		WithSmartMaxSize(100),
		WithSmartBaseDelay(1*time.Second),
	)
	agg.SetOutputDestination(OutputDestinationCloud, relay)
	agg.SetOutputHealthHandler(func(state OutputHealth) { health <- state })
	defer agg.Stop()

	data := bytes.Repeat([]byte("x"), 150)
	agg.Write(data)

	select {
	case state := <-health:
		if state.Reason != OutputDesyncBuffer {
			t.Fatalf("desync reason = %q, want %q", state.Reason, OutputDesyncBuffer)
		}
	case <-time.After(time.Second):
		t.Fatal("buffer overflow did not desynchronize output")
	}
	if got := relay.getData(); len(got) != 0 {
		t.Fatalf("overflow emitted a partial ANSI stream: %q", got)
	}
}

func TestSmartAggregatorSnapshotCutRetainsSingleUTF8Lead(t *testing.T) {
	relay := newMockRelayWriter(true)
	agg := NewSmartAggregator(
		func() float64 { return 0 },
		WithSmartMaxSize(1),
		WithSmartBaseDelay(time.Second),
	)
	agg.SetOutputDestination(OutputDestinationCloud, relay)
	defer agg.Stop()

	agg.Write([]byte{0xe4})
	boundary, ok := agg.PrepareSnapshot(OutputDestinationCloud, time.Second)
	if !ok {
		t.Fatal("snapshot boundary failed for an incomplete UTF-8 lead byte")
	}
	if got := relay.getData(); len(got) != 0 {
		t.Fatalf("snapshot cut emitted incomplete UTF-8: %x", got)
	}
	if got := agg.BufferLen(); got != 1 {
		t.Fatalf("snapshot cut retained %d bytes, want one UTF-8 lead byte", got)
	}
	if !boundary.Commit() {
		t.Fatal("snapshot boundary commit failed")
	}

	agg.Write([]byte{0xb8, 0xad})
	if !agg.FlushAndWait(time.Second) {
		t.Fatal("completed UTF-8 rune did not drain")
	}
	if got := relay.getData(); !bytes.Equal(got, []byte("中")) {
		t.Fatalf("post-snapshot UTF-8 stream = %x, want %x", got, []byte("中"))
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
	agg.SetOutputDestination(OutputDestinationCloud, relay)

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

func TestSmartAggregator_ClearScreenDoesNotDiscardPrefix(t *testing.T) {
	maxSize := 500
	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
		func() float64 { return 0.5 },
		WithSmartMaxSize(maxSize),
		WithSmartBaseDelay(10*time.Millisecond),
	)
	agg.SetOutputDestination(OutputDestinationCloud, relay)

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
	if !bytes.Contains(lastFlush, []byte("oldoldold")) {
		t.Error("ED2 discarded terminal state from before the clear")
	}
}
