package vt

import (
	"strings"
	"testing"
)

func TestColorAndCursorClampEdgeCases(t *testing.T) {
	unknown := Color{colorType: 99}
	if unknown.Equals(unknown) {
		t.Fatal("unknown color type compared equal")
	}

	terminal := NewVirtualTerminal(3, 2, 0)
	terminal.cursorX, terminal.cursorY = -2, -3
	terminal.clampCursor()
	if terminal.cursorX != 0 || terminal.cursorY != 0 {
		t.Fatalf("negative cursor clamped to (%d,%d)", terminal.cursorX, terminal.cursorY)
	}
	terminal.cursorX, terminal.cursorY = 99, 99
	terminal.clampCursor()
	if terminal.cursorX != 2 || terminal.cursorY != 1 {
		t.Fatalf("oversized cursor clamped to (%d,%d)", terminal.cursorX, terminal.cursorY)
	}
}

func TestEraseDisplayAndLineModes(t *testing.T) {
	for _, mode := range []int{0, 1, 2, 3} {
		terminal := NewVirtualTerminal(4, 3, 10)
		terminal.Feed([]byte("abcdefghijkl"))
		terminal.cursorX, terminal.cursorY = 1, 1
		terminal.eraseInDisplay(mode)
	}
	terminal := NewVirtualTerminal(4, 2, 0)
	terminal.Feed([]byte("abcd"))
	terminal.cursorX = 2
	terminal.eraseInLine(1)
}

func TestGraphemeWidthAndJoinEdgeCases(t *testing.T) {
	if width := standaloneGraphemeWidth('\ufe0f'); width != 2 {
		t.Fatalf("VS16 width = %d", width)
	}
	cell := NewCell('A')
	if width := graphemeClusterWidth(cell, '界'); width != 2 {
		t.Fatalf("widened cluster width = %d", width)
	}
	if graphemeJoins("", '\u0301') {
		t.Fatal("empty grapheme accepted a combining mark")
	}
}

func TestParserDiscardDefaultState(t *testing.T) {
	terminal := NewVirtualTerminal(2, 1, 0)
	terminal.escState = stateNormal
	terminal.escBuffer = []byte("partial")
	terminal.enterParserDiscard()
	if terminal.escState != stateNormal || terminal.escBuffer != nil {
		t.Fatal("default discard state did not reset parser")
	}
}

func TestAlternateScreenRepeatedEntry(t *testing.T) {
	terminal := NewVirtualTerminal(4, 2, 0)
	terminal.enterAltScreen(1049)
	screen := terminal.screen
	terminal.enterAltScreen(1049)
	if !terminal.useAltScreen || &terminal.screen[0][0] != &screen[0][0] {
		t.Fatal("repeated alternate-screen entry replaced active buffer")
	}
}

func TestCursorReplayTrailingCellEdges(t *testing.T) {
	terminal := NewVirtualTerminal(2, 1, 0)
	state := terminal.currentCursorState()
	state.x = terminal.cols
	var replay strings.Builder
	terminal.appendCursorState(&replay, state, terminal.cells)
	if !strings.Contains(replay.String(), " ") {
		t.Fatalf("empty delayed-wrap cell was not materialized: %q", replay.String())
	}

	if col, cell := trailingCell(nil, 0, 2); col != 0 || cell.HasContent {
		t.Fatalf("invalid trailing cell = (%d, %+v)", col, cell)
	}
	wide := NewFullStyledCell('界', DefaultColor(), DefaultColor(), AttrNone, 2, UnderlineNone, DefaultColor())
	placeholder := NewFullStyledCell(0, DefaultColor(), DefaultColor(), AttrNone, 0, UnderlineNone, DefaultColor())
	if col, cell := trailingCell([][]Cell{{wide, placeholder}}, 0, 2); col != 0 || cell.Char != '界' {
		t.Fatalf("wide trailing cell = (%d, %+v)", col, cell)
	}
}

func TestPrecedingJoinReplayRejectionBranches(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*VirtualTerminal)
	}{
		{name: "row shorter than cursor", setup: func(v *VirtualTerminal) {
			v.cells[0] = nil
			v.cursorX = 1
		}},
		{name: "placeholder has no owner", setup: func(v *VirtualTerminal) {
			v.cells[0][0] = NewFullStyledCell(0, DefaultColor(), DefaultColor(), AttrNone, 0, UnderlineNone, DefaultColor())
			v.cursorX = 1
		}},
		{name: "empty preceding cell", setup: func(v *VirtualTerminal) { v.cursorX = 1 }},
		{name: "style mismatch", setup: func(v *VirtualTerminal) {
			v.cells[0][0] = NewStyledCell('A', PaletteColor(1), DefaultColor(), AttrNone)
			v.cells[0][0].HasContent = true
			v.cursorX = 1
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			terminal := NewVirtualTerminal(2, 1, 0)
			terminal.canJoinGrapheme = true
			tt.setup(terminal)
			if replay := terminal.precedingJoinReplay(); replay != "" {
				t.Fatalf("rejected join replay = %q", replay)
			}
		})
	}
}

