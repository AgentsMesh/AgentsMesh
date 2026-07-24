package vt

type resizedTerminalBuffer struct {
	screen    [][]rune
	cells     [][]Cell
	wrapped   []bool
	rowOffset int
}

func resizeTerminalBuffer(
	source [][]Cell,
	sourceWrapped []bool,
	cols, rows, cursorY int,
) resizedTerminalBuffer {
	screen, cells := newScreenBuffers(cols, rows)
	wrapped := make([]bool, rows)
	rowOffset := resizeRowOffset(source, rows, cursorY)
	copyRows := min(rows, max(0, len(source)-rowOffset))

	for dstRow := 0; dstRow < copyRows; dstRow++ {
		srcRow := dstRow + rowOffset
		copy(cells[dstRow], source[srcRow][:min(cols, len(source[srcRow]))])
		if srcRow < len(sourceWrapped) {
			wrapped[dstRow] = sourceWrapped[srcRow]
		}
		sanitizeResizedRow(screen[dstRow], cells[dstRow])
	}
	if rowOffset > 0 && len(wrapped) > 0 {
		// Alternate buffers have no scrollback. Once the predecessor is cropped,
		// the first retained physical row must stand on its own.
		wrapped[0] = false
	}

	return resizedTerminalBuffer{
		screen: screen, cells: cells, wrapped: wrapped, rowOffset: rowOffset,
	}
}

type terminalBufferLine struct {
	cells   []Cell
	wrapped bool
}

type resizedNormalBuffer struct {
	screen         [][]rune
	cells          [][]Cell
	wrapped        []bool
	history        []string
	historyStyled  [][]Cell
	historyWrapped []bool
	cursor         cursorState
	savedCursor    cursorState
}

// resizeNormalBuffer models the xterm normal buffer as one sequence containing
// scrollback followed by the visible viewport. Row growth can therefore expose
// recent history, while column reflow can move physical rows across that
// boundary without losing their styles or wrap ownership.
func (vt *VirtualTerminal) resizeNormalBuffer(
	source [][]Cell,
	sourceWrapped []bool,
	cursor cursorState,
	savedCursor cursorState,
	cols, rows int,
) resizedNormalBuffer {
	oldCols := vt.cols
	oldRows := vt.rows
	lines := make([]terminalBufferLine, 0, len(vt.historyStyled)+oldRows)
	for row, cells := range vt.historyStyled {
		lines = append(lines, terminalBufferLine{
			cells:   resizeCellsPhysical(cells, oldCols),
			wrapped: row < len(vt.historyIsWrapped) && vt.historyIsWrapped[row],
		})
	}
	for row := 0; row < oldRows; row++ {
		var cells []Cell
		if row < len(source) {
			cells = source[row]
		}
		lines = append(lines, terminalBufferLine{
			cells:   resizeCellsPhysical(cells, oldCols),
			wrapped: row < len(sourceWrapped) && sourceWrapped[row],
		})
	}

	ybase := len(vt.historyStyled)
	cursorAbsolute := ybase + min(max(cursor.y, 0), oldRows-1)
	lines, ybase = resizeNormalRows(lines, ybase, cursorAbsolute, oldRows, rows, oldCols)

	// xterm reduces the circular buffer limit before column reflow. A row
	// shrink can therefore trim old scrollback immediately; savedY is an
	// absolute normal-buffer coordinate and follows that trim.
	maxLength := vt.maxHistory + rows
	if trim := max(0, len(lines)-maxLength); trim > 0 {
		lines = lines[trim:]
		ybase = max(ybase-trim, 0)
		cursorAbsolute = max(cursorAbsolute-trim, 0)
		savedCursor.y = max(savedCursor.y-trim, 0)
	}

	cursor.x = min(max(cursor.x, 0), cols-1)
	cursor.y = min(max(cursorAbsolute-ybase, 0), rows-1)
	savedCursor.x = min(max(savedCursor.x, 0), cols-1)

	if oldCols != cols {
		var changes []int
		var originalRows []bool
		originalLength := len(lines)
		lines, changes, originalRows = reflowNormalLines(
			lines, ybase+cursor.y, oldCols, cols,
		)
		if cols > oldCols {
			lines, ybase, cursor.y, savedCursor.y = adjustReflowLarger(
				lines, changes, ybase, cursor.y, savedCursor.y, rows, cols,
			)
		} else {
			lines, ybase, cursor.y, savedCursor.y = adjustReflowSmaller(
				lines, originalRows, changes, originalLength, ybase, cursor.y,
				savedCursor.y, rows, cols, maxLength,
			)
		}
	} else {
		for row := range lines {
			lines[row].cells = resizeCellsPhysical(lines[row].cells, cols)
		}
	}

	targetLength := ybase + rows
	if len(lines) > targetLength {
		lines = lines[:targetLength]
	}
	for len(lines) < targetLength {
		lines = append(lines, blankTerminalBufferLine(cols))
	}

	cursor.y = min(max(cursor.y, 0), rows-1)
	savedCursor.y = max(savedCursor.y, 0)

	historyStyled := make([][]Cell, ybase)
	historyWrapped := make([]bool, ybase)
	history := make([]string, 0, ybase)
	for row := 0; row < ybase; row++ {
		historyStyled[row] = append([]Cell(nil), lines[row].cells...)
		historyWrapped[row] = lines[row].wrapped
		if text := resizedRowText(lines[row].cells); text != "" {
			history = append(history, text)
		}
	}

	screen, cells := newScreenBuffers(cols, rows)
	wrapped := make([]bool, rows)
	for row := 0; row < rows; row++ {
		line := lines[ybase+row]
		copy(cells[row], line.cells)
		wrapped[row] = line.wrapped
		sanitizeResizedRow(screen[row], cells[row])
	}
	return resizedNormalBuffer{
		screen:         screen,
		cells:          cells,
		wrapped:        wrapped,
		history:        history,
		historyStyled:  historyStyled,
		historyWrapped: historyWrapped,
		cursor:         cursor,
		savedCursor:    savedCursor,
	}
}
