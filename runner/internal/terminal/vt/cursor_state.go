package vt

type cursorState struct {
	x              int
	y              int
	fg             Color
	bg             Color
	attrs          CellAttrs
	underlineStyle UnderlineStyle
	underlineColor Color
}

func (vt *VirtualTerminal) activeBufferIndex() int {
	if vt.useAltScreen {
		return 1
	}
	return 0
}

func (vt *VirtualTerminal) currentCursorState() cursorState {
	return cursorState{
		x:              vt.cursorX,
		y:              vt.cursorY,
		fg:             vt.currentFg,
		bg:             vt.currentBg,
		attrs:          vt.currentAttrs,
		underlineStyle: vt.currentUnderlineStyle,
		underlineColor: vt.currentUnderlineColor,
	}
}

func (vt *VirtualTerminal) savedMainCursorState() cursorState {
	return cursorState{
		x:              vt.altCursorX,
		y:              vt.altCursorY,
		fg:             vt.savedMainFg,
		bg:             vt.savedMainBg,
		attrs:          vt.savedMainAttrs,
		underlineStyle: vt.savedMainUlStyle,
		underlineColor: vt.savedMainUlColor,
	}
}

func (vt *VirtualTerminal) saveCursor() {
	buffer := vt.activeBufferIndex()
	state := vt.currentCursorState()
	if buffer == 0 {
		// xterm stores the normal-buffer register in absolute buffer
		// coordinates so scrolling moves the saved position toward the top of
		// the viewport instead of leaving it pinned to a screen row.
		state.y += len(vt.historyStyled)
	}
	vt.savedCursor[buffer] = state
}

func (vt *VirtualTerminal) restoreCursor() {
	buffer := vt.activeBufferIndex()
	state := vt.savedCursor[buffer]
	if buffer == 0 {
		state.y = max(state.y-len(vt.historyStyled), 0)
	}
	vt.cursorX = state.x
	vt.cursorY = state.y
	vt.currentFg = state.fg
	vt.currentBg = state.bg
	vt.currentAttrs = state.attrs
	vt.currentUnderlineStyle = state.underlineStyle
	vt.currentUnderlineColor = state.underlineColor
	vt.clampCursor()
}

func (vt *VirtualTerminal) savedCursorForReplay(buffer int) cursorState {
	state := vt.savedCursor[buffer]
	if buffer == 0 {
		state.y = min(max(state.y-len(vt.historyStyled), 0), vt.rows-1)
	}
	return state
}

func (vt *VirtualTerminal) resetCursorRegisters() {
	vt.savedCursor = [2]cursorState{}
}

func (state cursorState) styleCell() Cell {
	return NewFullStyledCell(
		' ', state.fg, state.bg, state.attrs, 1,
		state.underlineStyle, state.underlineColor,
	)
}
