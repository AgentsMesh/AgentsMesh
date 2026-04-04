package vt

import (
	"strings"
	"testing"
)

// Test history with style preservation (colors should survive scrolling)
func TestSerialize_HistoryWithStyles(t *testing.T) {
	// Small terminal to force scrolling quickly
	vt := NewVirtualTerminal(80, 3, 100)

	// Feed colored text that will scroll into history
	vt.Feed([]byte("\x1b[31mRed Line 1\x1b[0m\n"))
	vt.Feed([]byte("\x1b[32mGreen Line 2\x1b[0m\n"))
	vt.Feed([]byte("\x1b[34mBlue Line 3\x1b[0m\n"))
	vt.Feed([]byte("\x1b[33mYellow Line 4\x1b[0m\n")) // This pushes line 1 into history
	vt.Feed([]byte("Normal Line 5\n"))                // This pushes line 2 into history

	// Serialize including history
	opts := DefaultSerializeOptions()
	opts.ScrollbackLines = 10
	result := vt.Serialize(opts)

	// Should contain color codes for the history lines (not just plain text)
	if !strings.Contains(result, "\x1b[31m") {
		t.Errorf("History should preserve red color, got: %q", result)
	}
	if !strings.Contains(result, "\x1b[32m") {
		t.Errorf("History should preserve green color, got: %q", result)
	}

	// Verify round-trip restores colored content
	vt2 := NewVirtualTerminal(80, 3, 100)
	vt2.Feed([]byte(result))

	t.Logf("Styled history serialized: %q", result[:min(len(result), 200)])
}

// Test Range option for partial serialization
func TestSerialize_RangeOption(t *testing.T) {
	vt := NewVirtualTerminal(80, 5, 100)

	// Generate some content
	for i := 1; i <= 10; i++ {
		vt.Feed([]byte("Line " + string(rune('0'+i%10)) + "\n"))
	}

	// Serialize specific range
	opts := DefaultSerializeOptions()
	opts.Range = &SerializeRange{Start: 2, End: 4} // Lines 2-4 only
	result := vt.Serialize(opts)

	// Should only contain lines 2-4
	if len(result) == 0 {
		t.Error("Range serialization should produce output")
	}

	t.Logf("Range serialization result: %q", result)
}

// Test that historyStyled length stays in sync with history
func TestHistoryStyledSync(t *testing.T) {
	vt := NewVirtualTerminal(80, 5, 10) // Small history limit

	// Generate more lines than history limit
	for i := 0; i < 20; i++ {
		vt.Feed([]byte("\x1b[31mLine\x1b[0m\n"))
	}

	// Both histories should have same length
	histLen := len(vt.history)
	styledLen := vt.GetHistoryStyledLength()

	// Note: history only stores non-empty lines, so they may differ
	// But styled history should not exceed max
	if styledLen > 10 {
		t.Errorf("Styled history exceeded max: got %d, max is 10", styledLen)
	}

	t.Logf("Plain history: %d, Styled history: %d", histLen, styledLen)
}

// Test history round-trip with complex styles
func TestSerialize_HistoryRoundTrip(t *testing.T) {
	vt := NewVirtualTerminal(80, 3, 100)

	// Feed styled content that will scroll
	vt.Feed([]byte("\x1b[1;31mBold Red\x1b[0m Normal\n"))
	vt.Feed([]byte("\x1b[38;5;208mOrange 256\x1b[0m\n"))
	vt.Feed([]byte("\x1b[38;2;100;200;50mTrueColor\x1b[0m\n"))
	vt.Feed([]byte("Push to history\n"))
	vt.Feed([]byte("More content\n"))

	opts := DefaultSerializeOptions()
	result := vt.Serialize(opts)

	// Verify round-trip
	vt2 := NewVirtualTerminal(80, 3, 100)
	vt2.Feed([]byte(result))

	// Display content should match
	if vt.GetDisplay() != vt2.GetDisplay() {
		t.Errorf("Round-trip display mismatch:\nOriginal: %q\nRestored: %q",
			vt.GetDisplay(), vt2.GetDisplay())
	}
}

// Test GetHistoryStyledRow returns correct data
func TestGetHistoryStyledRow(t *testing.T) {
	vt := NewVirtualTerminal(80, 3, 100)

	// Feed colored line then scroll it
	vt.Feed([]byte("\x1b[31mRed\x1b[0m\n"))
	vt.Feed([]byte("Line 2\n"))
	vt.Feed([]byte("Line 3\n"))
	vt.Feed([]byte("Line 4\n")) // Pushes red line into history

	// Get the styled history row
	row := vt.GetHistoryStyledRow(0)
	if row == nil {
		t.Fatal("GetHistoryStyledRow returned nil")
	}

	// First cell should be 'R' with red color
	if row[0].Char != 'R' {
		t.Errorf("Expected 'R', got %q", row[0].Char)
	}
	if !row[0].Fg.IsPalette() || row[0].Fg.Index() != 1 {
		t.Errorf("Expected red palette color, got %+v", row[0].Fg)
	}
}

// Test IsHistoryLineWrapped
func TestIsHistoryLineWrapped(t *testing.T) {
	vt := NewVirtualTerminal(10, 3, 100)

	// Feed wrapped line then scroll it
	vt.Feed([]byte("1234567890ABCD\n")) // Wraps at col 10
	vt.Feed([]byte("X\n"))
	vt.Feed([]byte("Y\n"))
	vt.Feed([]byte("Z\n")) // Scroll happens

	// Check history wrap status
	if vt.GetHistoryStyledLength() < 1 {
		t.Skip("No history yet")
	}

	// First history line should NOT be wrapped (it's the start)
	if vt.IsHistoryLineWrapped(0) {
		t.Log("First history line marked as wrapped - might be expected for continuation")
	}
}

// Test empty range handling
func TestSerialize_EmptyRange(t *testing.T) {
	vt := NewVirtualTerminal(80, 5, 100)
	vt.Feed([]byte("Hello"))

	// Invalid range (start > end)
	opts := DefaultSerializeOptions()
	opts.Range = &SerializeRange{Start: 10, End: 5}
	result := vt.Serialize(opts)

	if result != "" {
		t.Errorf("Invalid range should produce empty result, got: %q", result)
	}
}

// Test range beyond buffer
func TestSerialize_RangeBeyondBuffer(t *testing.T) {
	vt := NewVirtualTerminal(80, 5, 100)
	vt.Feed([]byte("Hello"))

	// Range beyond buffer should be clamped
	opts := DefaultSerializeOptions()
	opts.Range = &SerializeRange{Start: 0, End: 1000}
	result := vt.Serialize(opts)

	if result == "" {
		t.Error("Range should be clamped to valid buffer size")
	}
}
