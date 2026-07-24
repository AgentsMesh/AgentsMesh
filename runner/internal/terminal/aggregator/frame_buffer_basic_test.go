package aggregator

import (
	"bytes"
	"testing"
)

// Note: buildFullRedrawFrame is defined in frame_detector_helpers_test.go and shared across test files

func TestFrameBuffer_Write(t *testing.T) {
	fb := NewFrameBuffer(1000)

	fb.Write([]byte("hello"))
	fb.Write([]byte(" world"))

	if fb.Len() != 11 {
		t.Errorf("Expected length 11, got %d", fb.Len())
	}
	if string(fb.Bytes()) != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", fb.Bytes())
	}
}

func TestFrameBuffer_WritePreservesIncrementalFrames(t *testing.T) {
	fb := NewFrameBuffer(1000)

	// Write multiple sync frames - both should be kept (incremental updates)
	frame1 := buildSyncFrame("old frame")
	frame2 := buildSyncFrame("new frame")

	fb.Write(frame1)
	fb.Write(frame2)

	// Both frames should be kept (for incremental TUI updates like Claude Code)
	if !bytes.Contains(fb.Bytes(), []byte("old frame")) {
		t.Error("Old frame should be preserved for incremental updates")
	}
	if !bytes.Contains(fb.Bytes(), []byte("new frame")) {
		t.Error("New frame should be kept")
	}
}

func TestFrameBuffer_WriteRejectsWholeStreamOnOverflow(t *testing.T) {
	frame1 := buildSyncFrame("old frame content here")
	frame2 := buildSyncFrame("new frame content here")
	fb := NewFrameBuffer(len(frame1) + 10)

	if !fb.Write(frame1) {
		t.Fatal("first frame should fit")
	}
	if fb.Write(frame2) {
		t.Fatal("overflow must be reported to the snapshot recovery owner")
	}
	if fb.Len() != 0 {
		t.Fatalf("partial ANSI stream retained after overflow: %q", fb.Bytes())
	}
}

func TestFrameBuffer_EnforcesMaxSize(t *testing.T) {
	maxSize := 100
	fb := NewFrameBuffer(maxSize)

	data := bytes.Repeat([]byte("x"), 200)
	if fb.Write(data) {
		t.Fatal("oversized chunk unexpectedly accepted")
	}
	if fb.Len() != 0 {
		t.Fatalf("oversized chunk left %d bytes behind", fb.Len())
	}
}

func TestFrameBuffer_FlushComplete_AllComplete(t *testing.T) {
	fb := NewFrameBuffer(1000)

	frame := buildSyncFrame("content")
	fb.Write(frame)

	data, remaining := fb.FlushComplete()

	if !bytes.Equal(data, frame) {
		t.Errorf("Expected complete frame, got %q", data)
	}
	if remaining != 0 {
		t.Errorf("Expected 0 remaining, got %d", remaining)
	}
}

func TestFrameBuffer_FlushComplete_KeepsIncomplete(t *testing.T) {
	fb := NewFrameBuffer(1000)

	complete := buildSyncFrame("complete")
	incomplete := append(syncOutputStartSeq, []byte("incomplete")...)

	fb.Write(complete)
	fb.Write(incomplete)

	data, remaining := fb.FlushComplete()

	// Should flush complete frame
	if !bytes.Equal(data, complete) {
		t.Errorf("Expected complete frame to be flushed, got %q", data)
	}

	// Should keep incomplete frame
	if remaining != len(incomplete) {
		t.Errorf("Expected %d remaining bytes, got %d", len(incomplete), remaining)
	}
	if !bytes.Equal(fb.Bytes(), incomplete) {
		t.Errorf("Expected incomplete frame in buffer, got %q", fb.Bytes())
	}
}

func TestFrameBuffer_FlushComplete_OnlyIncomplete(t *testing.T) {
	fb := NewFrameBuffer(1000)

	incomplete := append(syncOutputStartSeq, []byte("incomplete")...)
	fb.Write(incomplete)

	data, remaining := fb.FlushComplete()

	// Should not flush incomplete frame
	if len(data) != 0 {
		t.Errorf("Should not flush incomplete frame, got %q", data)
	}
	if remaining != len(incomplete) {
		t.Errorf("Expected %d remaining, got %d", len(incomplete), remaining)
	}
}

func TestFrameBuffer_FlushComplete_PlainText(t *testing.T) {
	fb := NewFrameBuffer(1000)

	text := []byte("plain text without frames")
	fb.Write(text)

	data, remaining := fb.FlushComplete()

	// Plain text should be flushed entirely
	if !bytes.Equal(data, text) {
		t.Errorf("Expected plain text to be flushed, got %q", data)
	}
	if remaining != 0 {
		t.Errorf("Expected 0 remaining, got %d", remaining)
	}
}

