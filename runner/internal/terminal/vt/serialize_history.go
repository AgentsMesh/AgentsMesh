package vt

import (
	"fmt"
	"strings"
)

// serializeWithHistory serializes both history and screen content with full style support
func (h *StringSerializeHandler) serializeWithHistory(startRow, endRow, historyLen int, excludeFinalCursorPosition bool) string {
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

		var cells []Cell
		var isWrapped bool

		if row < historyLen {
			cells = h.vt.historyStyled[row]
			isWrapped = h.vt.historyIsWrapped[row]
		} else {
			screenRow := row - historyLen
			if screenRow >= 0 && screenRow < h.vt.rows {
				cells = h.vt.cells[screenRow]
				isWrapped = h.vt.isWrapped[screenRow]
			}
		}

		if cells != nil {
			for col := 0; col < len(cells); col++ {
				cell := cells[col]
				h.nextCell(cell, prevCell, row, col)
				prevCell = cell
			}
		}
		h.rowEndWithWrap(row, row == endRow, isWrapped, row+1 < historyLen || (row+1 >= historyLen && row+1-historyLen < h.vt.rows))
	}

	return h.serializeStringForHistory(startRow, endRow, historyLen, excludeFinalCursorPosition)
}

// rowEndWithWrap handles end of row processing with explicit wrap info
func (h *StringSerializeHandler) rowEndWithWrap(row int, isLastRow bool, currentWrapped bool, hasNextRow bool) {
	if h.nullCellCount > 0 && !h.cursorStyle.Bg.Equals(h.backgroundCell.Bg) {
		fmt.Fprintf(&h.currentRow, "\x1b[%dX", h.nullCellCount)
	}

	rowSeparator := ""

	if !isLastRow && hasNextRow {
		historyLen := len(h.vt.historyStyled)
		var nextLineWrapped bool
		if row+1 < historyLen {
			nextLineWrapped = h.vt.historyIsWrapped[row+1]
		} else {
			screenRow := row + 1 - historyLen
			if screenRow >= 0 && screenRow < h.vt.rows {
				nextLineWrapped = h.vt.isWrapped[screenRow]
			}
		}

		if !nextLineWrapped {
			rowSeparator = "\r\n"
			h.lastCursorRow = row + 1
			h.lastCursorCol = 0
		} else {
			rowSeparator = ""
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

// serializeStringForHistory builds the final serialized string for history+screen
func (h *StringSerializeHandler) serializeStringForHistory(startRow, endRow, historyLen int, excludeFinalCursorPosition bool) string {
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
		cursorRow := h.vt.cursorY + 1
		cursorCol := h.vt.cursorX + 1
		fmt.Fprintf(&content, "\x1b[%d;%dH", cursorRow, cursorCol)
	}

	curFg, curBg, curAttrs, curUlStyle, curUlColor := h.vt.getCurrentStyleNoLock()
	curCell := NewFullStyledCell(' ', curFg, curBg, curAttrs, 1, curUlStyle, curUlColor)
	sgrSeq := h.diffStyle(curCell, h.cursorStyle)
	if len(sgrSeq) > 0 {
		content.WriteString("\x1b[")
		content.WriteString(strings.Join(sgrSeq, ";"))
		content.WriteString("m")
	}

	return content.String()
}
