package aggregator

import (
	"bytes"
	"testing"
	"time"
)

// Tests for frame handling and synchronized output

func TestSmartAggregator_PreservesIncrementalFrames(t *testing.T) {
	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
		func() float64 { return 0.3 }, // Moderate pressure
		WithSmartBaseDelay(10*time.Millisecond),
	)
	agg.SetOutputDestination(OutputDestinationCloud, relay)
	defer agg.Stop()

	// Write incremental sync frames - all should be preserved
	// Small sync frames without clear screen are incremental updates
	syncStart := "\x1b[?2026h"
	syncEnd := "\x1b[?2026l"
	agg.Write([]byte(syncStart + "frame 1" + syncEnd))
	agg.Write([]byte(syncStart + "frame 2" + syncEnd))
	agg.Write([]byte(syncStart + "frame 3" + syncEnd))

	// Wait for flush
	time.Sleep(200 * time.Millisecond)

	received := relay.getData()

	// Every incremental frame must remain in raw stream order.
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

// TestSmartAggregator_SynchronizedOutputFrameBoundary verifies that ED2 never
// discards earlier parser state from the raw stream.
func TestSmartAggregator_SynchronizedOutputFrameBoundary(t *testing.T) {
	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
		func() float64 { return 0.3 },
		WithSmartBaseDelay(10*time.Millisecond),
	)
	agg.SetOutputDestination(OutputDestinationCloud, relay)
	defer agg.Stop()

	syncStart := "\x1b[?2026h"
	syncEnd := "\x1b[?2026l"
	clearScreen := "\x1b[2J"

	agg.Write([]byte(syncStart + "old frame content" + syncEnd))
	agg.Write([]byte(syncStart + clearScreen + "new frame content" + syncEnd))

	time.Sleep(200 * time.Millisecond)

	received := relay.getData()

	if !bytes.Contains(received, []byte(syncStart)) {
		t.Errorf("Expected sync output start sequence in result")
	}
	if !bytes.Contains(received, []byte(syncEnd)) {
		t.Errorf("Expected sync output end sequence in result")
	}
	if !bytes.Contains(received, []byte("new frame content")) {
		t.Errorf("Expected 'new frame content' in result")
	}
	if !bytes.Contains(received, []byte("old frame content")) {
		t.Errorf("ED2 discarded earlier terminal parser state")
	}
}

// TestSmartAggregator_SyncOutputPreservesPrefixBeforeClearScreen verifies that
// synchronized output is a paint boundary, not a terminal-state baseline.
func TestSmartAggregator_SyncOutputPriorityOverClearScreen(t *testing.T) {
	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
		func() float64 { return 0.3 },
		WithSmartBaseDelay(10*time.Millisecond),
	)
	agg.SetOutputDestination(OutputDestinationCloud, relay)
	defer agg.Stop()

	syncStart := "\x1b[?2026h"
	syncEnd := "\x1b[?2026l"
	clearScreen := "\x1b[2J"

	agg.Write([]byte(clearScreen + "after clear"))
	agg.Write([]byte(syncStart + clearScreen + "sync frame" + syncEnd))

	time.Sleep(200 * time.Millisecond)

	received := relay.getData()

	if !bytes.Contains(received, []byte(syncStart)) {
		t.Errorf("Expected sync output start sequence")
	}
	if !bytes.Contains(received, []byte("sync frame")) {
		t.Errorf("Expected 'sync frame' in result")
	}
	if !bytes.Contains(received, []byte("after clear")) {
		t.Errorf("content before synchronized ED2 frame was discarded")
	}
}