func TestFrameBuffer_FlushAll(t *testing.T) {
	fb := NewFrameBuffer(1000)

	// Even incomplete frames should be flushed with FlushAll
	incomplete := append(syncOutputStartSeq, []byte("incomplete")...)
	fb.Write(incomplete)

	data, remaining := fb.FlushAll()

	if !bytes.Equal(data, incomplete) {
		t.Errorf("FlushAll should flush incomplete frame, got %q", data)
	}
	if remaining != 0 {
		t.Errorf("Expected 0 remaining after FlushAll, got %d", remaining)
	}
}

func TestFrameBuffer_Reset(t *testing.T) {
	fb := NewFrameBuffer(1000)

	fb.Write([]byte("some data"))
	fb.Reset()

	if fb.Len() != 0 {
		t.Errorf("Expected empty buffer after reset, got %d bytes", fb.Len())
	}
}

func TestFrameBuffer_MaxSizeWithFrames(t *testing.T) {
	maxSize := 200
	fb := NewFrameBuffer(maxSize)

	// Write multiple frames that exceed max size
	frame1 := buildSyncFrame(string(bytes.Repeat([]byte("1"), 50)))
	frame2 := buildSyncFrame(string(bytes.Repeat([]byte("2"), 50)))
	frame3 := buildSyncFrame(string(bytes.Repeat([]byte("3"), 50)))

	fb.Write(frame1)
	fb.Write(frame2)
	fb.Write(frame3)

	// Should keep only latest frame(s) within max size
	if fb.Len() > maxSize {
		t.Errorf("Buffer exceeded max size: %d > %d", fb.Len(), maxSize)
	}

	// Should contain latest frame
	if !bytes.Contains(fb.Bytes(), []byte("333")) {
		t.Error("Latest frame should be preserved")
	}
}

// TestFrameBuffer_FrameIntegrity tests that frames are kept complete during flush
func TestFrameBuffer_FrameIntegrity(t *testing.T) {
	fb := NewFrameBuffer(2000)

	// Simulate Claude Code pattern: multiple complete frames + incomplete
	frames := [][]byte{
		buildSyncFrame("Frame 1 content"),
		buildSyncFrame("Frame 2 content"),
		buildSyncFrame("Frame 3 content"),
	}
	incomplete := append(syncOutputStartSeq, []byte("Frame 4 partial...")...)

	for _, f := range frames {
		fb.Write(f)
	}
	fb.Write(incomplete)

	// Multiple flushes should maintain frame integrity
	totalStarts := 0
	totalEnds := 0

	for fb.Len() > 0 {
		data, _ := fb.FlushComplete()
		if len(data) == 0 {
			break
		}
		totalStarts += len(findAllPositions(data, syncOutputStartSeq))
		totalEnds += len(findAllPositions(data, syncOutputEndSeq))
	}

	// Flushed data should have matched starts and ends
	if totalStarts != totalEnds {
		t.Errorf("Frame integrity violated: %d starts, %d ends", totalStarts, totalEnds)
	}

	// Incomplete frame should remain in buffer
	remainingStarts := len(findAllPositions(fb.Bytes(), syncOutputStartSeq))
	remainingEnds := len(findAllPositions(fb.Bytes(), syncOutputEndSeq))

	if remainingStarts != 1 {
		t.Errorf("Expected 1 incomplete start in buffer, got %d", remainingStarts)
	}
	if remainingEnds != 0 {
		t.Errorf("Expected 0 ends in buffer (incomplete), got %d", remainingEnds)
	}
}

// TestFrameBuffer_WriteEmpty tests writing empty data
func TestFrameBuffer_WriteEmpty(t *testing.T) {
	fb := NewFrameBuffer(100)

	fb.Write(nil)
	if fb.Len() != 0 {
		t.Error("Write nil should not change buffer")
	}

	fb.Write([]byte{})
	if fb.Len() != 0 {
		t.Error("Write empty should not change buffer")
	}
}

// TestFrameBuffer_FlushComplete_Empty tests flushing empty buffer
func TestFrameBuffer_FlushComplete_Empty(t *testing.T) {
	fb := NewFrameBuffer(100)

	data, remaining := fb.FlushComplete()
	if data != nil {
		t.Error("FlushComplete on empty buffer should return nil")
	}
	if remaining != 0 {
		t.Errorf("Empty buffer should have 0 remaining, got %d", remaining)
	}
}

// TestFrameBuffer_FlushAll_Empty tests flushing all from empty buffer
func TestFrameBuffer_FlushAll_Empty(t *testing.T) {
	fb := NewFrameBuffer(100)

	data, remaining := fb.FlushAll()
	if data != nil {
		t.Error("FlushAll on empty buffer should return nil")
	}
	if remaining != 0 {
		t.Errorf("Empty buffer should have 0 remaining, got %d", remaining)
	}
}
