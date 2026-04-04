package vt

import (
	"testing"
)

func TestEraseToEndOfLine(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("Hello, World!"))
	vt.Feed([]byte("\x1b[7D")) // Back 7 chars
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

func TestDeleteChars(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("ABCDEF"))
	vt.Feed([]byte("\x1b[4D")) // Back 4
	vt.Feed([]byte("\x1b[2P")) // Delete 2 chars
	display := vt.GetDisplay()
	if display != "ABEF" {
		t.Errorf("expected 'ABEF', got '%s'", display)
	}
}

func TestInsertChars(t *testing.T) {
	vt := NewVirtualTerminal(80, 24, 1000)
	vt.Feed([]byte("ABCD"))
	vt.Feed([]byte("\x1b[2D")) // Back 2
	vt.Feed([]byte("\x1b[2@")) // Insert 2 chars
	vt.Feed([]byte("XY"))
	display := vt.GetDisplay()
	if display != "ABXYCD" {
		t.Errorf("expected 'ABXYCD', got '%s'", display)
	}
}
