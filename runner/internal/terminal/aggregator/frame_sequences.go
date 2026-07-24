// Package terminal provides terminal management for PTY sessions.
package aggregator

import (
	"bytes"
)

// Frame boundary sequences for TUI applications
var (
	// Legacy: ANSI clear screen sequence: ESC[2J
	// Used by traditional terminal apps like `clear` command
	clearScreenSeq = []byte{0x1b, '[', '2', 'J'}

	// Modern: Synchronized Output sequences: ESC[?2026h (start) and ESC[?2026l (end)
	// Used by Claude Code and modern TUI frameworks (Ink, Bubbletea, etc.)
	// Reference: https://gist.github.com/christianparpart/d8a62cc1ab659194337d73e399004036
	//
	// A complete frame looks like: ESC[?2026h <content> ESC[?2026l
	syncOutputStartSeq = []byte{0x1b, '[', '?', '2', '0', '2', '6', 'h'}
	syncOutputEndSeq   = []byte{0x1b, '[', '?', '2', '0', '2', '6', 'l'}
)

// Full redraw sequences. Synchronized output only makes a frame render atomically;
// it does not replace the erase-screen or cursor-home semantics of these sequences.
var (
	// ESC[2J - Erase entire screen
	eraseScreenSeq = []byte{0x1b, '[', '2', 'J'}
	// ESC[H or ESC[;H - Cursor home (move to 0,0)
	cursorHomeSeq  = []byte{0x1b, '[', 'H'}
	cursorHomeSeq2 = []byte{0x1b, '[', ';', 'H'}
)

// findAllPositions finds all occurrences of seq in data and returns their positions.
func findAllPositions(data, seq []byte) []int {
	var positions []int
	searchStart := 0
	for {
		idx := bytes.Index(data[searchStart:], seq)
		if idx < 0 {
			break
		}
		pos := searchStart + idx
		positions = append(positions, pos)
		searchStart = pos + 1
	}
	return positions
}
