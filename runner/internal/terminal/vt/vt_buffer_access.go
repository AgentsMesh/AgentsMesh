package vt

// GetCellsRow returns a copy of the cells for a given row
// Used by serializer to access styled cell data
func (vt *VirtualTerminal) GetCellsRow(row int) []Cell {
	vt.mu.RLock()
	defer vt.mu.RUnlock()
	return vt.getCellsRowNoLock(row)
}

// getCellsRowNoLock returns a copy of the cells for a given row without locking (caller must hold lock)
func (vt *VirtualTerminal) getCellsRowNoLock(row int) []Cell {
	if row < 0 || row >= len(vt.cells) {
		return nil
	}
	result := make([]Cell, len(vt.cells[row]))
	copy(result, vt.cells[row])
	return result
}

// IsLineWrapped returns true if the given line is wrapped from the previous line
func (vt *VirtualTerminal) IsLineWrapped(row int) bool {
	vt.mu.RLock()
	defer vt.mu.RUnlock()
	return vt.isLineWrappedNoLock(row)
}

// isLineWrappedNoLock returns true if the given line is wrapped without locking (caller must hold lock)
func (vt *VirtualTerminal) isLineWrappedNoLock(row int) bool {
	if row < 0 || row >= len(vt.isWrapped) {
		return false
	}
	return vt.isWrapped[row]
}

// GetCurrentStyle returns the current text style (used for cursor style serialization)
func (vt *VirtualTerminal) GetCurrentStyle() (fg, bg Color, attrs CellAttrs, ulStyle UnderlineStyle, ulColor Color) {
	vt.mu.RLock()
	defer vt.mu.RUnlock()
	return vt.currentFg, vt.currentBg, vt.currentAttrs, vt.currentUnderlineStyle, vt.currentUnderlineColor
}

// getCurrentStyleNoLock returns the current text style without locking (caller must hold lock)
func (vt *VirtualTerminal) getCurrentStyleNoLock() (fg, bg Color, attrs CellAttrs, ulStyle UnderlineStyle, ulColor Color) {
	return vt.currentFg, vt.currentBg, vt.currentAttrs, vt.currentUnderlineStyle, vt.currentUnderlineColor
}

// GetHistoryStyledRow returns a copy of styled history cells for a given history index
// Index is relative to history start (0 = oldest history line)
func (vt *VirtualTerminal) GetHistoryStyledRow(index int) []Cell {
	vt.mu.RLock()
	defer vt.mu.RUnlock()

	if index < 0 || index >= len(vt.historyStyled) {
		return nil
	}
	result := make([]Cell, len(vt.historyStyled[index]))
	copy(result, vt.historyStyled[index])
	return result
}

// GetHistoryStyledLength returns the number of styled history lines
func (vt *VirtualTerminal) GetHistoryStyledLength() int {
	vt.mu.RLock()
	defer vt.mu.RUnlock()
	return len(vt.historyStyled)
}

// IsHistoryLineWrapped returns true if the given history line was wrapped
func (vt *VirtualTerminal) IsHistoryLineWrapped(index int) bool {
	vt.mu.RLock()
	defer vt.mu.RUnlock()

	if index < 0 || index >= len(vt.historyIsWrapped) {
		return false
	}
	return vt.historyIsWrapped[index]
}
