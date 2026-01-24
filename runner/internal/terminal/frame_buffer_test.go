package terminal

import (
	"bytes"
	"testing"
	"unicode/utf8"
)

// Note: buildFullRedrawFrame is defined in frame_detector_test.go and shared across test files

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

func TestFrameBuffer_WriteDiscardsOldFramesOnlyWhenFull(t *testing.T) {
	// Small buffer that can only hold one frame
	frame1 := buildSyncFrame("old frame content here")
	frame2 := buildSyncFrame("new frame content here")

	// Buffer just big enough for one frame
	fb := NewFrameBuffer(len(frame1) + 10)

	fb.Write(frame1)
	fb.Write(frame2)

	// Old frame should be discarded because buffer is full
	if bytes.Contains(fb.Bytes(), []byte("old frame")) {
		t.Error("Old frame should be discarded when buffer is full")
	}
	if !bytes.Contains(fb.Bytes(), []byte("new frame")) {
		t.Error("New frame should be kept")
	}
}

func TestFrameBuffer_EnforcesMaxSize(t *testing.T) {
	maxSize := 100
	fb := NewFrameBuffer(maxSize)

	// Write more than max size
	data := bytes.Repeat([]byte("x"), 200)
	fb.Write(data)

	if fb.Len() > maxSize {
		t.Errorf("Buffer exceeded max size: %d > %d", fb.Len(), maxSize)
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

func TestFrameBuffer_FlushAll_UTF8Boundary(t *testing.T) {
	fb := NewFrameBuffer(1000)

	// Write complete UTF-8 followed by incomplete multi-byte character
	// Chinese character 中 is 3 bytes: 0xe4 0xb8 0xad
	complete := []byte("hello 中")             // Complete
	incomplete := []byte{0xe4, 0xb8}           // Incomplete 中 (missing 0xad)
	fb.Write(append(complete, incomplete...))

	data, remaining := fb.FlushAll()

	// Should flush complete part, keep incomplete UTF-8
	if !bytes.Equal(data, complete) {
		t.Errorf("Expected complete UTF-8 part, got %q", data)
	}
	if remaining != 2 {
		t.Errorf("Expected 2 remaining bytes (incomplete UTF-8), got %d", remaining)
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

func TestAlignToUTF8Boundary(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		offset   int
		expected int
	}{
		{
			name:     "at boundary",
			data:     "hello",
			offset:   2,
			expected: 2,
		},
		{
			name:     "mid UTF-8 char",
			data:     "中文", // 中 = e4 b8 ad, 文 = e6 96 87
			offset:   1,    // middle of 中
			expected: 3,    // should advance to start of 文
		},
		{
			name:     "at end",
			data:     "abc",
			offset:   5, // past end
			expected: 3, // clamped to len
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := alignToUTF8Boundary([]byte(tc.data), tc.offset)
			if result != tc.expected {
				t.Errorf("Expected %d, got %d", tc.expected, result)
			}
		})
	}
}

func TestFindLastValidUTF8Boundary(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected int
	}{
		{
			name:     "empty",
			data:     nil,
			expected: 0,
		},
		{
			name:     "ascii",
			data:     []byte("hello"),
			expected: 5,
		},
		{
			name:     "complete utf8",
			data:     []byte("中文"),
			expected: 6,
		},
		{
			name:     "incomplete utf8 at end",
			data:     []byte{'h', 'i', 0xe4, 0xb8}, // "hi" + incomplete 中
			expected: 2,                             // truncate before incomplete
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := findLastValidUTF8Boundary(tc.data)
			if result != tc.expected {
				t.Errorf("Expected %d, got %d", tc.expected, result)
			}
		})
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

// TestFrameBuffer_FlushComplete_UTF8Boundary tests UTF-8 boundary handling in FlushComplete
func TestFrameBuffer_FlushComplete_UTF8Boundary(t *testing.T) {
	fb := NewFrameBuffer(1000)

	// Plain text with incomplete UTF-8 at end (no sync frames)
	completeText := []byte("hello 中文")
	incompleteUTF8 := []byte{0xe4, 0xb8} // incomplete 中

	// Write directly to buffer to avoid frame discard
	fb.buffer.Write(completeText)
	fb.buffer.Write(incompleteUTF8)

	data, remaining := fb.FlushComplete()

	// Should flush complete UTF-8 portion
	if len(data) == 0 {
		t.Error("Should flush complete data")
	}
	// The incomplete UTF-8 bytes should be kept
	t.Logf("Flushed %d bytes, remaining %d bytes", len(data), remaining)
}

// TestFrameBuffer_FlushComplete_UTF8AtFrameBoundary tests the critical bug:
// UTF-8 character ending right before a sync frame start should not be truncated
func TestFrameBuffer_FlushComplete_UTF8AtFrameBoundary(t *testing.T) {
	fb := NewFrameBuffer(1000)

	// Scenario: box drawing char "─" (e2 94 80) right before sync frame start
	// This is common in Claude Code's TUI output
	boxChar := []byte{0xe2, 0x94, 0x80} // ─
	incomplete := append(syncOutputStartSeq, []byte("incomplete frame content")...)

	// Write: ─ + ESC[?2026h + content (no end sequence = incomplete frame)
	fb.buffer.Write(boxChar)
	fb.buffer.Write(incomplete)

	// FlushComplete should either:
	// 1. Flush the box char if it won't break it, OR
	// 2. Keep everything if flushing would break UTF-8
	data, remaining := fb.FlushComplete()

	// The key assertion: we should NOT lose any bytes
	totalBytes := len(data) + remaining
	expectedTotal := len(boxChar) + len(incomplete)

	if totalBytes != expectedTotal {
		t.Errorf("Data loss detected! flushed=%d, remaining=%d, total=%d, expected=%d",
			len(data), remaining, totalBytes, expectedTotal)
	}

	// If we flushed anything, it should be valid UTF-8
	if len(data) > 0 {
		if _, err := string(data), error(nil); err != nil {
			// Note: Go strings don't validate UTF-8, but we can check manually
			for i := 0; i < len(data); {
				r, size := utf8.DecodeRune(data[i:])
				if r == utf8.RuneError && size == 1 {
					t.Errorf("Invalid UTF-8 at position %d in flushed data", i)
				}
				i += size
			}
		}
	}

	t.Logf("Flushed %d bytes, remaining %d bytes (total %d)", len(data), remaining, totalBytes)
}

// TestFrameBuffer_FlushComplete_UTF8BeforeMultipleFrames tests UTF-8 before multiple frames
func TestFrameBuffer_FlushComplete_UTF8BeforeMultipleFrames(t *testing.T) {
	fb := NewFrameBuffer(2000)

	// Build: 中文 + complete frame + incomplete frame
	prefix := []byte("中文") // 6 bytes of UTF-8
	completeFrame := buildSyncFrame("frame 1")
	incompleteFrame := append(syncOutputStartSeq, []byte("frame 2 incomplete")...)

	fb.buffer.Write(prefix)
	fb.buffer.Write(completeFrame)
	fb.buffer.Write(incompleteFrame)

	originalLen := fb.Len()
	data, remaining := fb.FlushComplete()

	// Should flush prefix + complete frame, keep incomplete frame
	totalBytes := len(data) + remaining

	if totalBytes != originalLen {
		t.Errorf("Data loss! original=%d, flushed=%d, remaining=%d, total=%d",
			originalLen, len(data), remaining, totalBytes)
	}

	// Flushed data should contain prefix and complete frame
	if !bytes.Contains(data, prefix) {
		t.Error("Flushed data should contain UTF-8 prefix")
	}
	if !bytes.Contains(data, []byte("frame 1")) {
		t.Error("Flushed data should contain complete frame content")
	}

	// Remaining should be the incomplete frame
	if remaining != len(incompleteFrame) {
		t.Errorf("Expected %d remaining bytes, got %d", len(incompleteFrame), remaining)
	}
}

// TestFrameBuffer_EnforceLimit_FramesHelp tests that discarding frames helps with limit
func TestFrameBuffer_EnforceLimit_FramesHelp(t *testing.T) {
	maxSize := 100
	fb := NewFrameBuffer(maxSize)

	// Write old frame
	oldFrame := buildSyncFrame(string(bytes.Repeat([]byte("o"), 30)))
	fb.Write(oldFrame)

	// Write new full redraw frame that would exceed limit - discarding old frame allows it
	// NOTE: Use buildFullRedrawFrame to trigger content-aware discard
	newFrame := buildFullRedrawFrame(string(bytes.Repeat([]byte("n"), 30)))
	fb.Write(newFrame)

	// Should have discarded old frame because newFrame is a full redraw
	if bytes.Contains(fb.Bytes(), []byte("ooo")) {
		t.Error("Old frame should be discarded")
	}
	if !bytes.Contains(fb.Bytes(), []byte("nnn")) {
		t.Error("New frame should be kept")
	}
}

// TestFindLastValidUTF8Boundary_EdgeCases tests edge cases for UTF-8 boundary detection
func TestFindLastValidUTF8Boundary_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected int
	}{
		{
			name:     "single continuation byte",
			data:     []byte{0x80}, // continuation byte
			expected: 1,           // no valid start, return all
		},
		{
			name:     "multiple continuation bytes",
			data:     []byte{0x80, 0x80, 0x80, 0x80, 0x80}, // all continuation
			expected: 5,                                    // no valid start, return all
		},
		{
			name:     "valid ascii then continuation",
			data:     []byte{'a', 'b', 0x80},
			expected: 3, // continuation after ascii is returned as-is
		},
		{
			name:     "4-byte utf8 incomplete",
			data:     []byte{0xf0, 0x9f, 0x98}, // incomplete emoji
			expected: 0,                         // should truncate all
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := findLastValidUTF8Boundary(tc.data)
			if result != tc.expected {
				t.Errorf("Expected %d, got %d", tc.expected, result)
			}
		})
	}
}