func TestScreenOnlySerializationBoundsAndBackground(t *testing.T) {
	terminal := NewVirtualTerminal(3, 3, 0)
	handler := newStringSerializeHandler(terminal)
	handler.allRows = make([]string, 1)
	handler.allRowSeparators = make([]string, 1)
	handler.cursorStyle.Bg = PaletteColor(1)
	handler.nullCellCount = 2
	handler.rowEndScreenOnly(0, true, false)
	if handler.allRows[0] != "\x1b[2X" {
		t.Fatalf("screen-only trailing erase = %q", handler.allRows[0])
	}

	negative := newStringSerializeHandler(terminal)
	negative.allRows = []string{"ignored"}
	negative.allRowSeparators = []string{""}
	negative.firstRow = 2
	negative.lastContentCursorRow = 0
	if got := negative.serializeStringScreenOnly(screenSerializeState{}, 2, 2, true); got != "" {
		t.Fatalf("negative screen extent = %q", got)
	}

	oversized := newStringSerializeHandler(terminal)
	oversized.allRows = []string{"a"}
	oversized.allRowSeparators = []string{""}
	oversized.lastContentCursorRow = 99
	if got := oversized.serializeStringScreenOnly(screenSerializeState{}, 0, 0, true); got != "a" {
		t.Fatalf("oversized screen extent = %q", got)
	}
}

func TestFullBufferEmptyAndInvalidRows(t *testing.T) {
	terminal := NewVirtualTerminal(2, 1, 0)
	handler := newStringSerializeHandler(terminal)
	if got := handler.serializeFullBufferStateNoLock(screenSerializeState{}); got != "" {
		t.Fatalf("empty full buffer = %q", got)
	}
	if cells, wrapped := fullBufferRow(screenSerializeState{}, -1); cells != nil || wrapped {
		t.Fatalf("negative full buffer row = (%v, %v)", cells, wrapped)
	}
}

func TestGraphemeAppendAndWrappedMoveEdges(t *testing.T) {
	invalid := NewVirtualTerminal(2, 1, 0)
	invalid.cursorY = -1
	if invalid.appendToPrecedingGrapheme('\u0301') {
		t.Fatal("invalid cursor accepted grapheme")
	}
	invalid.cursorY = 0
	invalid.cells[0] = nil
	invalid.cursorX = 1
	if invalid.appendToPrecedingGrapheme('\u0301') {
		t.Fatal("missing row cell accepted grapheme")
	}

	noWrap := NewVirtualTerminal(1, 1, 0)
	noWrap.Feed([]byte("A"))
	noWrap.privateModes[7] = false
	if !noWrap.appendToPrecedingGrapheme('\ufe0f') || noWrap.cursorX != 0 {
		t.Fatalf("right-margin widening was not consumed: cursor=%d", noWrap.cursorX)
	}

	wrapped := NewVirtualTerminal(2, 1, 2)
	cell := NewFullStyledCell('界', DefaultColor(), DefaultColor(), AttrNone, 2, UnderlineNone, DefaultColor())
	col, moved := wrapped.moveGraphemeToWrappedLine(0, cell)
	if col != 0 || moved.Char != '界' || wrapped.cursorY != 0 || !wrapped.isWrapped[0] || !wrapped.cells[0][1].IsPlaceholder() {
		t.Fatalf("wrapped grapheme state invalid: col=%d cell=%+v", col, *moved)
	}
}

func TestLineProjectionAndBufferAccessBounds(t *testing.T) {
	terminal := NewVirtualTerminal(2, 1, 0)
	if line := terminal.screenLineLocked(-1); line != "" {
		t.Fatalf("negative screen row = %q", line)
	}
	if line := terminal.screenLineLocked(0); line != "" {
		t.Fatalf("empty screen row = %q", line)
	}
	if terminal.GetCellsRow(-1) != nil || terminal.IsLineWrapped(-1) {
		t.Fatal("negative buffer index returned data")
	}
	terminal.GetCurrentStyle()
	if terminal.GetHistoryStyledRow(-1) != nil || terminal.IsHistoryLineWrapped(-1) {
		t.Fatal("negative history index returned data")
	}
}

