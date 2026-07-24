package vt

import (
	"strings"
	"testing"
)

func TestSerializeRowsScreenContract(t *testing.T) {
	terminal := NewVirtualTerminal(6, 2, 10)
	terminal.Feed([]byte("abc\r\nxy"))

	handler := newStringSerializeHandler(terminal)
	serialized := handler.serialize(0, terminal.rows-1, false)

	if !strings.Contains(serialized, "abc\r\nxy") {
		t.Fatalf("serialized screen lost row content: %q", serialized)
	}
	if !strings.Contains(serialized, "\x1b[2;3H") {
		t.Fatalf("serialized screen lost cursor position: %q", serialized)
	}
}

func TestSerializeRowsPreservesPhysicalWrap(t *testing.T) {
	terminal := NewVirtualTerminal(3, 2, 10)
	terminal.Feed([]byte("abcd"))
	if !terminal.isWrapped[1] {
		t.Fatal("fixture did not create a wrapped continuation")
	}

	handler := newStringSerializeHandler(terminal)
	serialized := handler.serialize(0, terminal.rows-1, true)

	if strings.Contains(serialized, "\r\n") {
		t.Fatalf("wrapped rows were separated by CRLF: %q", serialized)
	}
	if !strings.Contains(serialized, "abcd") {
		t.Fatalf("wrapped row content was not materialized: %q", serialized)
	}
}

func TestSerializeRowsEmitsTrailingBackgroundErase(t *testing.T) {
	terminal := NewVirtualTerminal(3, 1, 0)
	handler := newStringSerializeHandler(terminal)
	handler.allRows = make([]string, 1)
	handler.allRowSeparators = make([]string, 1)
	handler.cursorStyle.Bg = PaletteColor(1)
	handler.nullCellCount = 2

	handler.rowEnd(0, true)

	if got := handler.allRows[0]; got != "\x1b[2X" {
		t.Fatalf("trailing colored null cells = %q, want ECH", got)
	}
}

func TestSerializeStringBoundsAndStyleRestoration(t *testing.T) {
	t.Run("clamps negative content extent", func(t *testing.T) {
		terminal := NewVirtualTerminal(4, 4, 0)
		handler := newStringSerializeHandler(terminal)
		handler.allRows = []string{"ignored"}
		handler.allRowSeparators = []string{""}
		handler.firstRow = 3
		handler.lastContentCursorRow = 0

		if got := handler.serializeString(3, 3, true); got != "" {
			t.Fatalf("negative extent serialization = %q, want empty", got)
		}
	})

	t.Run("clamps oversized content extent and restores style", func(t *testing.T) {
		terminal := NewVirtualTerminal(4, 4, 0)
		terminal.Feed([]byte("\x1b[31m"))
		handler := newStringSerializeHandler(terminal)
		handler.allRows = []string{"first", "second"}
		handler.allRowSeparators = []string{"|", ""}
		handler.firstRow = 0
		handler.lastContentCursorRow = 99

		got := handler.serializeString(0, 1, false)
		if !strings.HasPrefix(got, "first|second\x1b[1;1H") {
			t.Fatalf("oversized extent was not clamped: %q", got)
		}
		if !strings.HasSuffix(got, "\x1b[31m") {
			t.Fatalf("active style was not restored: %q", got)
		}
	})
}