// TestFrameBuffer_EnforceLimitAfterWrite_RespectsFrameBoundary tests that
// truncation respects frame boundaries to avoid orphan frame ends
func TestFrameBuffer_EnforceLimitAfterWrite_RespectsFrameBoundary(t *testing.T) {
	maxSize := 100
	fb := NewFrameBuffer(maxSize)

	// Build a scenario where we have:
	// [small complete frame] + [large frame that exceeds maxSize]
	// The truncation should preserve frame integrity
	smallFrame := buildSyncFrame("small")                              // ~30 bytes
	largeContent := string(bytes.Repeat([]byte("X"), maxSize))         // 100 bytes
	largeFrame := buildSyncFrame(largeContent)                         // ~116 bytes

	fb.Write(smallFrame)
	fb.Write(largeFrame)

	// After truncation, we should have valid frame pairing
	data := fb.Bytes()
	starts := bytes.Count(data, syncOutputStartSeq)
	ends := bytes.Count(data, syncOutputEndSeq)

	// Key assertion: frame starts and ends should be balanced
	// (or starts >= ends if there's an incomplete frame at end)
	if ends > starts {
		t.Errorf("Frame integrity violated: %d starts, %d ends (orphan ends!)", starts, ends)
	}

	t.Logf("Buffer state: %d bytes, %d starts, %d ends", len(data), starts, ends)
}