func TestEscapeRestartDiscardsInterruptedCSI(t *testing.T) {
	terminal := NewVirtualTerminal(4, 2, 0)
	terminal.escState = stateCSI
	terminal.escBuffer = []byte("31")
	terminal.escParams = []int{31}
	terminal.escPrivate = '?'
	terminal.escRawSeq = []byte("\x1b[31")

	terminal.processByte(0x1b)

	if terminal.escState != stateEscape || terminal.escBuffer != nil ||
		terminal.escParams != nil || terminal.escPrivate != 0 || terminal.escRawSeq != nil {
		t.Fatalf("interrupted CSI did not restart the parser: state=%v buffer=%q", terminal.escState, terminal.escBuffer)
	}
}

func TestCompleteCellRangeOwnsBothHalvesOfWideCells(t *testing.T) {
	wide := NewFullStyledCell('界', DefaultColor(), DefaultColor(), AttrNone, 2, UnderlineNone, DefaultColor())
	placeholder := NewFullStyledCell(0, DefaultColor(), DefaultColor(), AttrNone, 0, UnderlineNone, DefaultColor())
	cells := []Cell{wide, placeholder, wide, placeholder}

	start, end := completeCellRange(cells, 1, 3)
	if start != 0 || end != 4 {
		t.Fatalf("wide-cell range = [%d,%d), want [0,4)", start, end)
	}
}

func TestScreenProjectionMaterializesZeroRuneAsBlank(t *testing.T) {
	terminal := NewVirtualTerminal(2, 1, 0)
	terminal.screen[0][0] = 0
	terminal.cells[0][0] = NewCell(' ')
	if line := terminal.screenLineLocked(0); line != "" {
		t.Fatalf("zero-rune compatibility cell projected as %q", line)
	}
}

func TestHistoryRowEndEmitsBackgroundErase(t *testing.T) {
	terminal := NewVirtualTerminal(3, 2, 0)
	handler := newStringSerializeHandler(terminal)
	handler.allRows = make([]string, 1)
	handler.allRowSeparators = make([]string, 1)
	handler.cursorStyle.Bg = PaletteColor(1)
	handler.nullCellCount = 2

	handler.rowEndWithWrap(0, true, false, false)

	if got := handler.allRows[0]; got != "\x1b[2X" {
		t.Fatalf("history trailing colored null cells = %q, want ECH", got)
	}
}

func TestInitAndResizeAlternateScreenFallbackDimensions(t *testing.T) {
	terminal := NewVirtualTerminal(2, 1, 0)
	terminal.useAltScreen = true
	terminal.initScreen()
	if len(terminal.altCells) != 1 || len(terminal.altCells[0]) != 2 ||
		&terminal.altCells[0][0] != &terminal.cells[0][0] {
		t.Fatal("initScreen did not retain alternate-plane aliases")
	}

	terminal.Resize(0, -1)
	if terminal.cols != 80 || terminal.rows != 24 {
		t.Fatalf("fallback resize = %dx%d, want 80x24", terminal.cols, terminal.rows)
	}
	if &terminal.altCells[0][0] != &terminal.cells[0][0] {
		t.Fatal("resize broke alternate-plane aliases")
	}
}

func TestReflowWideCellIntoSingleColumnDoesNotSplitGlyph(t *testing.T) {
	wide := NewFullStyledCell('界', DefaultColor(), DefaultColor(), AttrNone, 2, UnderlineNone, DefaultColor())
	placeholder := NewFullStyledCell(0, DefaultColor(), DefaultColor(), AttrNone, 0, UnderlineNone, DefaultColor())
	rows, starts, ends := reflowLogicalGroup([]terminalBufferLine{{
		cells: []Cell{wide, placeholder},
	}}, 2, 1)

	if len(rows) != 1 || len(rows[0].cells) != 1 || !rows[0].cells[0].IsEmpty() {
		t.Fatalf("single-column wide glyph was split: %+v", rows)
	}
	if len(starts) != 2 || starts[1] != 2 || len(ends) != 1 || ends[0] != 2 {
		t.Fatalf("reflow source mapping lost the consumed wide cell: starts=%v ends=%v", starts, ends)
	}
}

func TestSanitizeCellsRepairsOrphanWideHalves(t *testing.T) {
	wide := NewFullStyledCell('界', DefaultColor(), DefaultColor(), AttrNone, 2, UnderlineNone, DefaultColor())
	placeholder := NewFullStyledCell(0, DefaultColor(), DefaultColor(), AttrNone, 0, UnderlineNone, DefaultColor())
	cells := []Cell{placeholder, wide}
	sanitizeCells(cells)
	if !cells[0].IsEmpty() || !cells[1].IsEmpty() || cells[0].IsPlaceholder() || cells[1].Width == 2 {
		t.Fatalf("orphan wide-cell halves survived sanitization: %+v", cells)
	}
}
