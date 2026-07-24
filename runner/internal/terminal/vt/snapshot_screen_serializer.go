package vt

import (
	"fmt"
	"strings"
)

type screenSerializeState struct {
	historyCells   [][]Cell
	historyWrapped []bool
	cells          [][]Cell
	wrapped        []bool
	cursorX        int
	cursorY        int
	activeStyle    Cell
}

func (vt *VirtualTerminal) serializeFullNormalBuffer() string {
	state := screenSerializeState{
		historyCells:   vt.historyStyled,
		historyWrapped: vt.historyIsWrapped,
		cells:          vt.cells,
		wrapped:        vt.isWrapped,
		cursorX:        vt.cursorX,
		cursorY:        vt.cursorY,
		activeStyle:    NewFullStyledCell(' ', vt.currentFg, vt.currentBg, vt.currentAttrs, 1, vt.currentUnderlineStyle, vt.currentUnderlineColor),
	}
	handler := newStringSerializeHandler(vt)
	return handler.serializeFullBufferStateNoLock(state)
}

func (vt *VirtualTerminal) serializeSavedMainFullBuffer() string {
	state := screenSerializeState{
		historyCells:   vt.historyStyled,
		historyWrapped: vt.historyIsWrapped,
		cells:          vt.savedMainCells,
		wrapped:        vt.savedMainWrapped,
		cursorX:        vt.altCursorX,
		cursorY:        vt.altCursorY,
		activeStyle:    NewFullStyledCell(' ', vt.savedMainFg, vt.savedMainBg, vt.savedMainAttrs, 1, vt.savedMainUlStyle, vt.savedMainUlColor),
	}
	handler := newStringSerializeHandler(vt)
	return handler.serializeFullBufferStateNoLock(state)
}

func (vt *VirtualTerminal) serializeScreenOnly() string {
	handler := newStringSerializeHandler(vt)
	return handler.serializeScreenNoLock(0, vt.rows-1, false)
}

// Caller holds the VirtualTerminal lock.
func (h *StringSerializeHandler) serializeScreenNoLock(startRow, endRow int, excludeFinalCursorPosition bool) string {
	state := screenSerializeState{
		cells:       h.vt.cells,
		wrapped:     h.vt.isWrapped,
		cursorX:     h.vt.cursorX,
		cursorY:     h.vt.cursorY,
		activeStyle: NewFullStyledCell(' ', h.vt.currentFg, h.vt.currentBg, h.vt.currentAttrs, 1, h.vt.currentUnderlineStyle, h.vt.currentUnderlineColor),
	}
	return h.serializeScreenStateNoLock(state, startRow, endRow, excludeFinalCursorPosition)
}

func (h *StringSerializeHandler) serializeScreenStateNoLock(state screenSerializeState, startRow, endRow int, excludeFinalCursorPosition bool) string {
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
		if row < len(state.cells) {
			cells = state.cells[row]
		}
		for col := 0; col < len(cells); col++ {
			cell := cells[col]
			h.nextCell(cell, prevCell, row, col)
			prevCell = cell
		}

		isLastRow := row == endRow
		var nextLineWrapped bool
		if !isLastRow && row+1 < len(state.wrapped) {
			nextLineWrapped = state.wrapped[row+1]
		}
		h.rowEndScreenOnly(row, isLastRow, nextLineWrapped)
	}

	return h.serializeStringScreenOnly(state, startRow, endRow, excludeFinalCursorPosition)
}

func (h *StringSerializeHandler) serializeFullBufferStateNoLock(state screenSerializeState) string {
	historyLen := len(state.historyCells)
	totalRows := historyLen + len(state.cells)
	if totalRows == 0 {
		return ""
	}

	h.allRows = make([]string, totalRows)
	h.allRowSeparators = make([]string, totalRows)
	h.firstRow = 0
	h.lastContentCursorRow = 0
	h.lastCursorRow = 0
	h.lastCursorCol = 0
	h.lastContentCursorCol = 0
	h.rowIndex = 0

	var prevCell = NewCell(' ')
	for row := 0; row < totalRows; row++ {
		h.currentRow.Reset()
		h.nullCellCount = 0

		cells, _ := fullBufferRow(state, row)
		for col := 0; col < len(cells); col++ {
			cell := cells[col]
			h.nextCell(cell, prevCell, row, col)
			prevCell = cell
		}
		_, nextWrapped := fullBufferRow(state, row+1)
		h.rowEndScreenOnly(row, row == totalRows-1, nextWrapped)
	}

	var content strings.Builder
	for row := range h.allRows {
		content.WriteString(h.allRows[row])
		if row+1 < len(h.allRows) {
			content.WriteString(h.allRowSeparators[row])
		}
	}
	fmt.Fprintf(&content, "\x1b[%d;%dH", state.cursorY+1, state.cursorX+1)

	sgrSeq := h.diffStyle(state.activeStyle, h.cursorStyle)
	if len(sgrSeq) > 0 {
		content.WriteString("\x1b[")
		content.WriteString(strings.Join(sgrSeq, ";"))
		content.WriteString("m")
	}
	return content.String()
}

func fullBufferRow(state screenSerializeState, row int) ([]Cell, bool) {
	if row < 0 {
		return nil, false
	}
	if row < len(state.historyCells) {
		wrapped := row < len(state.historyWrapped) && state.historyWrapped[row]
		return state.historyCells[row], wrapped
	}
	row -= len(state.historyCells)
	if row >= len(state.cells) {
		return nil, false
	}
	wrapped := row < len(state.wrapped) && state.wrapped[row]
	return state.cells[row], wrapped
}
