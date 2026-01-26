package terminal

import (
	"strings"
	"sync"
	"testing"
	"time"
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

func TestCursorUp(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("Line 1\nLine 2"))
	vt.Feed([]byte("\x1b[1A")) // Cursor up 1
	vt.Feed([]byte("X"))
	display := vt.GetDisplay()
	// Cursor is at col 6 (after "Line 2"), then move up to line 0
	// and write X at that position (col 6)
	expected := "Line 1X\nLine 2"
	if display != expected {
		t.Errorf("expected '%s', got '%s'", expected, display)
	}
}

func TestCursorDown(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("Line 1"))
	vt.Feed([]byte("\x1b[1B")) // Cursor down 1
	vt.Feed([]byte("X"))
	display := vt.GetDisplay()
	expected := "Line 1\n      X"
	if display != expected {
		t.Errorf("expected '%s', got '%s'", expected, display)
	}
}

func TestCursorForward(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("AB"))
	vt.Feed([]byte("\x1b[3C")) // Cursor forward 3
	vt.Feed([]byte("X"))
	display := vt.GetDisplay()
	expected := "AB   X"
	if display != expected {
		t.Errorf("expected '%s', got '%s'", expected, display)
	}
}

func TestCursorBack(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("ABCD"))
	vt.Feed([]byte("\x1b[2D")) // Cursor back 2
	vt.Feed([]byte("XY"))
	display := vt.GetDisplay()
	if display != "ABXY" {
		t.Errorf("expected 'ABXY', got '%s'", display)
	}
}

func TestCursorPosition(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("ABC\nDEF\nGHI"))
	vt.Feed([]byte("\x1b[2;2H")) // Row 2, Col 2
	vt.Feed([]byte("X"))
	display := vt.GetDisplay()
	expected := "ABC\nDXF\nGHI"
	if display != expected {
		t.Errorf("expected '%s', got '%s'", expected, display)
	}
}

func TestEraseToEndOfLine(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("Hello, World!"))
	vt.Feed([]byte("\x1b[7D"))  // Back 7 chars
	vt.Feed([]byte("\x1b[0K")) // Erase to end of line
	display := vt.GetDisplay()
	if display != "Hello," {
		t.Errorf("expected 'Hello,', got '%s'", display)
	}
}

func TestEraseEntireLine(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("Line 1\nLine 2\nLine 3"))
	vt.Feed([]byte("\x1b[2;1H")) // Go to line 2
	vt.Feed([]byte("\x1b[2K"))   // Erase entire line
	display := vt.GetDisplay()
	expected := "Line 1\n\nLine 3"
	if display != expected {
		t.Errorf("expected '%s', got '%s'", expected, display)
	}
}

func TestEraseScreen(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("Hello, World!"))
	vt.Feed([]byte("\x1b[2J")) // Erase entire screen
	display := vt.GetDisplay()
	if display != "" {
		t.Errorf("expected empty screen, got '%s'", display)
	}
}

func TestSaveCursorRestore(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("AB"))
	vt.Feed([]byte("\x1b[s")) // Save cursor
	vt.Feed([]byte("CD"))
	vt.Feed([]byte("\x1b[u")) // Restore cursor
	vt.Feed([]byte("XY"))
	display := vt.GetDisplay()
	if display != "ABXY" {
		t.Errorf("expected 'ABXY', got '%s'", display)
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
	if display != "Alt Screen" {
		t.Errorf("expected 'Alt Screen', got '%s'", display)
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

func TestDeleteChars(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("ABCDEF"))
	vt.Feed([]byte("\x1b[4D"))  // Back 4
	vt.Feed([]byte("\x1b[2P")) // Delete 2 chars
	display := vt.GetDisplay()
	if display != "ABEF" {
		t.Errorf("expected 'ABEF', got '%s'", display)
	}
}

func TestInsertChars(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("ABCD"))
	vt.Feed([]byte("\x1b[2D"))  // Back 2
	vt.Feed([]byte("\x1b[2@")) // Insert 2 chars
	vt.Feed([]byte("XY"))
	display := vt.GetDisplay()
	if display != "ABXYCD" {
		t.Errorf("expected 'ABXYCD', got '%s'", display)
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

func TestClear(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("Hello, World!"))
	vt.Clear()
	display := vt.GetDisplay()
	if display != "" {
		t.Errorf("expected empty after clear, got '%s'", display)
	}
}

func TestCursorPosition_Method(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("ABC\nDEF"))
	row, col := vt.CursorPosition()
	if row != 1 || col != 3 {
		t.Errorf("expected cursor at (1, 3), got (%d, %d)", row, col)
	}
}

