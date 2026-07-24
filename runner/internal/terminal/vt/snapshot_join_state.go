package vt

import (
	"fmt"
	"strings"
)

func (vt *VirtualTerminal) precedingJoinReplay() string {
	if !vt.canJoinGrapheme || vt.cursorY < 0 || vt.cursorY >= len(vt.cells) || vt.cursorX <= 0 {
		return ""
	}

	col := min(vt.cursorX-1, vt.cols-1)
	if col < 0 || col >= len(vt.cells[vt.cursorY]) {
		return ""
	}
	if vt.cells[vt.cursorY][col].IsPlaceholder() {
		col--
	}
	if col < 0 {
		return ""
	}
	cell := vt.cells[vt.cursorY][col]
	if !cell.HasContent || cell.Char == 0 || cell.Width == 0 || col+int(cell.Width) != vt.cursorX {
		return ""
	}
	if !cell.StyleEquals(vt.currentCursorState().styleCell()) {
		return ""
	}

	var replay strings.Builder
	fmt.Fprintf(&replay, "\x1b[%d;%dH", vt.cursorY+1, col+1)
	vt.appendAbsoluteStyle(&replay, cell)
	replay.WriteString(cell.Text())
	return replay.String()
}
