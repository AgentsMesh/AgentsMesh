package aggregator

import (
	"bytes"
	"testing"
)

// ============== IsLastFrameFullRedraw Tests ==============

// TestFrameBuffer_IsLastFrameFullRedraw_EmptyBuffer tests empty buffer case
func TestFrameBuffer_IsLastFrameFullRedraw_EmptyBuffer(t *testing.T) {
	fb := NewFrameBuffer(1000)
	if fb.IsLastFrameFullRedraw() {
		t.Error("Empty buffer should not report full redraw")
	}
}

// TestFrameBuffer_IsLastFrameFullRedraw_NoSyncFrames tests plain text without sync frames
func TestFrameBuffer_IsLastFrameFullRedraw_NoSyncFrames(t *testing.T) {
	fb := NewFrameBuffer(1000)
	fb.Write([]byte("plain text without sync frames"))
	if fb.IsLastFrameFullRedraw() {
		t.Error("Buffer without sync frames should not report full redraw")
	}
}

// TestFrameBuffer_IsLastFrameFullRedraw_IncrementalFrame tests small incremental frame
func TestFrameBuffer_IsLastFrameFullRedraw_IncrementalFrame(t *testing.T) {
	fb := NewFrameBuffer(1000)
	// Small frame without clear screen - should be incremental
	frame := buildSyncFrame("small content")
	fb.Write(frame)

	if fb.IsLastFrameFullRedraw() {
		t.Error("Small incremental frame should not report full redraw")
	}
}

// TestFrameBuffer_IsLastFrameFullRedraw_WithClearScreen tests frame with ESC[2J
func TestFrameBuffer_IsLastFrameFullRedraw_WithClearScreen(t *testing.T) {
	fb := NewFrameBuffer(1000)
	// Frame with clear screen sequence
	frame := buildFullRedrawFrame("some content")
	fb.Write(frame)

	if !fb.IsLastFrameFullRedraw() {
		t.Error("Frame with clear screen should be detected as full redraw")
	}
}

// TestFrameBuffer_IsLastFrameFullRedraw_WithCursorHome tests frame starting with ESC[H
func TestFrameBuffer_IsLastFrameFullRedraw_WithCursorHome(t *testing.T) {
	fb := NewFrameBuffer(1000)
	// Frame that starts with cursor home
	cursorHome := []byte{0x1b, '[', 'H'}
	content := append(cursorHome, []byte("content after cursor home")...)
	frame := append(append(syncOutputStartSeq, content...), syncOutputEndSeq...)
	fb.Write(frame)

	if !fb.IsLastFrameFullRedraw() {
		t.Error("Frame starting with cursor home should be detected as full redraw")
	}
}

// TestFrameBuffer_IsLastFrameFullRedraw_LargeFrame tests large frame (>1KB)
func TestFrameBuffer_IsLastFrameFullRedraw_LargeFrame(t *testing.T) {
	fb := NewFrameBuffer(5000)
	// Large frame (>1KB) without clear screen or cursor home
	largeContent := string(bytes.Repeat([]byte("x"), 2000))
	frame := buildSyncFrame(largeContent)
	fb.Write(frame)

	if !fb.IsLastFrameFullRedraw() {
		t.Error("Large frame (>1KB) should be detected as full redraw")
	}
}

// TestFrameBuffer_IsLastFrameFullRedraw_IncompleteFrame tests incomplete frame
func TestFrameBuffer_IsLastFrameFullRedraw_IncompleteFrame(t *testing.T) {
	fb := NewFrameBuffer(1000)
	// Incomplete frame (no end sequence)
	incomplete := append(syncOutputStartSeq, []byte("incomplete frame")...)
	fb.Write(incomplete)

	// Should return false because there's no complete frame
	if fb.IsLastFrameFullRedraw() {
		t.Error("Incomplete frame should not report full redraw")
	}
}

func TestFrameBuffer_IsLastFrameFullRedraw_OrphanEnd(t *testing.T) {
	fb := NewFrameBuffer(1000)
	fb.Write(append([]byte("orphan"), syncOutputEndSeq...))

	if fb.IsLastFrameFullRedraw() {
		t.Error("orphan synchronized-output end must not report a full redraw")
	}
}

// TestFrameBuffer_IsLastFrameFullRedraw_MultipleFrames tests with multiple frames
func TestFrameBuffer_IsLastFrameFullRedraw_MultipleFrames(t *testing.T) {
	fb := NewFrameBuffer(2000)

	// First frame: incremental
	frame1 := buildSyncFrame("frame 1 small")
	fb.Write(frame1)

	// Check: should not be full redraw (incremental)
	if fb.IsLastFrameFullRedraw() {
		t.Error("Last frame (incremental) should not be full redraw")
	}

	// Second frame: full redraw (with clear screen)
	frame2 := buildFullRedrawFrame("frame 2 full redraw")
	fb.Write(frame2)

	// Now last frame should be full redraw
	if !fb.IsLastFrameFullRedraw() {
		t.Error("Last frame (with clear screen) should be full redraw")
	}
}

// TestFrameBuffer_IsLastFrameFullRedraw_CompleteAndIncomplete tests mix of complete and incomplete
func TestFrameBuffer_IsLastFrameFullRedraw_CompleteAndIncomplete(t *testing.T) {
	fb := NewFrameBuffer(2000)

	// Complete full redraw frame
	completeFrame := buildFullRedrawFrame("complete frame")
	fb.Write(completeFrame)

	// Add incomplete frame at the end
	incomplete := append(syncOutputStartSeq, []byte("incomplete")...)
	fb.Write(incomplete)

	// Should check the last COMPLETE frame, which is the full redraw
	if !fb.IsLastFrameFullRedraw() {
		t.Error("Last complete frame is a full redraw, should return true")
	}
}
