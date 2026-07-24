package vt

// clearLine clears part of a line
func (vt *VirtualTerminal) clearLine(row, startCol, endCol int) {
	if row < 0 || row >= vt.rows {
		return
	}
	startCol, endCol = completeCellRange(vt.cells[row], startCol, endCol)
	for i := startCol; i < endCol && i < vt.cols; i++ {
		if i >= 0 {
			vt.screen[row][i] = ' '
			vt.cells[row][i] = NewCell(' ')
		}
	}
}

// insertLines inserts n blank lines at cursor position
func (vt *VirtualTerminal) insertLines(n int) {
	for i := 0; i < n; i++ {
		// Shift lines down
		for j := vt.rows - 1; j > vt.cursorY; j-- {
			copy(vt.screen[j], vt.screen[j-1])
			copy(vt.cells[j], vt.cells[j-1])
			vt.isWrapped[j] = vt.isWrapped[j-1]
		}
		// Clear current line
		for j := range vt.screen[vt.cursorY] {
			vt.screen[vt.cursorY][j] = ' '
			vt.cells[vt.cursorY][j] = NewCell(' ')
		}
		vt.isWrapped[vt.cursorY] = false
	}
}

// deleteLines deletes n lines at cursor position
func (vt *VirtualTerminal) deleteLines(n int) {
	for i := 0; i < n; i++ {
		// Shift lines up
		for j := vt.cursorY; j < vt.rows-1; j++ {
			copy(vt.screen[j], vt.screen[j+1])
			copy(vt.cells[j], vt.cells[j+1])
			vt.isWrapped[j] = vt.isWrapped[j+1]
		}
		// Clear bottom line
		for j := range vt.screen[vt.rows-1] {
			vt.screen[vt.rows-1][j] = ' '
			vt.cells[vt.rows-1][j] = NewCell(' ')
		}
		vt.isWrapped[vt.rows-1] = false
	}
}

// deleteChars deletes n characters at cursor position
func (vt *VirtualTerminal) deleteChars(n int) {
	row := vt.screen[vt.cursorY]
	cellRow := vt.cells[vt.cursorY]
	for i := vt.cursorX; i < vt.cols-n; i++ {
		row[i] = row[i+n]
		cellRow[i] = cellRow[i+n]
	}
	for i := vt.cols - n; i < vt.cols; i++ {
		if i >= 0 {
			row[i] = ' '
			cellRow[i] = NewCell(' ')
		}
	}
	sanitizeResizedRow(row, cellRow)
}

// insertChars inserts n blank characters at cursor position
func (vt *VirtualTerminal) insertChars(n int) {
	row := vt.screen[vt.cursorY]
	cellRow := vt.cells[vt.cursorY]
	for i := vt.cols - 1; i >= vt.cursorX+n; i-- {
		row[i] = row[i-n]
		cellRow[i] = cellRow[i-n]
	}
	for i := 0; i < n && vt.cursorX+i < vt.cols; i++ {
		row[vt.cursorX+i] = ' '
		cellRow[vt.cursorX+i] = NewCell(' ')
	}
	sanitizeResizedRow(row, cellRow)
}

func completeCellRange(cells []Cell, startCol, endCol int) (int, int) {
	startCol = min(max(startCol, 0), len(cells))
	endCol = min(max(endCol, startCol), len(cells))
	if startCol < len(cells) && cells[startCol].IsPlaceholder() && startCol > 0 {
		startCol--
	}
	if endCol > startCol && cells[endCol-1].Width == 2 && endCol < len(cells) {
		endCol++
	}
	return startCol, endCol
}

// scrollDown scrolls the screen down (reverse scroll)
func (vt *VirtualTerminal) scrollDown() {
	// Shift all lines down
	for i := vt.rows - 1; i > 0; i-- {
		vt.screen[i] = vt.screen[i-1]
		vt.cells[i] = vt.cells[i-1]
		vt.isWrapped[i] = vt.isWrapped[i-1]
	}
	// Clear top line
	vt.screen[0] = make([]rune, vt.cols)
	vt.cells[0] = make([]Cell, vt.cols)
	vt.isWrapped[0] = false
	for j := range vt.screen[0] {
		vt.screen[0][j] = ' '
		vt.cells[0][j] = NewCell(' ')
	}
}

// scroll scrolls the screen up by one line
func (vt *VirtualTerminal) scroll() {
	if !vt.useAltScreen {
		line := vt.screenLineLocked(0)
		if line != "" {
			vt.history = append(vt.history, line)
			// Trim history if too large
			if len(vt.history) > vt.maxHistory {
				vt.history = vt.history[1:]
			}
		}

		// Save styled history line (cells with color/attribute info)
		styledLine := append([]Cell(nil), vt.cells[0]...)
		vt.historyStyled = append(vt.historyStyled, styledLine)
		vt.historyIsWrapped = append(vt.historyIsWrapped, vt.isWrapped[0])

		// Trim styled history if too large
		if len(vt.historyStyled) > vt.maxHistory {
			vt.historyStyled = vt.historyStyled[1:]
			vt.historyIsWrapped = vt.historyIsWrapped[1:]
		}
	}

	// Scroll screen up
	for i := 0; i < vt.rows-1; i++ {
		vt.screen[i] = vt.screen[i+1]
		vt.cells[i] = vt.cells[i+1]
		vt.isWrapped[i] = vt.isWrapped[i+1]
	}

	// Clear bottom line
	vt.screen[vt.rows-1] = make([]rune, vt.cols)
	vt.cells[vt.rows-1] = make([]Cell, vt.cols)
	vt.isWrapped[vt.rows-1] = false
	for j := range vt.screen[vt.rows-1] {
		vt.screen[vt.rows-1][j] = ' '
		vt.cells[vt.rows-1][j] = NewCell(' ')
	}
}
