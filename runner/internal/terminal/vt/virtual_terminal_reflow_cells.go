package vt

import "strings"

func reflowLogicalGroup(
	group []terminalBufferLine,
	oldCols, cols int,
) ([]terminalBufferLine, []int, []int) {
	sourceStarts := make([]int, len(group)+1)
	flat := make([]Cell, 0, len(group)*oldCols)
	for row := range group {
		length := logicalRowLength(group, row, oldCols)
		length = min(length, len(group[row].cells))
		flat = append(flat, group[row].cells[:length]...)
		sourceStarts[row+1] = len(flat)
	}

	if len(flat) == 0 {
		return []terminalBufferLine{{
			cells:   newBlankCellRow(cols),
			wrapped: group[0].wrapped,
		}}, sourceStarts, []int{0}
	}

	result := make([]terminalBufferLine, 0, (len(flat)+cols-1)/cols)
	outputEnds := make([]int, 0, cap(result))
	row := newBlankCellRow(cols)
	col := 0
	consumed := 0
	appendRow := func() {
		result = append(result, terminalBufferLine{
			cells:   row,
			wrapped: len(result) > 0 || group[0].wrapped,
		})
		outputEnds = append(outputEnds, consumed)
		row = newBlankCellRow(cols)
		col = 0
	}

	for index := 0; index < len(flat); {
		cell := flat[index]
		width := 1
		if cell.Width == 2 && index+1 < len(flat) && flat[index+1].IsPlaceholder() {
			width = 2
		}
		if width == 2 && cols == 1 {
			cell = NewCell(' ')
			width = 1
			index += 2
			consumed += 2
		} else {
			if width == 2 && col+2 > cols {
				appendRow()
			}
			index += width
			consumed += width
		}

		if cell.IsPlaceholder() {
			cell = NewCell(' ')
		}
		row[col] = cell
		col++
		if width == 2 {
			row[col] = flat[index-1]
			col++
		}
		if col == cols && index < len(flat) {
			appendRow()
		}
	}
	appendRow()
	for row := range result {
		sanitizeCells(result[row].cells)
	}
	return result, sourceStarts, outputEnds
}

func logicalRowLength(group []terminalBufferLine, row, cols int) int {
	if row == len(group)-1 {
		return trimmedCellLength(group[row].cells)
	}
	length := min(cols, len(group[row].cells))
	if length > 0 && group[row].cells[length-1].IsEmpty() &&
		len(group[row+1].cells) > 0 && group[row+1].cells[0].Width == 2 {
		return length - 1
	}
	return length
}

func trimmedCellLength(cells []Cell) int {
	for col := len(cells) - 1; col >= 0; col-- {
		if !cells[col].IsEmpty() {
			return col + 1
		}
	}
	return 0
}

func blankTerminalBufferLine(cols int) terminalBufferLine {
	return terminalBufferLine{cells: newBlankCellRow(cols)}
}

func newBlankCellRow(cols int) []Cell {
	row := make([]Cell, cols)
	for col := range row {
		row[col] = NewCell(' ')
	}
	return row
}

func resizeCellsPhysical(source []Cell, cols int) []Cell {
	cells := newBlankCellRow(cols)
	copy(cells, source[:min(cols, len(source))])
	sanitizeCells(cells)
	return cells
}

func resizeRowOffset(source [][]Cell, rows, cursorY int) int {
	if len(source) <= rows {
		return 0
	}
	return min(len(source)-rows, max(0, cursorY-rows+1))
}

func sanitizeResizedRow(screen []rune, cells []Cell) {
	sanitizeCells(cells)
	for col := range cells {
		screen[col] = cells[col].Char
	}
}

func sanitizeCells(cells []Cell) {
	for col := range cells {
		cell := cells[col]
		switch {
		case cell.Width == 2:
			if col+1 >= len(cells) || !cells[col+1].IsPlaceholder() {
				cells[col] = NewCell(' ')
			}
		case cell.IsPlaceholder():
			if col == 0 || cells[col-1].Width != 2 {
				cells[col] = NewCell(' ')
			}
		}
	}
}

func resizeCursor(x, y, cols, rows, rowOffset int) (int, int) {
	y = min(max(y-rowOffset, 0), rows-1)
	// A resize cancels delayed autowrap. Growing keeps the old physical cursor
	// column; shrinking clamps it to the new rightmost cell.
	x = min(max(x, 0), cols-1)
	return x, y
}

func resizeCursorRegister(
	state cursorState,
	cols, rows, rowOffset int,
) cursorState {
	state.x, state.y = resizeCursor(state.x, state.y, cols, rows, rowOffset)
	return state
}

func resizedRowText(cells []Cell) string {
	var line strings.Builder
	for _, cell := range cells {
		if cell.IsPlaceholder() {
			continue
		}
		if cell.HasContent {
			line.WriteString(cell.Text())
		} else {
			line.WriteByte(' ')
		}
	}
	return strings.TrimRight(line.String(), " ")
}