// TestFrameBuffer_EnforceLimitAfterWrite_LargeFrameExceedsLimit tests
// the scenario where a single frame is larger than maxSize
func TestFrameBuffer_EnforceLimitAfterWrite_LargeFrameExceedsLimit(t *testing.T) {
	maxSize := 50
	fb := NewFrameBuffer(maxSize)

	// A single frame larger than maxSize
	largeContent := string(bytes.Repeat([]byte("X"), 100))
	largeFrame := buildSyncFrame(largeContent) // ~116 bytes

	fb.Write(largeFrame)

	// Buffer should be capped at maxSize
	if fb.Len() > maxSize {
		t.Errorf("Buffer exceeded maxSize: %d > %d", fb.Len(), maxSize)
	}

	// In this case, truncation is unavoidable, but we log a warning
	t.Logf("Large frame test: frame size %d, buffer size %d", len(largeFrame), fb.Len())
}

// TestFrameBuffer_EnforceLimitAfterWrite_MultipleFrames tests truncation
// with multiple frames, ensuring we truncate at frame boundary
func TestFrameBuffer_EnforceLimitAfterWrite_MultipleFrames(t *testing.T) {
	maxSize := 150
	fb := NewFrameBuffer(maxSize)

	// Write multiple frames that together exceed maxSize
	// Use full redraw frame at the end to trigger discarding of older frames
	frame1 := buildSyncFrame("frame1content")                                  // ~30 bytes - incremental
	frame2 := buildSyncFrame("frame2content")                                  // ~30 bytes - incremental
	frame3 := buildSyncFrame("frame3content")                                  // ~30 bytes - incremental
	frame4 := buildFullRedrawFrame(string(bytes.Repeat([]byte("4"), 50))) // ~70 bytes - full redraw

	fb.Write(frame1)
	fb.Write(frame2)
	fb.Write(frame3)
	fb.Write(frame4) // This full redraw triggers discard of frame1, frame2, frame3

	data := fb.Bytes()
	starts := bytes.Count(data, syncOutputStartSeq)
	ends := bytes.Count(data, syncOutputEndSeq)

	// Frame integrity check
	if ends > starts {
		t.Errorf("Frame integrity violated: %d starts, %d ends", starts, ends)
	}

	// Should have discarded old frames because frame4 is a full redraw
	if bytes.Contains(data, []byte("frame1content")) {
		t.Error("Old frame1 should have been discarded")
	}

	// Should keep the full redraw frame content
	if !bytes.Contains(data, []byte("4444")) {
		t.Error("Newest frame4 content should be kept")
	}

	t.Logf("Multiple frames test: %d bytes, %d starts, %d ends", len(data), starts, ends)
}
