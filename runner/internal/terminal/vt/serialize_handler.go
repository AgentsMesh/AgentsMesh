package vt

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// StringSerializeHandler implements xterm.js-compatible terminal serialization.
type StringSerializeHandler struct {
	vt *VirtualTerminal

	// Row tracking
	allRows          []string
	allRowSeparators []string
	currentRow       strings.Builder
	rowIndex         int

	// Null cell optimization
	nullCellCount   int
	pendingNullCell Cell

	// Current cursor style
	cursorStyle    Cell
	cursorStyleRow int
	cursorStyleCol int

	// Background cell for BCE
	backgroundCell Cell

	// Position tracking
	firstRow             int
	lastCursorRow        int
	lastCursorCol        int
	lastContentCursorRow int
	lastContentCursorCol int
	previousContent      string
}

// newStringSerializeHandler creates a new handler
func newStringSerializeHandler(vt *VirtualTerminal) *StringSerializeHandler {
	return &StringSerializeHandler{
		vt:             vt,
		cursorStyle:    NewCell(' '),
		backgroundCell: NewCell(' '),
	}
}

// serialize serializes the terminal content from startRow to endRow (inclusive)
func (h *StringSerializeHandler) serialize(startRow, endRow int, excludeFinalCursorPosition bool) string {
	rowCount := endRow - startRow + 1
	h.allRows = make([]string, rowCount)
	h.allRowSeparators = make([]string, rowCount)
	h.firstRow = startRow
	h.lastContentCursorRow = startRow
	h.lastCursorRow = startRow
	h.lastCursorCol = 0
	h.lastContentCursorCol = 0
	h.rowIndex = 0

	var prevCell = NewCell(' ')
	for row := startRow; row <= endRow; row++ {
		h.currentRow.Reset()
		h.nullCellCount = 0

		cells := h.vt.getCellsRowNoLock(row)
		if cells != nil {
			for col := 0; col < len(cells); col++ {
				cell := cells[col]
				h.nextCell(cell, prevCell, row, col)
				prevCell = cell
			}
		}
		h.rowEnd(row, row == endRow)
	}

	return h.serializeString(startRow, endRow, excludeFinalCursorPosition)
}

// nextCell processes a single cell
func (h *StringSerializeHandler) nextCell(cell, oldCell Cell, row, col int) {
	if cell.IsPlaceholder() {
		return
	}

	standaloneZeroWidth := cell.Width == 0
	isEmptyCell := !cell.HasContent
	sgrSeq := h.diffStyle(cell, h.cursorStyle)

	styleChanged := standaloneZeroWidth
	if isEmptyCell {
		styleChanged = styleChanged || !cell.Bg.Equals(h.cursorStyle.Bg)
	} else {
		styleChanged = styleChanged || len(sgrSeq) > 0
	}

	if styleChanged {
		if h.nullCellCount > 0 {
			if !h.cursorStyle.Bg.Equals(h.backgroundCell.Bg) {
				fmt.Fprintf(&h.currentRow, "\x1b[%dX", h.nullCellCount)
			}
			fmt.Fprintf(&h.currentRow, "\x1b[%dC", h.nullCellCount)
			h.nullCellCount = 0
		}

		h.lastContentCursorRow = row
		h.lastContentCursorCol = col
		h.lastCursorRow = row
		h.lastCursorCol = col

		if standaloneZeroWidth {
			h.vt.appendAbsoluteStyle(&h.currentRow, cell)
		} else if len(sgrSeq) > 0 {
			h.currentRow.WriteString("\x1b[")
			h.currentRow.WriteString(strings.Join(sgrSeq, ";"))
			h.currentRow.WriteString("m")
		}

		h.cursorStyle = cell
		h.cursorStyleRow = row
		h.cursorStyleCol = col
	}

	if isEmptyCell {
		h.previousContent = ""
		width := cell.GetWidth()
		if width == 0 {
			width = 1
		}
		h.pendingNullCell = cell
		h.nullCellCount += int(width)
	} else {
		if h.nullCellCount > 0 {
			if h.cursorStyle.Bg.Equals(h.backgroundCell.Bg) {
				fmt.Fprintf(&h.currentRow, "\x1b[%dC", h.nullCellCount)
			} else {
				fmt.Fprintf(&h.currentRow, "\x1b[%dX", h.nullCellCount)
				fmt.Fprintf(&h.currentRow, "\x1b[%dC", h.nullCellCount)
			}
			h.nullCellCount = 0
			h.previousContent = ""
		}
		h.preserveGraphemeCellBoundary(cell)

		h.currentRow.WriteString(cell.Text())
		h.previousContent = cell.Text()

		width := cell.GetWidth()
		if width == 0 {
			width = 1
		}
		h.lastContentCursorRow = row
		h.lastContentCursorCol = col + int(width)
		h.lastCursorRow = row
		h.lastCursorCol = col + int(width)
	}
}

// preserveGraphemeCellBoundary prevents snapshot replay from joining two cells
// that were separated by a control sequence in the original PTY stream. SGR is
// display-neutral here because the exact cell style is immediately restored.
func (h *StringSerializeHandler) preserveGraphemeCellBoundary(cell Cell) {
	if h.previousContent == "" {
		return
	}
	text := cell.Text()
	first, _ := utf8.DecodeRuneInString(text)
	if first == utf8.RuneError || !graphemeJoins(h.previousContent, first) {
		return
	}
	h.vt.appendAbsoluteStyle(&h.currentRow, cell)
	h.cursorStyle = cell
}

// materializeTrailingNullCellsForWrap advances through the physical row with
// printable blanks. Cursor movement alone cannot create xterm's delayed-wrap
// state, so a snapshot of a non-reflowed cursor line must write the final blank
// before replaying its wrapped continuation.
func (h *StringSerializeHandler) materializeTrailingNullCellsForWrap() {
	if h.nullCellCount == 0 {
		return
	}
	h.vt.appendAbsoluteStyle(&h.currentRow, h.pendingNullCell)
	h.cursorStyle = h.pendingNullCell
	h.currentRow.WriteString(strings.Repeat(" ", h.nullCellCount))
	// These null cells become printable blanks in the replay stream. Track the
	// final blank as emitted content so the next physical row cannot absorb a
	// leading Extend/ZWJ code point that was a separate source cell.
	h.previousContent = " "
	h.nullCellCount = 0
}
