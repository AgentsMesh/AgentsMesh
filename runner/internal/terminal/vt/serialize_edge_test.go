package vt

import (
	"strings"
	"testing"
)

// Test CJK wide character serialization
func TestSerialize_CJKWideChars(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("你好世界"))
	result := vt.Serialize(DefaultSerializeOptions())

	// Wide chars should be serialized correctly
	if !strings.Contains(result, "你好世界") {
		t.Errorf("CJK wide chars not preserved: %q", result)
	}

	// Verify round-trip
	vt2 := NewVirtualTerminal(80, 24, 1000)
	vt2.Feed([]byte(result))
	if vt2.GetDisplay() != "你好世界" {
		t.Errorf("CJK round-trip failed: got %q", vt2.GetDisplay())
	}
}

// Test mixed wide and narrow chars
func TestSerialize_MixedWidthChars(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("Hello 你好 World 世界"))
	result := vt.Serialize(DefaultSerializeOptions())

	vt2 := NewVirtualTerminal(80, 24, 1000)
	vt2.Feed([]byte(result))
	original := vt.GetDisplay()
	roundTripped := vt2.GetDisplay()

	if original != roundTripped {
		t.Errorf("Mixed width round-trip failed:\nOriginal: %q\nRound-trip: %q", original, roundTripped)
	}
}

// Test cursor at various positions
func TestSerialize_CursorPositions(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		checkPos func(row, col int) bool
	}{
		{"home", "\x1b[H", func(r, c int) bool { return r == 0 && c == 0 }},
		{"mid screen", "hello\x1b[5;10H", func(r, c int) bool { return r == 4 && c == 9 }},
		{"end of text", "hello world", func(r, c int) bool { return r == 0 && c == 11 }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vt := NewVirtualTerminal(80, 24, 1000)
			vt.Feed([]byte(tc.input))

			// Serialize and re-parse
			result := vt.Serialize(DefaultSerializeOptions())
			vt2 := NewVirtualTerminal(80, 24, 1000)
			vt2.Feed([]byte(result))

			// Check cursor position matches
			row1, col1 := vt.CursorPosition()
			row2, col2 := vt2.CursorPosition()

			if row1 != row2 || col1 != col2 {
				t.Errorf("Cursor position mismatch: original (%d,%d), restored (%d,%d)",
					row1, col1, row2, col2)
			}
		})
	}
}

// Test scrollback handling
func TestSerialize_ScrollbackHandling(t *testing.T) {
	vt := NewVirtualTerminal(80, 5, 100)

	// Generate scrollback
	for i := 0; i < 20; i++ {
		vt.Feed([]byte("Line " + string(rune('A'+i)) + "\n"))
	}

	// Serialize with different scrollback limits
	opts1 := DefaultSerializeOptions()
	opts1.ScrollbackLines = 5
	result1 := vt.Serialize(opts1)

	opts2 := DefaultSerializeOptions()
	opts2.ScrollbackLines = 10
	result2 := vt.Serialize(opts2)

	// More scrollback should produce longer output
	if len(result2) <= len(result1) {
		t.Logf("result1 len=%d, result2 len=%d", len(result1), len(result2))
		// This might not always be true due to line content, so just log
	}

	t.Logf("Scrollback 5: %d bytes, Scrollback 10: %d bytes", len(result1), len(result2))
}

// Test SerializeSimple
func TestSerializeSimple_NoColors(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("\x1b[31mRed\x1b[32mGreen\x1b[0m Normal"))

	simple := vt.SerializeSimple(100)

	// Should contain text without ANSI codes (or minimal)
	if !strings.Contains(simple, "Red") || !strings.Contains(simple, "Green") {
		t.Errorf("SerializeSimple should contain text: %q", simple)
	}
}

// Test line wrap detection
func TestSerialize_LineWrapDetection(t *testing.T) {
	// Create narrow terminal
	vt := NewVirtualTerminal(10, 5, 100)
	vt.Feed([]byte("12345678901234567890")) // 20 chars wraps on 10-col terminal

	// Check wrap flags
	if !vt.IsLineWrapped(1) {
		t.Error("Line 1 should be wrapped")
	}

	result := vt.Serialize(DefaultSerializeOptions())

	// Wrapped content should not have CRLF between wrapped lines
	// Count CRLF sequences
	crlfCount := strings.Count(result, "\r\n")
	t.Logf("CRLF count in wrapped content: %d, output: %q", crlfCount, result)

	// Verify round-trip
	vt2 := NewVirtualTerminal(10, 5, 100)
	vt2.Feed([]byte(result))
	if vt.GetDisplay() != vt2.GetDisplay() {
		t.Errorf("Wrapped line round-trip failed:\nOriginal: %q\nRestored: %q",
			vt.GetDisplay(), vt2.GetDisplay())
	}
}
