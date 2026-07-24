package vt

func resizeNormalRows(
	lines []terminalBufferLine,
	ybase, cursorAbsolute, oldRows, rows, cols int,
) ([]terminalBufferLine, int) {
	cursorY := min(max(cursorAbsolute-ybase, 0), oldRows-1)
	if rows > oldRows {
		added := rows - oldRows
		pulled := 0
		if cursorY == oldRows-1 {
			pulled = min(added, ybase)
			ybase -= pulled
		}
		for count := pulled; count < added; count++ {
			lines = append(lines, blankTerminalBufferLine(cols))
		}
		return lines, ybase
	}
	if rows == oldRows {
		return lines, ybase
	}

	removed := oldRows - rows
	belowCursor := max(oldRows-1-cursorY, 0)
	popCount := min(removed, belowCursor)
	if popCount > 0 {
		lines = lines[:len(lines)-popCount]
	}
	ybase += removed - popCount
	return lines, ybase
}

func reflowNormalLines(
	lines []terminalBufferLine,
	cursorAbsolute int,
	oldCols, cols int,
) ([]terminalBufferLine, []int, []bool) {
	result := make([]terminalBufferLine, 0, len(lines))
	changes := make([]int, 0, len(lines))
	originalRows := make([]bool, 0, len(lines))

	for start := 0; start < len(lines); {
		end := start + 1
		for end < len(lines) && lines[end].wrapped {
			end++
		}
		group := lines[start:end]
		before := len(result)
		if cursorAbsolute >= start && cursorAbsolute < end {
			for _, line := range group {
				result = append(result, terminalBufferLine{
					cells:   resizeCellsPhysical(line.cells, cols),
					wrapped: line.wrapped,
				})
				originalRows = append(originalRows, true)
			}
		} else {
			reflowed, _, _ := reflowLogicalGroup(group, oldCols, cols)
			result = append(result, reflowed...)
			for row := range reflowed {
				originalRows = append(originalRows, row < len(group))
			}
		}
		changes = append(changes, len(result)-before-len(group))
		start = end
	}
	return result, changes, originalRows
}

// adjustReflowLarger mirrors Buffer._reflowLargerAdjustViewport in xterm 6.
// With reflowCursorLine=false, physical rows can be removed anywhere in the
// buffer, but xterm applies the total removal count to viewport state globally.
func adjustReflowLarger(
	lines []terminalBufferLine,
	changes []int,
	ybase, cursorY, savedY, rows, cols int,
) ([]terminalBufferLine, int, int, int) {
	removed := 0
	for _, change := range changes {
		if change < 0 {
			removed -= change
		}
	}
	for count := 0; count < removed; count++ {
		if ybase == 0 {
			if cursorY > 0 {
				cursorY--
			}
			if len(lines) < rows {
				lines = append(lines, blankTerminalBufferLine(cols))
			}
		} else {
			ybase--
		}
	}
	return lines, ybase, cursorY, max(savedY-removed, 0)
}

// adjustReflowSmaller mirrors Buffer._reflowSmaller's backwards viewport
// adjustment. The insertion itself is built above in one pass; this function
// reproduces xterm's global cursor/ybase/savedY effects and then applies the
// same bottom-pop and circular-buffer trimming decisions.
func adjustReflowSmaller(
	lines []terminalBufferLine,
	originalRows []bool,
	changes []int,
	originalLength, ybase, cursorY, savedY, rows, cols, maxLength int,
) ([]terminalBufferLine, int, int, int) {
	workingLength := originalLength
	countToInsert := 0
	poppedBottom := 0

	for index := len(changes) - 1; index >= 0; index-- {
		linesToAdd := max(changes[index], 0)
		if linesToAdd == 0 {
			continue
		}

		trimmedLines := 0
		if ybase == 0 && cursorY != workingLength-1 {
			trimmedLines = max(0, cursorY-maxLength+linesToAdd)
		} else {
			trimmedLines = max(0, workingLength-maxLength+linesToAdd)
		}
		countToInsert += linesToAdd

		for adjustment := linesToAdd - trimmedLines; adjustment > 0; adjustment-- {
			if ybase == 0 {
				if cursorY < rows-1 {
					cursorY++
					workingLength--
					poppedBottom++
				} else {
					ybase++
				}
			} else {
				limit := min(maxLength, workingLength+countToInsert) - rows
				if ybase < limit {
					ybase++
				}
			}
		}
		savedY = min(savedY+linesToAdd, ybase+rows-1)
	}

	if poppedBottom > 0 {
		kept := make([]terminalBufferLine, 0, len(lines)-poppedBottom)
		removeRow := make([]bool, len(lines))
		remove := poppedBottom
		for row := len(lines) - 1; row >= 0; row-- {
			if remove > 0 && row < len(originalRows) && originalRows[row] {
				removeRow[row] = true
				remove--
			}
		}
		for row, line := range lines {
			if removeRow[row] {
				continue
			}
			kept = append(kept, line)
		}
		lines = kept
	}
	if trim := max(0, len(lines)-maxLength); trim > 0 {
		lines = lines[trim:]
	}
	targetLength := ybase + rows
	if len(lines) > targetLength {
		lines = lines[:targetLength]
	}
	for len(lines) < targetLength {
		lines = append(lines, blankTerminalBufferLine(cols))
	}
	return lines, ybase, cursorY, savedY
}
