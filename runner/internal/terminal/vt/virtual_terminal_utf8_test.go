package vt

import (
	"testing"
)

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
	vt.Feed([]byte("世界"))      // Overwrites from column 0
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
