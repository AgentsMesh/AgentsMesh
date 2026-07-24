package vt

import (
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

func (vt *VirtualTerminal) initScreen() {
	vt.canJoinGrapheme = false
	vt.screen, vt.cells = newScreenBuffers(vt.cols, vt.rows)
	vt.isWrapped = make([]bool, vt.rows)
	if vt.useAltScreen {
		vt.altScreen = vt.screen
		vt.altCells = vt.cells
		vt.altIsWrapped = vt.isWrapped
	}
	vt.cursorX = 0
	vt.cursorY = 0
	vt.currentFg = DefaultColor()
	vt.currentBg = DefaultColor()
	vt.currentAttrs = AttrNone
	vt.currentUnderlineStyle = UnderlineNone
	vt.currentUnderlineColor = DefaultColor()
}

func newScreenBuffers(cols, rows int) ([][]rune, [][]Cell) {
	screen := make([][]rune, rows)
	cells := make([][]Cell, rows)
	for row := 0; row < rows; row++ {
		screen[row] = make([]rune, cols)
		cells[row] = make([]Cell, cols)
		for col := 0; col < cols; col++ {
			screen[row][col] = ' '
			cells[row][col] = NewCell(' ')
		}
	}
	return screen, cells
}

// Resize changes geometry without destroying the terminal state used for
// reconnect snapshots. The real PTY may redraw after SIGWINCH, but snapshot
// correctness cannot depend on that redraw arriving before a new subscriber.
func (vt *VirtualTerminal) Resize(cols, rows int) {
	lockStart := time.Now()
	vt.mu.Lock()
	lockWait := time.Since(lockStart)
	defer vt.mu.Unlock()

	if lockWait > 10*time.Millisecond {
		logger.Terminal().Warn("VT Resize lock acquisition slow",
			"lock_wait", lockWait, "cols", cols, "rows", rows)
	}

	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	if cols == vt.cols && rows == vt.rows {
		return
	}

	if vt.useAltScreen {
		main := vt.resizeNormalBuffer(
			vt.savedMainCells,
			vt.savedMainWrapped,
			vt.savedMainCursorState(),
			vt.savedCursor[0],
			cols,
			rows,
		)
		active := resizeTerminalBuffer(vt.cells, vt.isWrapped, cols, rows, vt.cursorY)

		vt.history = main.history
		vt.historyStyled = main.historyStyled
		vt.historyIsWrapped = main.historyWrapped
		vt.savedMainScreen = main.screen
		vt.savedMainCells = main.cells
		vt.savedMainWrapped = main.wrapped
		vt.altCursorX = main.cursor.x
		vt.altCursorY = main.cursor.y
		vt.savedCursor[0] = main.savedCursor
		vt.savedCursor[1] = resizeCursorRegister(
			vt.savedCursor[1], cols, rows, active.rowOffset,
		)

		vt.screen = active.screen
		vt.cells = active.cells
		vt.isWrapped = active.wrapped
		vt.cursorX, vt.cursorY = resizeCursor(
			vt.cursorX, vt.cursorY, cols, rows, active.rowOffset,
		)
	} else {
		main := vt.resizeNormalBuffer(
			vt.cells,
			vt.isWrapped,
			vt.currentCursorState(),
			vt.savedCursor[0],
			cols,
			rows,
		)
		vt.history = main.history
		vt.historyStyled = main.historyStyled
		vt.historyIsWrapped = main.historyWrapped
		vt.screen = main.screen
		vt.cells = main.cells
		vt.isWrapped = main.wrapped
		vt.cursorX = main.cursor.x
		vt.cursorY = main.cursor.y
		vt.savedCursor[0] = main.savedCursor
		vt.savedCursor[1] = resizeCursorRegister(
			vt.savedCursor[1], cols, rows, 0,
		)
	}

	vt.cols = cols
	vt.rows = rows
	vt.canJoinGrapheme = false

	if vt.useAltScreen {
		// The active planes must keep aliasing their named alternate buffers;
		// exitAltScreen swaps the separately resized hidden main planes back in.
		vt.altScreen = vt.screen
		vt.altCells = vt.cells
		vt.altIsWrapped = vt.isWrapped
	}
}