func TestDecSaveCursorRestore(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("AB"))
	vt.Feed([]byte("\x1b7")) // DEC Save cursor
	vt.Feed([]byte("CD"))
	vt.Feed([]byte("\x1b8")) // DEC Restore cursor
	vt.Feed([]byte("XY"))
	display := vt.GetDisplay()
	if display != "ABXY" {
		t.Errorf("expected 'ABXY', got '%s'", display)
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

// UTF-8 Multi-byte character tests

func TestUTF8ChineseCharacters(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("你好世界"))
	display := vt.GetDisplay()
	if display != "你好世界" {
		t.Errorf("Chinese characters corrupted: expected '你好世界', got '%s'", display)
	}
}

func TestUTF8BoxDrawingCharacters(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("┌───┐"))
	display := vt.GetDisplay()
	if display != "┌───┐" {
		t.Errorf("Box drawing characters corrupted: expected '┌───┐', got '%s'", display)
	}
}

func TestUTF8BoxDrawingTable(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	input := "┌─────┬─────┐\n│ A   │ B   │\n├─────┼─────┤\n│ C   │ D   │\n└─────┴─────┘"
	vt.Feed([]byte(input))
	display := vt.GetDisplay()
	if display != input {
		t.Errorf("Box drawing table corrupted:\nexpected:\n%s\ngot:\n%s", input, display)
	}
}

func TestUTF8MixedWithANSI(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("\x1b[31m中文\x1b[0m English"))
	display := vt.GetDisplay()
	expected := "中文 English"
	if display != expected {
		t.Errorf("Mixed UTF-8 and ANSI corrupted: expected '%s', got '%s'", expected, display)
	}
}

func TestUTF8Emoji(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("Hello 🚀 World"))
	display := vt.GetDisplay()
	if display != "Hello 🚀 World" {
		t.Errorf("Emoji corrupted: expected 'Hello 🚀 World', got '%s'", display)
	}
}

func TestUTF8Japanese(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("こんにちは"))
	display := vt.GetDisplay()
	if display != "こんにちは" {
		t.Errorf("Japanese characters corrupted: expected 'こんにちは', got '%s'", display)
	}
}

func TestUTF8MixedScripts(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("English 中文 日本語 한국어"))
	display := vt.GetDisplay()
	expected := "English 中文 日本語 한국어"
	if display != expected {
		t.Errorf("Mixed scripts corrupted: expected '%s', got '%s'", expected, display)
	}
}

func TestUTF8SpecialSymbols(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("• bullet · middle dot ─ box"))
	display := vt.GetDisplay()
	expected := "• bullet · middle dot ─ box"
	if display != expected {
		t.Errorf("Special symbols corrupted: expected '%s', got '%s'", expected, display)
	}
}

func TestUTF8WithCursorMovement(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("你好"))
	// Cursor is at column 4 (after two wide chars)
	vt.Feed([]byte("\x1b[4D")) // Cursor back 4 columns to start
	vt.Feed([]byte("世界"))    // Overwrites from column 0
	display := vt.GetDisplay()
	if display != "世界" {
		t.Errorf("UTF-8 with cursor movement: expected '世界', got '%s'", display)
	}
}

func TestUTF8InProgressBar(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	// Simulate a progress bar with box drawing characters
	vt.Feed([]byte("进度: [████░░░░░░] 40%"))
	display := vt.GetDisplay()
	expected := "进度: [████░░░░░░] 40%"
	if display != expected {
		t.Errorf("Progress bar with UTF-8: expected '%s', got '%s'", expected, display)
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

// ==================== OSC Handler Tests ====================

func TestOSC777NotifyHandler(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)

	var wg sync.WaitGroup
	wg.Add(1)

	var gotOSCType int
	var gotParams []string

	vt.SetOSCHandler(func(oscType int, params []string) {
		gotOSCType = oscType
		gotParams = params
		wg.Done()
	})

	// OSC 777;notify;Title;Body BEL
	vt.Feed([]byte("\x1b]777;notify;Build Complete;Success!\x07"))

	// Wait for goroutine with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Handler was called
	case <-time.After(100 * time.Millisecond):
		t.Fatal("OSC handler was not called within timeout")
	}

	if gotOSCType != 777 {
		t.Errorf("expected OSC type 777, got %d", gotOSCType)
	}
	if len(gotParams) != 3 {
		t.Errorf("expected 3 params, got %d: %v", len(gotParams), gotParams)
	}
	if len(gotParams) >= 3 {
		if gotParams[0] != "notify" {
			t.Errorf("expected params[0]='notify', got '%s'", gotParams[0])
		}
		if gotParams[1] != "Build Complete" {
			t.Errorf("expected params[1]='Build Complete', got '%s'", gotParams[1])
		}
		if gotParams[2] != "Success!" {
			t.Errorf("expected params[2]='Success!', got '%s'", gotParams[2])
		}
	}
}

