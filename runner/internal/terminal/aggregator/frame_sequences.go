// Package terminal provides terminal management for PTY sessions.
package aggregator

import (
	"bytes"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
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

// Sequences that cause xterm.js to jump to top and are redundant inside sync frames
var (
	// ESC[2J - Erase entire screen
	eraseScreenSeq = []byte{0x1b, '[', '2', 'J'}
	// ESC[H or ESC[;H - Cursor home (move to 0,0)
	cursorHomeSeq  = []byte{0x1b, '[', 'H'}
	cursorHomeSeq2 = []byte{0x1b, '[', ';', 'H'}
)

// StripRedundantSequencesInFrames removes ESC[2J and ESC[H sequences from INSIDE
// synchronized output frames. These sequences are redundant because:
//  1. Sync frames already provide atomic updates (no need to clear first)
//  2. After resize, Claude Code sends ESC[2J + ESC[H with every frame, causing xterm.js
//     to continuously jump to top, making scrolling impossible
//
// This does NOT affect:
// - Clear screen sequences OUTSIDE sync frames (e.g., `clear` command)
// - Clear screen sequences in apps that don't use sync output mode
//
// Returns the filtered data (may be the same slice if no changes needed).
func (d *FrameDetector) StripRedundantSequencesInFrames(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	// Quick check: if no sync frames, return as-is
	if !bytes.Contains(data, syncOutputStartSeq) {
		return data
	}

	// Find all frame boundaries
	startPositions := findAllPositions(data, syncOutputStartSeq)
	endPositions := findAllPositions(data, syncOutputEndSeq)

	if len(startPositions) == 0 {
		return data
	}

	// Build frame ranges (start, end) pairs
	type frameRange struct {
		start int // Position after ESC[?2026h
		end   int // Position of ESC[?2026l (or end of data if incomplete)
	}

	var frames []frameRange
	usedEnds := make(map[int]bool)

	for _, startPos := range startPositions {
		frameStart := startPos + len(syncOutputStartSeq)
		frameEnd := len(data) // Default: incomplete frame extends to end

		// Find matching end
		for _, endPos := range endPositions {
			if endPos > startPos && !usedEnds[endPos] {
				usedEnds[endPos] = true
				frameEnd = endPos
				break
			}
		}

		frames = append(frames, frameRange{start: frameStart, end: frameEnd})
	}

	if len(frames) == 0 {
		return data
	}

	// Check if any frame contains sequences to strip
	needsStrip := false
	for _, fr := range frames {
		if fr.start >= fr.end {
			continue
		}
		frameData := data[fr.start:fr.end]
		if bytes.Contains(frameData, eraseScreenSeq) ||
			bytes.Contains(frameData, cursorHomeSeq) ||
			bytes.Contains(frameData, cursorHomeSeq2) {
			needsStrip = true
			break
		}
	}

	if !needsStrip {
		return data
	}

	// Build new data with sequences stripped from inside frames
	result := make([]byte, 0, len(data))
	lastPos := 0

	for _, fr := range frames {
		// Copy everything before frame content
		if fr.start > lastPos {
			result = append(result, data[lastPos:fr.start]...)
		}

		// Process frame content - strip redundant sequences
		frameData := data[fr.start:fr.end]
		cleanedFrame := stripSequences(frameData, eraseScreenSeq, cursorHomeSeq, cursorHomeSeq2)
		result = append(result, cleanedFrame...)

		lastPos = fr.end
	}

	// Copy everything after last frame
	if lastPos < len(data) {
		result = append(result, data[lastPos:]...)
	}

	if len(result) != len(data) {
		logger.Terminal().Debug("FrameDetector: stripped redundant sequences in frames",
			"original_len", len(data), "new_len", len(result),
			"stripped_bytes", len(data)-len(result))
	}

	return result
}

// stripSequences removes all occurrences of the given sequences from data.
func stripSequences(data []byte, seqs ...[]byte) []byte {
	result := data
	for _, seq := range seqs {
		result = bytes.ReplaceAll(result, seq, nil)
	}
	return result
}

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
