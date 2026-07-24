package vt

import (
	"strings"
	"testing"
)

func TestNewVirtualTerminal(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	if vt.cols != 80 {
		t.Errorf("expected cols=80, got %d", vt.cols)
	}
	if vt.rows != 24 {
		t.Errorf("expected rows=24, got %d", vt.rows)
	}
}

func TestBasicText(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("Hello, World!"))
	display := vt.GetDisplay()
	if display != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got '%s'", display)
	}
}

func TestNewLine(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("Line 1\nLine 2"))
	display := vt.GetDisplay()
	expected := "Line 1\nLine 2"
	if display != expected {
		t.Errorf("expected '%s', got '%s'", expected, display)
	}
}

func TestCarriageReturn(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("Hello\rWorld"))
	display := vt.GetDisplay()
	if display != "World" {
		t.Errorf("expected 'World' (CR overwrites), got '%s'", display)
	}
}

func TestSGRIgnored(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("\x1b[31mRed\x1b[0m Normal"))
	display := vt.GetDisplay()
	if display != "Red Normal" {
		t.Errorf("expected 'Red Normal' (colors stripped), got '%s'", display)
	}
}

func TestAlternativeScreenBuffer(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("Main Screen"))
	vt.Feed([]byte("\x1b[?1049h")) // Enter alt screen
	vt.Feed([]byte("Alt Screen"))
	display := vt.GetDisplay()
	// xterm carries the active cursor into a newly activated alternate buffer.
	if display != "           Alt Screen" {
		t.Errorf("expected alternate output at the inherited cursor, got %q", display)
	}

	vt.Feed([]byte("\x1b[?1049l")) // Exit alt screen
	display = vt.GetDisplay()
	if display != "Main Screen" {
		t.Errorf("expected 'Main Screen' after exit, got '%s'", display)
	}
}

func TestOSCSequence(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	// OSC to set window title, should be ignored
	vt.Feed([]byte("\x1b]0;My Title\x07"))
	vt.Feed([]byte("Hello"))
	display := vt.GetDisplay()
	if display != "Hello" {
		t.Errorf("expected 'Hello' (OSC ignored), got '%s'", display)
	}
}

func TestScrollUp(t *testing.T) {
	vt := NewVirtualTerminal(80, 5, 1000)
	vt.Feed([]byte("Line 1\nLine 2\nLine 3\nLine 4\nLine 5"))
	vt.Feed([]byte("\x1b[2S")) // Scroll up 2 lines
	display := vt.GetDisplay()
	expected := "Line 3\nLine 4\nLine 5"
	if display != expected {
		t.Errorf("expected '%s', got '%s'", expected, display)
	}
}

func TestTabCharacter(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("A\tB"))
	display := vt.GetDisplay()
	// Tab should move to column 8
	expected := "A       B"
	if display != expected {
		t.Errorf("expected '%s', got '%s'", expected, display)
	}
}

func TestBackspace(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("ABC\b\b"))
	vt.Feed([]byte("XY"))
	display := vt.GetDisplay()
	if display != "AXY" {
		t.Errorf("expected 'AXY', got '%s'", display)
	}
}

func TestComplexSequence(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	// Simulate a progress bar update
	vt.Feed([]byte("Progress: [          ] 0%"))
	vt.Feed([]byte("\r")) // Return to start
	vt.Feed([]byte("Progress: [##        ] 20%"))
	display := vt.GetDisplay()
	if display != "Progress: [##        ] 20%" {
		t.Errorf("expected progress update, got '%s'", display)
	}
}

