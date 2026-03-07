package vt

import (
	"fmt"
	"strings"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// TerminalSnapshot represents a complete terminal state for relay transmission
type TerminalSnapshot struct {
	Cols              int      `json:"cols"`
	Rows              int      `json:"rows"`
	Lines             []string `json:"lines"`              // Plain text lines (kept for compatibility)
	SerializedContent string   `json:"serialized_content"` // ANSI-escaped serialized content for xterm.js
	CursorX           int      `json:"cursor_x"`
	CursorY           int      `json:"cursor_y"`
	CursorVisible     bool     `json:"cursor_visible"`
	IsAltScreen       bool     `json:"is_alt_screen"` // Whether currently in alternate screen mode (TUI apps)
}

// TryGetSnapshot attempts to get a terminal snapshot without blocking.
// Returns nil if the lock cannot be acquired immediately (e.g., Feed is in progress).
// This is useful for periodic polling where skipping a snapshot is acceptable.
func (vt *VirtualTerminal) TryGetSnapshot() *TerminalSnapshot {
	log := logger.TerminalTrace()
	if !vt.mu.TryRLock() {
		log.Trace("VirtualTerminal.TryGetSnapshot: lock busy, skipping")
		return nil // Lock held by Feed(), skip this tick
	}
	log.Trace("VirtualTerminal.TryGetSnapshot: got RLock")
	defer func() {
		vt.mu.RUnlock()
		log.Trace("VirtualTerminal.TryGetSnapshot: released RLock")
	}()
	return vt.getSnapshotLocked()
}

// TryGetLines attempts to get terminal screen lines without blocking.
// Returns nil if the lock cannot be acquired immediately.
// This is a lightweight alternative to TryGetSnapshot for state detection
// that only needs text content, not serialized ANSI output.
func (vt *VirtualTerminal) TryGetLines() []string {
	if !vt.mu.TryRLock() {
		return nil // Lock held by Feed(), skip this tick
	}
	defer vt.mu.RUnlock()

	// Quick extraction of screen lines - no serialization needed
	screen := vt.screen
	lines := make([]string, vt.rows)
	for row := 0; row < vt.rows; row++ {
		var line strings.Builder
		if row < len(screen) {
			for _, ch := range screen[row] {
				if ch == 0 {
					line.WriteRune(' ')
				} else {
					line.WriteRune(ch)
				}
			}
		}
		lines[row] = strings.TrimRight(line.String(), " ")
	}
	return lines
}

// GetSnapshot returns a complete terminal snapshot for relay transmission.
// When in alternate screen mode (TUI apps like Claude Code), returns the alt screen content.
// The snapshot includes SerializedContent with ANSI escape sequences for proper xterm.js rendering.
func (vt *VirtualTerminal) GetSnapshot() *TerminalSnapshot {
	log := logger.TerminalTrace()
	log.Trace("VirtualTerminal.GetSnapshot: acquiring RLock")
	vt.mu.RLock()
	log.Trace("VirtualTerminal.GetSnapshot: got RLock")
	defer func() {
		vt.mu.RUnlock()
		log.Trace("VirtualTerminal.GetSnapshot: released RLock")
	}()
	return vt.getSnapshotLocked()
}

// getSnapshotLocked returns a terminal snapshot. Caller must hold vt.mu (read or write).
func (vt *VirtualTerminal) getSnapshotLocked() *TerminalSnapshot {
	log := logger.TerminalTrace()
	log.Trace("VirtualTerminal.getSnapshotLocked: ENTER", "rows", vt.rows, "cols", vt.cols, "hasData", vt.hasData)

	// Use the current screen buffer (which points to altScreen when in alt mode)
	// This is already set correctly by enterAltScreen/exitAltScreen
	screen := vt.screen

	log.Trace("VirtualTerminal.getSnapshotLocked: collecting lines", "screen_len", len(screen))

	// Collect all visible lines from the screen buffer (plain text for backward compatibility)
	lines := make([]string, vt.rows)
	for row := 0; row < vt.rows; row++ {
		var line strings.Builder
		if row < len(screen) {
			for _, ch := range screen[row] {
				if ch == 0 {
					line.WriteRune(' ')
				} else {
					line.WriteRune(ch)
				}
			}
		}
		lines[row] = strings.TrimRight(line.String(), " ")
	}

	log.Trace("VirtualTerminal.getSnapshotLocked: lines collected, serializing", "hasData", vt.hasData)

	// Generate serialized content with ANSI sequences for proper xterm.js rendering.
	// Use the existing Serialize() method which is well-tested.
	// Serialize when hasData is true, even if screen appears empty (might have control sequences).
	// This handles cases like TUI apps that clear screen and set cursor position without visible text.
	var serialized string
	if vt.hasData {
		log.Trace("VirtualTerminal.getSnapshotLocked: calling serializeNoLock")
		serialized = vt.serializeNoLock(SerializeOptions{
			ScrollbackLines:  0,     // Don't include scrollback history
			ExcludeAltBuffer: false, // Include alt buffer if in alt screen mode
			ExcludeModes:     true,  // Don't include mode sequences (DECCKM, etc.)
		})
		log.Trace("VirtualTerminal.getSnapshotLocked: serializeNoLock done", "serialized_len", len(serialized))
	}

	log.Trace("VirtualTerminal.getSnapshotLocked: EXIT")

	return &TerminalSnapshot{
		Cols:              vt.cols,
		Rows:              vt.rows,
		Lines:             lines,
		SerializedContent: serialized,
		CursorX:           vt.cursorX,
		CursorY:           vt.cursorY,
		CursorVisible:     true,            // Default to visible
		IsAltScreen:       vt.useAltScreen, // Indicate whether in alternate screen mode
	}
}

// serializeScreenOnly serializes only the current screen buffer (no history).
// This method is called with the lock already held.
func (vt *VirtualTerminal) serializeScreenOnly() string {
	// Check if screen has any content
	hasContent := false
	for row := 0; row < vt.rows && !hasContent; row++ {
		if row < len(vt.screen) {
			for _, ch := range vt.screen[row] {
				if ch != 0 && ch != ' ' {
					hasContent = true
					break
				}
			}
		}
	}

	if !hasContent {
		// Screen is empty, return empty string
		return ""
	}

	// Create handler and serialize only the visible screen (no history)
	handler := newStringSerializeHandler(vt)
	return handler.serializeScreenNoLock(0, vt.rows-1, false)
}

// serializeScreenNoLock serializes the screen buffer without acquiring the lock.
// Called with the lock already held.
func (h *StringSerializeHandler) serializeScreenNoLock(startRow, endRow int, excludeFinalCursorPosition bool) string {
	rowCount := endRow - startRow + 1
	h.allRows = make([]string, rowCount)
	h.allRowSeparators = make([]string, rowCount)
	h.firstRow = startRow
	h.lastContentCursorRow = startRow
	h.lastCursorRow = startRow
	h.lastCursorCol = 0
	h.lastContentCursorCol = 0
	h.rowIndex = 0

	// Process each row from the screen buffer
	var prevCell = NewCell(' ')
	for row := startRow; row <= endRow; row++ {
		// Reset per-row state
		h.currentRow.Reset()
		h.nullCellCount = 0

		cells := h.vt.cells[row]
		if cells != nil {
			for col := 0; col < len(cells); col++ {
				cell := cells[col]
				h.nextCell(cell, prevCell, row, col)
				prevCell = cell
			}
		}

		// Check if next line is wrapped
		isLastRow := row == endRow
		var nextLineWrapped bool
		if !isLastRow && row+1 < h.vt.rows {
			nextLineWrapped = h.vt.isWrapped[row+1]
		}
		h.rowEndScreenOnly(row, isLastRow, nextLineWrapped)
	}

	return h.serializeStringScreenOnly(startRow, endRow, excludeFinalCursorPosition)
}

// rowEndScreenOnly handles end of row processing for screen-only serialization
func (h *StringSerializeHandler) rowEndScreenOnly(row int, isLastRow bool, nextLineWrapped bool) {
	// If there are colorful empty cells at line end, we must preserve them
	if h.nullCellCount > 0 && !h.cursorStyle.Bg.Equals(h.backgroundCell.Bg) {
		fmt.Fprintf(&h.currentRow, "\x1b[%dX", h.nullCellCount)
	}

	rowSeparator := ""

	if !isLastRow {
		if !nextLineWrapped {
			// Not wrapped - insert CRLF
			rowSeparator = "\r\n"
			h.lastCursorRow = row + 1
			h.lastCursorCol = 0
		} else {
			// Line is wrapped - no separator needed
			rowSeparator = ""
			h.lastContentCursorRow = row + 1
			h.lastContentCursorCol = 0
			h.lastCursorRow = row + 1
			h.lastCursorCol = 0
		}
	}

	h.allRows[h.rowIndex] = h.currentRow.String()
	h.allRowSeparators[h.rowIndex] = rowSeparator
	h.rowIndex++
	h.currentRow.Reset()
	h.nullCellCount = 0
}

// serializeStringScreenOnly builds the final serialized string for screen-only content
func (h *StringSerializeHandler) serializeStringScreenOnly(startRow, endRow int, excludeFinalCursorPosition bool) string {
	var content strings.Builder

	// Calculate how many rows to include
	rowEnd := len(h.allRows)

	// Trim trailing empty rows within screen size
	bufferLength := endRow - startRow + 1
	if bufferLength <= h.vt.rows {
		rowEnd = h.lastContentCursorRow + 1 - h.firstRow
		if rowEnd < 0 {
			rowEnd = 0
		}
		if rowEnd > len(h.allRows) {
			rowEnd = len(h.allRows)
		}
		h.lastCursorCol = h.lastContentCursorCol
		h.lastCursorRow = h.lastContentCursorRow
	}

	// Build content
	for i := 0; i < rowEnd; i++ {
		content.WriteString(h.allRows[i])
		if i+1 < rowEnd {
			content.WriteString(h.allRowSeparators[i])
		}
	}

	// Restore cursor position using absolute positioning (CUP)
	if !excludeFinalCursorPosition {
		// Use CUP (Cursor Position) - 1-indexed
		cursorRow := h.vt.cursorY + 1
		cursorCol := h.vt.cursorX + 1
		fmt.Fprintf(&content, "\x1b[%d;%dH", cursorRow, cursorCol)
	}

	// Restore cursor's current style
	curFg, curBg, curAttrs, curUlStyle, curUlColor := h.vt.getCurrentStyleNoLock()
	curCell := NewFullStyledCell(' ', curFg, curBg, curAttrs, 1, curUlStyle, curUlColor)
	sgrSeq := h.diffStyle(curCell, h.cursorStyle)
	if len(sgrSeq) > 0 {
		content.WriteString("\x1b[")
		content.WriteString(strings.Join(sgrSeq, ";"))
		content.WriteString("m")
	}

	return content.String()
}
