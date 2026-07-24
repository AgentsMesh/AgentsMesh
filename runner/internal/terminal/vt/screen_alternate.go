package vt

func (vt *VirtualTerminal) enterAltScreen(mode int) {
	if mode == 1049 {
		vt.saveCursor()
	}
	if vt.useAltScreen {
		return
	}
	vt.savedMainScreen = make([][]rune, vt.rows)
	vt.savedMainCells = make([][]Cell, vt.rows)
	vt.savedMainWrapped = make([]bool, vt.rows)
	for row := range vt.screen {
		vt.savedMainScreen[row] = append([]rune(nil), vt.screen[row]...)
		vt.savedMainCells[row] = append([]Cell(nil), vt.cells[row]...)
	}
	copy(vt.savedMainWrapped, vt.isWrapped)
	vt.savedMainFg = vt.currentFg
	vt.savedMainBg = vt.currentBg
	vt.savedMainAttrs = vt.currentAttrs
	vt.savedMainUlStyle = vt.currentUnderlineStyle
	vt.savedMainUlColor = vt.currentUnderlineColor

	vt.altScreen, vt.altCells = newScreenBuffers(vt.cols, vt.rows)
	vt.altIsWrapped = make([]bool, vt.rows)
	vt.altCursorX = vt.cursorX
	vt.altCursorY = vt.cursorY
	vt.altScreenMode = mode
	vt.screen = vt.altScreen
	vt.cells = vt.altCells
	vt.isWrapped = vt.altIsWrapped
	vt.useAltScreen = true
}

func (vt *VirtualTerminal) exitAltScreen(mode int) {
	if !vt.useAltScreen {
		if mode == 1049 {
			vt.restoreCursor()
		}
		return
	}
	if vt.savedMainScreen != nil {
		vt.screen = vt.savedMainScreen
		vt.cells = vt.savedMainCells
		vt.isWrapped = vt.savedMainWrapped
		vt.savedMainScreen = nil
		vt.savedMainCells = nil
		vt.savedMainWrapped = nil
	}
	vt.useAltScreen = false
	vt.altScreenMode = 0
	if mode == 1049 {
		vt.restoreCursor()
	}
}
