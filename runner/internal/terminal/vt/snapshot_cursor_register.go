package vt

import (
	"fmt"
	"strings"
)

func (vt *VirtualTerminal) cursorRegisterReplay(
	saved cursorState,
	current cursorState,
	cells [][]Cell,
) string {
	var replay strings.Builder
	vt.appendCursorState(&replay, saved, cells)
	replay.WriteString("\x1b7")
	vt.appendCursorState(&replay, current, cells)
	return replay.String()
}

func (vt *VirtualTerminal) appendCursorState(
	replay *strings.Builder,
	state cursorState,
	cells [][]Cell,
) {
	row := min(max(state.y, 0), vt.rows-1)
	col := min(max(state.x, 0), vt.cols-1)
	if state.x >= vt.cols {
		// CUP clamps to the last column. Rewriting the same trailing cell is the
		// only side-effect-free way to restore xterm's delayed-wrap position.
		cellCol, cell := trailingCell(cells, row, vt.cols)
		fmt.Fprintf(replay, "\x1b[%d;%dH", row+1, cellCol+1)
		vt.appendAbsoluteStyle(replay, cell)
		if !cell.HasContent {
			replay.WriteByte(' ')
		} else {
			replay.WriteString(cell.Text())
		}
	} else {
		fmt.Fprintf(replay, "\x1b[%d;%dH", row+1, col+1)
	}
	vt.appendAbsoluteStyle(replay, state.styleCell())
}

func (vt *VirtualTerminal) appendAbsoluteStyle(replay *strings.Builder, target Cell) {
	replay.WriteString("\x1b[0m")
	params := newStringSerializeHandler(vt).diffStyle(target, NewCell(' '))
	if len(params) == 0 {
		return
	}
	replay.WriteString("\x1b[")
	replay.WriteString(strings.Join(params, ";"))
	replay.WriteByte('m')
}

func trailingCell(cells [][]Cell, row, cols int) (int, Cell) {
	if row < 0 || row >= len(cells) || cols <= 0 || len(cells[row]) == 0 {
		return 0, NewCell(' ')
	}
	col := min(cols-1, len(cells[row])-1)
	cell := cells[row][col]
	if cell.IsPlaceholder() && col > 0 {
		col--
		cell = cells[row][col]
	}
	return col, cell
}
