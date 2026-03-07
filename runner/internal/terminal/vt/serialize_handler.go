package vt

import (
	"fmt"
	"strings"
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
	nullCellCount int

	// Current cursor style
	cursorStyle    Cell
	cursorStyleRow int
	cursorStyleCol int

	// Background cell for BCE
	backgroundCell Cell

	// Position tracking
	firstRow              int
	lastCursorRow         int
	lastCursorCol         int
	lastContentCursorRow  int
	lastContentCursorCol  int
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

		cells := h.vt.GetCellsRow(row)
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
	if cell.GetWidth() == 0 {
		return
	}

	isEmptyCell := cell.Char == ' ' || cell.Char == 0
	sgrSeq := h.diffStyle(cell, h.cursorStyle)

	styleChanged := false
	if isEmptyCell {
		styleChanged = !cell.Bg.Equals(h.cursorStyle.Bg)
	} else {
		styleChanged = len(sgrSeq) > 0
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

		if len(sgrSeq) > 0 {
			h.currentRow.WriteString("\x1b[")
			h.currentRow.WriteString(strings.Join(sgrSeq, ";"))
			h.currentRow.WriteString("m")
		}

		h.cursorStyle = cell
		h.cursorStyleRow = row
		h.cursorStyleCol = col
	}

	if isEmptyCell {
		width := cell.GetWidth()
		if width == 0 {
			width = 1
		}
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
		}

		if cell.Char != 0 {
			h.currentRow.WriteRune(cell.Char)
		} else {
			h.currentRow.WriteRune(' ')
		}

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

// rowEnd handles end of row processing
func (h *StringSerializeHandler) rowEnd(row int, isLastRow bool) {
	if h.nullCellCount > 0 && !h.cursorStyle.Bg.Equals(h.backgroundCell.Bg) {
		fmt.Fprintf(&h.currentRow, "\x1b[%dX", h.nullCellCount)
	}

	rowSeparator := ""
	if !isLastRow {
		if !h.vt.IsLineWrapped(row + 1) {
			rowSeparator = "\r\n"
			h.lastCursorRow = row + 1
			h.lastCursorCol = 0
		} else {
			h.lastContentCursorRow = row + 1
			h.lastContentCursorCol = 0
			h.lastCursorRow = row + 1
			h.lastCursorCol = 0
		}
	}

	h.allRows[h.rowIndex] = h.currentRow.String()
	h.allRowSeparators[h.rowIndex] = rowSeparator
	h.rowIndex++
	h.currentRow.Reset()
	h.nullCellCount = 0
}

// serializeString builds the final serialized string
func (h *StringSerializeHandler) serializeString(startRow, endRow int, excludeFinalCursorPosition bool) string {
	var content strings.Builder

	rowEnd := len(h.allRows)
	bufferLength := endRow - startRow + 1
	if bufferLength <= h.vt.Rows() {
		rowEnd = h.lastContentCursorRow + 1 - h.firstRow
		if rowEnd < 0 {
			rowEnd = 0
		}
		if rowEnd > len(h.allRows) {
			rowEnd = len(h.allRows)
		}
		h.lastCursorCol = h.lastContentCursorCol
		h.lastCursorRow = h.lastContentCursorRow
	}

	for i := 0; i < rowEnd; i++ {
		content.WriteString(h.allRows[i])
		if i+1 < rowEnd {
			content.WriteString(h.allRowSeparators[i])
		}
	}

	if !excludeFinalCursorPosition {
		cursorRow, cursorCol := h.vt.CursorPosition()
		fmt.Fprintf(&content, "\x1b[%d;%dH", cursorRow+1, cursorCol+1)
	}

	curFg, curBg, curAttrs, curUlStyle, curUlColor := h.vt.GetCurrentStyle()
	curCell := NewFullStyledCell(' ', curFg, curBg, curAttrs, 1, curUlStyle, curUlColor)
	sgrSeq := h.diffStyle(curCell, h.cursorStyle)
	if len(sgrSeq) > 0 {
		content.WriteString("\x1b[")
		content.WriteString(strings.Join(sgrSeq, ";"))
		content.WriteString("m")
	}

	return content.String()
}

// diffStyle generates SGR parameters for style transition
func (h *StringSerializeHandler) diffStyle(cell, oldCell Cell) []string {
	var sgrSeq []string

	fgChanged := !cell.Fg.Equals(oldCell.Fg)
	bgChanged := !cell.Bg.Equals(oldCell.Bg)
	flagsChanged := !equalFlags(cell, oldCell)

	if !fgChanged && !bgChanged && !flagsChanged {
		return nil
	}

	if cell.IsAttributeDefault() {
		if !oldCell.IsAttributeDefault() {
			sgrSeq = append(sgrSeq, "0")
		}
	} else {
		if fgChanged {
			sgrSeq = append(sgrSeq, buildFgColorSGR(cell.Fg)...)
		}
		if bgChanged {
			sgrSeq = append(sgrSeq, buildBgColorSGR(cell.Bg)...)
		}
		if flagsChanged {
			sgrSeq = append(sgrSeq, buildFlagsSGR(cell, oldCell)...)
		}
	}

	return sgrSeq
}