func TestOSC9NotificationHandler(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)

	var wg sync.WaitGroup
	wg.Add(1)

	var gotOSCType int
	var gotParams []string

	vt.SetOSCHandler(func(oscType int, params []string) {
		gotOSCType = oscType
		gotParams = params
		wg.Done()
	})

	// OSC 9;message BEL (ConEmu/Windows Terminal format)
	vt.Feed([]byte("\x1b]9;Task completed\x07"))

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("OSC handler was not called within timeout")
	}

	if gotOSCType != 9 {
		t.Errorf("expected OSC type 9, got %d", gotOSCType)
	}
	if len(gotParams) != 1 {
		t.Errorf("expected 1 param, got %d: %v", len(gotParams), gotParams)
	}
	if len(gotParams) >= 1 && gotParams[0] != "Task completed" {
		t.Errorf("expected params[0]='Task completed', got '%s'", gotParams[0])
	}
}

func TestOSC0TitleHandler(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)

	var wg sync.WaitGroup
	wg.Add(1)

	var gotOSCType int
	var gotParams []string

	vt.SetOSCHandler(func(oscType int, params []string) {
		gotOSCType = oscType
		gotParams = params
		wg.Done()
	})

	// OSC 0;title BEL (window title)
	vt.Feed([]byte("\x1b]0;My Terminal Title\x07"))

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("OSC handler was not called within timeout")
	}

	if gotOSCType != 0 {
		t.Errorf("expected OSC type 0, got %d", gotOSCType)
	}
	if len(gotParams) != 1 {
		t.Errorf("expected 1 param, got %d: %v", len(gotParams), gotParams)
	}
	if len(gotParams) >= 1 && gotParams[0] != "My Terminal Title" {
		t.Errorf("expected params[0]='My Terminal Title', got '%s'", gotParams[0])
	}
}

func TestOSC2TitleHandler(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)

	var wg sync.WaitGroup
	wg.Add(1)

	var gotOSCType int
	var gotParams []string

	vt.SetOSCHandler(func(oscType int, params []string) {
		gotOSCType = oscType
		gotParams = params
		wg.Done()
	})

	// OSC 2;title BEL (window title only)
	vt.Feed([]byte("\x1b]2;Another Title\x07"))

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("OSC handler was not called within timeout")
	}

	if gotOSCType != 2 {
		t.Errorf("expected OSC type 2, got %d", gotOSCType)
	}
	if len(gotParams) != 1 {
		t.Errorf("expected 1 param, got %d: %v", len(gotParams), gotParams)
	}
	if len(gotParams) >= 1 && gotParams[0] != "Another Title" {
		t.Errorf("expected params[0]='Another Title', got '%s'", gotParams[0])
	}
}

func TestOSCHandlerNilSafe(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)

	// No handler set - should not panic
	vt.Feed([]byte("\x1b]777;notify;Title;Body\x07"))

	// If we get here without panic, test passes
	display := vt.GetDisplay()
	if display != "" {
		t.Errorf("expected empty display (OSC should not be visible), got '%s'", display)
	}
}

func TestOSCWithSTTerminator(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)

	var wg sync.WaitGroup
	wg.Add(1)

	var gotOSCType int

	vt.SetOSCHandler(func(oscType int, params []string) {
		gotOSCType = oscType
		wg.Done()
	})

	// OSC with ST terminator (ESC \) instead of BEL
	vt.Feed([]byte("\x1b]0;Title with ST\x1b\\"))

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("OSC handler was not called with ST terminator")
	}

	if gotOSCType != 0 {
		t.Errorf("expected OSC type 0, got %d", gotOSCType)
	}
}

func TestOSCMixedWithNormalOutput(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)

	var wg sync.WaitGroup
	wg.Add(1)

	var oscCount int

	vt.SetOSCHandler(func(oscType int, params []string) {
		oscCount++
		wg.Done()
	})

	// Mix OSC sequences with normal text
	vt.Feed([]byte("Hello\x1b]9;notification\x07World"))

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("OSC handler was not called within timeout")
	}

	if oscCount != 1 {
		t.Errorf("expected 1 OSC call, got %d", oscCount)
	}

	display := vt.GetDisplay()
	if display != "HelloWorld" {
		t.Errorf("expected 'HelloWorld', got '%s'", display)
	}
}
