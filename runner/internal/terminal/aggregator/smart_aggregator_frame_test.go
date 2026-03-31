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
	agg.SetRelayClient(relay)
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
	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
		func() float64 { return 0.3 },
		WithSmartBaseDelay(10*time.Millisecond),
	)
	agg.SetRelayClient(relay)
	defer agg.Stop()

	syncStart := "\x1b[?2026h"
	syncEnd := "\x1b[?2026l"
	clearScreen := "\x1b[2J" // ESC[2J marks a full redraw frame

	// Write Frame 1 (old incremental frame - will be discarded when full redraw arrives)
	agg.Write([]byte(syncStart + "old frame content" + syncEnd))
	// Write Frame 2 (full redraw frame with ESC[2J - triggers discard of old frame)
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
	// Old frame should be discarded because a full redraw frame was written
	if bytes.Contains(received, []byte("old frame content")) {
		t.Errorf("Old frame should be discarded when full redraw frame arrives")
	}
}

// TestSmartAggregator_SyncOutputPriorityOverClearScreen tests that full redraw frame
// discards content before it (content-aware discard)
func TestSmartAggregator_SyncOutputPriorityOverClearScreen(t *testing.T) {
	relay := newMockRelayWriter(true)

	agg := NewSmartAggregator(
		func() float64 { return 0.3 },
		WithSmartBaseDelay(10*time.Millisecond),
	)
	agg.SetRelayClient(relay)
	defer agg.Stop()

	syncStart := "\x1b[?2026h"
	syncEnd := "\x1b[?2026l"
	clearScreen := "\x1b[2J"

	// Write content before sync frame
	agg.Write([]byte(clearScreen + "after clear"))
	// Write a full redraw sync frame (contains ESC[2J inside)
	agg.Write([]byte(syncStart + clearScreen + "sync frame" + syncEnd))

	time.Sleep(200 * time.Millisecond)

	received := relay.getData()

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