func TestMultipleCSIParams(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("123456789"))
	vt.Feed([]byte("\x1b[1;5H")) // Row 1, Col 5
	vt.Feed([]byte("X"))
	display := vt.GetDisplay()
	if display != "1234X6789" {
		t.Errorf("expected '1234X6789', got '%s'", display)
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"\x1b[31mRed\x1b[0m", "Red"},
		{"\x1b[1;32;40mGreen on black\x1b[0m", "Green on black"},
		{"\x1b]0;Title\x07Content", "Content"},
		{"No escape", "No escape"},
		// DEC private mode sequences
		{"\x1b[?2026hHello\x1b[?2026l", "Hello"},
		{"\x1b[?25lHidden cursor\x1b[?25h", "Hidden cursor"},
		{"\x1b[?2004hBracketed paste\x1b[?2004l", "Bracketed paste"},
		{"\x1b[?1004hFocus reporting", "Focus reporting"},
		// Kitty keyboard protocol
		{"\x1b[>1uKitty mode", "Kitty mode"},
		// Complex mixed sequences
		{"\x1b[?2026h\x1b[?25l\r\nHello\r\n\x1b[?2026l\x1b[?25h", "\r\nHello\r\n"},
	}

	for _, tt := range tests {
		result := StripANSI(tt.input)
		if result != tt.expected {
			t.Errorf("StripANSI(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestGetOutput(t *testing.T) {
	vt := NewVirtualTerminal(80, 5, 10)
	// Feed more than screen height
	vt.Feed([]byte("Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7"))
	output := vt.GetOutput(5)
	// Should get last 5 lines including history
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestHistory(t *testing.T) {
	vt := NewVirtualTerminal(80, 3, 100)
	vt.Feed([]byte("A\nB\nC\nD\nE"))
	// Lines A and B should be in history
	output := vt.GetOutput(10)
	if output == "" {
		t.Error("expected non-empty output with history")
	}
}

func TestResize(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("Hello"))
	vt.Resize(40, 12)
	if vt.Cols() != 40 || vt.Rows() != 12 {
		t.Errorf("expected 40x12, got %dx%d", vt.Cols(), vt.Rows())
	}
}

func TestResizeSameSizePreservesBothScreenBuffers(t *testing.T) {
	vt := NewVirtualTerminal(20, 4, 1000)
	vt.Feed([]byte("main content"))
	mainBefore := vt.GetDisplay()
	mainRow, mainCol := vt.CursorPosition()

	vt.Feed([]byte("\x1b[?1049h"))
	vt.Feed([]byte("alternate content"))
	altBefore := vt.GetDisplay()
	altRow, altCol := vt.CursorPosition()

	vt.Resize(20, 4)

	if got := vt.GetDisplay(); got != altBefore {
		t.Fatalf("same-size resize cleared alternate screen: got %q, want %q", got, altBefore)
	}
	if row, col := vt.CursorPosition(); row != altRow || col != altCol {
		t.Fatalf("same-size resize moved alternate cursor: got (%d,%d), want (%d,%d)", row, col, altRow, altCol)
	}

	vt.Feed([]byte("\x1b[?1049l"))
	if got := vt.GetDisplay(); got != mainBefore {
		t.Fatalf("same-size resize damaged saved main screen: got %q, want %q", got, mainBefore)
	}
	if row, col := vt.CursorPosition(); row != mainRow || col != mainCol {
		t.Fatalf("same-size resize moved main cursor: got (%d,%d), want (%d,%d)", row, col, mainRow, mainCol)
	}
}

func TestClear(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("Hello, World!"))
	vt.Clear()
	display := vt.GetDisplay()
	if display != "" {
		t.Errorf("expected empty after clear, got '%s'", display)
	}
}

func TestVTDECPrivateModeStripping(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)

	// Simulate Claude Code output with DEC private mode sequences
	data := []byte("\x1b[?2026h\x1b[?25l\r\n──────────────────────────────────────────────────────────────────────────────\r\n Do you trust the files in this folder?\r\n\r\n /private/tmp/test\r\n\r\n ❯ 1. Yes, proceed\r\n   2. No, exit\r\n\r\n Enter to confirm · Esc to cancel\r\n\x1b[?2026l\x1b[?25l\x1b[?2004h\x1b[?1004h\x1b[>1u")

	vt.Feed(data)

	display := vt.GetDisplay()

	// Check that ESC sequences are stripped
	if strings.Contains(display, "\x1b") {
		t.Errorf("GetDisplay() should not contain ESC sequences, got: %q", display)
	}

	// Check content is preserved
	if !strings.Contains(display, "Do you trust") {
		t.Errorf("GetDisplay() should contain 'Do you trust', got: %q", display)
	}

	if !strings.Contains(display, "──────") {
		t.Errorf("GetDisplay() should contain box drawing chars, got: %q", display)
	}

	t.Logf("Display output:\n%s", display)
}
