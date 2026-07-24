package vt

func (vt *VirtualTerminal) appendToPrecedingGrapheme(ch rune) bool {
	if vt.cursorY < 0 || vt.cursorY >= vt.rows || vt.cols <= 0 {
		return false
	}
	// Delayed wrap leaves the cursor at cols, immediately after the final cell.
	col := min(vt.cursorX-1, vt.cols-1)
	if col < 0 || col >= len(vt.cells[vt.cursorY]) {
		return false
	}
	if vt.cells[vt.cursorY][col].IsPlaceholder() && col > 0 {
		col--
	}
	cell := &vt.cells[vt.cursorY][col]
	if !cell.HasContent || cell.Char == 0 || col+int(cell.Width) != vt.cursorX ||
		!graphemeJoins(cell.Text(), ch) {
		return false
	}

	oldWidth := int(cell.Width)
	newWidth := graphemeClusterWidth(*cell, ch)
	if newWidth > oldWidth && vt.cursorX+newWidth-oldWidth > vt.cols {
		if !vt.privateModes[7] {
			// xterm drops a joining code point that would widen a right-margin cell.
			vt.cursorX = vt.cols - 1
			return true
		}
		col, cell = vt.moveGraphemeToWrappedLine(col, *cell)
	}

	cell.Combining += string(ch)
	if newWidth > int(cell.Width) {
		cell.Width = uint8(newWidth)
		placeholderCol := col + newWidth - 1
		vt.screen[vt.cursorY][placeholderCol] = 0
		placeholder := NewFullStyledCell(
			0, cell.Fg, cell.Bg, cell.Attrs, 0,
			cell.UnderlineStyle, cell.UnderlineColor,
		)
		placeholder.HasContent = false
		vt.cells[vt.cursorY][placeholderCol] = placeholder
		vt.cursorX += newWidth - oldWidth
	}
	vt.canJoinGrapheme = true
	return true
}

func (vt *VirtualTerminal) moveGraphemeToWrappedLine(col int, cell Cell) (int, *Cell) {
	blank := NewFullStyledCell(
		' ', vt.currentFg, vt.currentBg, vt.currentAttrs, 1,
		vt.currentUnderlineStyle, vt.currentUnderlineColor,
	)
	for current := col; current < vt.cols; current++ {
		vt.screen[vt.cursorY][current] = ' '
		vt.cells[vt.cursorY][current] = blank
	}

	if vt.cursorY+1 < vt.rows {
		vt.isWrapped[vt.cursorY+1] = true
		vt.cursorY++
	} else {
		vt.scroll()
		vt.cursorY = vt.rows - 1
		vt.isWrapped[vt.cursorY] = true
	}
	vt.cursorX = int(cell.Width)
	vt.screen[vt.cursorY][0] = cell.Char
	vt.cells[vt.cursorY][0] = cell
	if cell.Width == 2 {
		vt.screen[vt.cursorY][1] = 0
		placeholder := NewFullStyledCell(
			0, cell.Fg, cell.Bg, cell.Attrs, 0,
			cell.UnderlineStyle, cell.UnderlineColor,
		)
		vt.cells[vt.cursorY][1] = placeholder
	}
	return 0, &vt.cells[vt.cursorY][0]
}
