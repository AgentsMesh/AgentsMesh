package vt

// processChar processes a single character
func (vt *VirtualTerminal) processChar(ch rune) {
	switch ch {
	case '\n':
		vt.canJoinGrapheme = false
		vt.newLine()
	case '\r':
		vt.canJoinGrapheme = false
		vt.cursorX = 0
	case '\b':
		vt.canJoinGrapheme = false
		if vt.cursorX > 0 {
			vt.cursorX--
		}
	case '\t':
		vt.canJoinGrapheme = false
		// Move to next tab stop (every 8 columns)
		vt.cursorX = ((vt.cursorX / 8) + 1) * 8
		if vt.cursorX >= vt.cols {
			vt.cursorX = vt.cols - 1
		}
	case '\x1b':
		vt.canJoinGrapheme = false
		// Start of escape sequence - handled by stripping later
	case '\x7f':
		// xterm ignores DEL in ground state without resetting Unicode join state.
	default:
		if ch >= ' ' {
			vt.putChar(ch)
		} else {
			vt.canJoinGrapheme = false
		}
	}
}

// putChar puts a character at the current cursor position
func (vt *VirtualTerminal) putChar(ch rune) {
	if vt.canJoinGrapheme && vt.appendToPrecedingGrapheme(ch) {
		return
	}
	vt.canJoinGrapheme = false
	width := standaloneGraphemeWidth(ch)
	vt.clearWideOwnerAtCursor()

	// Handle line wrap when cursor reaches end of line
	// For wide chars, need to check if there's room for both cells
	if vt.cursorX+width > vt.cols && vt.privateModes[7] {
		// xterm clears cells that cannot be retained before wrapping a wide
		// character from the right margin. A delayed wrap starts at cols and
		// therefore leaves the completed previous row untouched.
		for col := vt.cursorX; col < vt.cols; col++ {
			vt.screen[vt.cursorY][col] = ' '
			vt.cells[vt.cursorY][col] = NewFullStyledCell(
				' ', vt.currentFg, vt.currentBg, vt.currentAttrs, 1,
				vt.currentUnderlineStyle, vt.currentUnderlineColor,
			)
		}
		// Mark the next line as wrapped (soft wrap)
		if vt.cursorY+1 < vt.rows {
			vt.isWrapped[vt.cursorY+1] = true
		}
		vt.newLine()
	} else if vt.cursorX+width > vt.cols {
		// With DECAWM disabled, single-width characters overwrite the last
		// column. A wide character that cannot fit is ignored by xterm.
		vt.cursorX = vt.cols - 1
		if width > 1 {
			return
		}
	}

	if vt.cursorY >= 0 && vt.cursorY < vt.rows && vt.cursorX >= 0 && vt.cursorX < vt.cols {
		// Handle overwriting wide characters:
		currentCell := vt.cells[vt.cursorY][vt.cursorX]

		// If we're overwriting a wide char (width 2), clear its placeholder
		if currentCell.Width == 2 && vt.cursorX+1 < vt.cols {
			vt.screen[vt.cursorY][vt.cursorX+1] = ' '
			vt.cells[vt.cursorY][vt.cursorX+1] = NewCell(' ')
		}

		// If we're writing a wide char and it will overlap with something
		if width == 2 && vt.cursorX+1 < vt.cols {
			nextCell := vt.cells[vt.cursorY][vt.cursorX+1]
			// If next cell is placeholder of a wide char, clear the wide char before it
			if nextCell.IsPlaceholder() && vt.cursorX > 0 {
				// The wide char is at cursorX (which we're overwriting anyway)
			}
			// If next cell is a wide char, clear it and its placeholder
			if nextCell.Width == 2 {
				vt.screen[vt.cursorY][vt.cursorX+1] = ' '
				vt.cells[vt.cursorY][vt.cursorX+1] = NewCell(' ')
				if vt.cursorX+2 < vt.cols && vt.cells[vt.cursorY][vt.cursorX+2].IsPlaceholder() {
					vt.screen[vt.cursorY][vt.cursorX+2] = ' '
					vt.cells[vt.cursorY][vt.cursorX+2] = NewCell(' ')
				}
			}
		}

		vt.screen[vt.cursorY][vt.cursorX] = ch
		// Update styled cell with full style information
		cell := NewFullStyledCell(
			ch,
			vt.currentFg,
			vt.currentBg,
			vt.currentAttrs,
			uint8(width),
			vt.currentUnderlineStyle,
			vt.currentUnderlineColor,
		)
		cell.HasContent = true
		vt.cells[vt.cursorY][vt.cursorX] = cell
		vt.cursorX++

		// For wide characters (CJK), add placeholder cell
		if width == 2 && vt.cursorX < vt.cols {
			vt.screen[vt.cursorY][vt.cursorX] = 0 // Placeholder
			vt.cells[vt.cursorY][vt.cursorX] = NewFullStyledCell(
				0, // No character
				vt.currentFg,
				vt.currentBg,
				vt.currentAttrs,
				0, // Width 0 = placeholder
				vt.currentUnderlineStyle,
				vt.currentUnderlineColor,
			)
			vt.cursorX++
		}
		vt.canJoinGrapheme = true
	} else {
		vt.cursorX++
	}
}

func (vt *VirtualTerminal) clearWideOwnerAtCursor() {
	if vt.cursorY < 0 || vt.cursorY >= vt.rows ||
		vt.cursorX <= 0 || vt.cursorX >= vt.cols ||
		!vt.cells[vt.cursorY][vt.cursorX].IsPlaceholder() {
		return
	}
	vt.screen[vt.cursorY][vt.cursorX-1] = ' '
	vt.cells[vt.cursorY][vt.cursorX-1] = NewCell(' ')
}

// newLine moves to the next line, scrolling if necessary
func (vt *VirtualTerminal) newLine() {
	vt.cursorX = 0
	vt.cursorY++
	if vt.cursorY >= vt.rows {
		vt.scroll()
		vt.cursorY = vt.rows - 1
	}
}
