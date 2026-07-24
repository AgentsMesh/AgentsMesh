package vt

import (
	"fmt"
	"strings"
)

func (h *StringSerializeHandler) rowEnd(row int, isLastRow bool) {
	nextLineWrapped := !isLastRow && h.vt.isLineWrappedNoLock(row+1)
	if nextLineWrapped {
		h.materializeTrailingNullCellsForWrap()
	}
	if h.nullCellCount > 0 && !h.cursorStyle.Bg.Equals(h.backgroundCell.Bg) {
		fmt.Fprintf(&h.currentRow, "\x1b[%dX", h.nullCellCount)
	}

	rowSeparator := ""
	if !isLastRow {
		if !nextLineWrapped {
			rowSeparator = "\r\n"
			h.previousContent = ""
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

func (h *StringSerializeHandler) serializeString(startRow, endRow int, excludeFinalCursorPosition bool) string {
	var content strings.Builder

	rowEnd := len(h.allRows)
	bufferLength := endRow - startRow + 1
	if bufferLength <= h.vt.rows {
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
		fmt.Fprintf(&content, "\x1b[%d;%dH", h.vt.cursorY+1, h.vt.cursorX+1)
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
