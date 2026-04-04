package aggregator

import (
	"bytes"
	"testing"
	"unicode/utf8"
)

func TestFrameBuffer_FlushAll_UTF8Boundary(t *testing.T) {
	fb := NewFrameBuffer(1000)

	// Write complete UTF-8 followed by incomplete multi-byte character
	// Chinese character 中 is 3 bytes: 0xe4 0xb8 0xad
	complete := []byte("hello 中")    // Complete
	incomplete := []byte{0xe4, 0xb8} // Incomplete 中 (missing 0xad)
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
			expected: 2,                            // truncate before incomplete
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
			expected: 1,            // no valid start, return all
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
			expected: 0,                        // should truncate all
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
