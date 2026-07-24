package vt

func (vt *VirtualTerminal) clampCursor() {
	if vt.cursorY < 0 {
		vt.cursorY = 0
	}
	if vt.cursorY >= vt.rows {
		vt.cursorY = vt.rows - 1
	}
	if vt.cursorX < 0 {
		vt.cursorX = 0
	}
	if vt.cursorX >= vt.cols {
		vt.cursorX = vt.cols - 1
	}
}

// eraseInDisplay handles ED command.
func (vt *VirtualTerminal) eraseInDisplay(n int) {
	switch n {
	case 0:
		vt.clearLine(vt.cursorY, vt.cursorX, vt.cols)
		for i := vt.cursorY + 1; i < vt.rows; i++ {
			vt.clearLine(i, 0, vt.cols)
		}
	case 1:
		for i := 0; i < vt.cursorY; i++ {
			vt.clearLine(i, 0, vt.cols)
		}
		vt.clearLine(vt.cursorY, 0, vt.cursorX+1)
	case 2:
		for i := 0; i < vt.rows; i++ {
			vt.clearLine(i, 0, vt.cols)
			vt.isWrapped[i] = false
		}
	case 3:
		// DECSED 3 clears saved lines in the active normal buffer without
		// erasing the viewport. Alternate buffers have no scrollback and must
		// not mutate the hidden normal buffer.
		if !vt.useAltScreen {
			vt.history = vt.history[:0]
			vt.historyStyled = vt.historyStyled[:0]
			vt.historyIsWrapped = vt.historyIsWrapped[:0]
		}
	}
}

// eraseInLine handles EL command.
func (vt *VirtualTerminal) eraseInLine(n int) {
	switch n {
	case 0:
		vt.clearLine(vt.cursorY, vt.cursorX, vt.cols)
	case 1:
		vt.clearLine(vt.cursorY, 0, vt.cursorX+1)
	case 2:
		vt.clearLine(vt.cursorY, 0, vt.cols)
	}
}
