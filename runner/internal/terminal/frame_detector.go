// Package terminal provides terminal management for PTY sessions.
package terminal

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

// FrameDetector detects Synchronized Output frame boundaries.
// It ensures complete frames are preserved during aggregation and flushing.
//
// The key improvement: instead of just finding the last frame START (which breaks
// incomplete frames), we now detect complete frames and preserve frame integrity.
type FrameDetector struct{}

// NewFrameDetector creates a new frame detector.
func NewFrameDetector() *FrameDetector {
	return &FrameDetector{}
}

// FrameBoundary represents the analysis result of frame boundaries in data.
type FrameBoundary struct {
	// CompleteEnd is the position after the last complete frame's end sequence.
	// -1 if no complete frame found.
	CompleteEnd int

	// IncompleteStart is the position where an incomplete frame begins.
	// -1 if no incomplete frame found.
	IncompleteStart int

	// HasSyncFrames indicates if sync output sequences were found.
	HasSyncFrames bool

	// ClearScreenPos is the position of the last clear screen sequence.
	// -1 if not found.
	ClearScreenPos int
}

// AnalyzeFrameBoundaries finds complete and incomplete frame boundaries in data.
//
// Algorithm:
// 1. Find all frame start (ESC[?2026h) and end (ESC[?2026l) positions
// 2. Match them in order to identify complete frames
// 3. Return the boundary after last complete frame and start of any trailing incomplete frame
func (d *FrameDetector) AnalyzeFrameBoundaries(data []byte) FrameBoundary {
	result := FrameBoundary{
		CompleteEnd:     -1,
		IncompleteStart: -1,
		ClearScreenPos:  -1,
	}

	if len(data) == 0 {
		return result
	}

	// Find all start and end positions
	startPositions := findAllPositions(data, syncOutputStartSeq)
	endPositions := findAllPositions(data, syncOutputEndSeq)

	result.HasSyncFrames = len(startPositions) > 0 || len(endPositions) > 0

	// Also check for clear screen (fallback)
	if idx := bytes.LastIndex(data, clearScreenSeq); idx >= 0 {
		result.ClearScreenPos = idx
	}

	if len(startPositions) == 0 {
		// No sync frames found
		return result
	}

	// Match starts with ends to find complete frames
	// Strategy: iterate through starts, find matching end after each start
	var lastCompleteEnd int = -1
	var usedEnds = make(map[int]bool)

	for _, startPos := range startPositions {
		// Find the first end position after this start that hasn't been used
		for _, endPos := range endPositions {
			if endPos > startPos && !usedEnds[endPos] {
				// This start+end pair forms a complete frame
				usedEnds[endPos] = true
				lastCompleteEnd = endPos + len(syncOutputEndSeq)
				break
			}
		}
	}

	result.CompleteEnd = lastCompleteEnd

	// Check if there's an incomplete frame at the end
	// (a start without a matching end after it)
	if len(startPositions) > 0 {
		lastStart := startPositions[len(startPositions)-1]
		hasMatchingEnd := false
		for _, endPos := range endPositions {
			if endPos > lastStart {
				hasMatchingEnd = true
				break
			}
		}
		if !hasMatchingEnd {
			result.IncompleteStart = lastStart
		}
	}

	return result
}

// DiscardOldFrames intelligently removes old frames based on content analysis.
//
// Strategy:
// - If a frame contains "full redraw" sequences (ESC[2J clear screen, ESC[H cursor home),
//   it's safe to discard everything before that frame.
// - If frames only contain incremental updates (relative cursor movement), we keep them
//   because they depend on previous terminal state.
//
// This is critical for Claude Code which uses both patterns:
// - Full redraws when the UI layout changes significantly
// - Incremental updates for animations (spinner, typing effects)
//
// Returns the number of bytes discarded.
func (d *FrameDetector) DiscardOldFrames(buffer *bytes.Buffer) int {
	data := buffer.Bytes()
	if len(data) == 0 {
		return 0
	}

	boundary := d.AnalyzeFrameBoundaries(data)

	// If we have sync frames, use content-aware discard logic
	if boundary.HasSyncFrames {
		return d.discardWithSyncFramesContentAware(buffer, data, boundary)
	}

	// Fallback: use clear screen sequence (outside sync frames)
	if boundary.ClearScreenPos > 0 {
		discardLen := boundary.ClearScreenPos
		newData := make([]byte, len(data)-discardLen)
		copy(newData, data[discardLen:])
		buffer.Reset()
		buffer.Write(newData)
		logger.Terminal().Debug("FrameDetector: discarded old frames (clear screen)",
			"discarded_bytes", discardLen, "kept_bytes", len(newData))
		return discardLen
	}

	return 0
}

// isFullRedrawFrame checks if a frame contains sequences that indicate a full screen redraw.
// Full redraw frames contain ESC[2J (clear screen) or ESC[H (cursor home at start).
func (d *FrameDetector) isFullRedrawFrame(frameData []byte) bool {
	// Check for clear screen
	if bytes.Contains(frameData, eraseScreenSeq) {
		return true
	}

	// Check for cursor home at the beginning of frame content
	// (after the sync start sequence)
	frameContent := frameData
	if idx := bytes.Index(frameData, syncOutputStartSeq); idx >= 0 {
		frameContent = frameData[idx+len(syncOutputStartSeq):]
	}

	// If frame starts with cursor home, it's a full redraw
	if bytes.HasPrefix(frameContent, cursorHomeSeq) || bytes.HasPrefix(frameContent, cursorHomeSeq2) {
		return true
	}

	// Large frames (>1KB) are likely full redraws
	if len(frameData) > 1024 {
		return true
	}

	return false
}

// discardWithSyncFramesContentAware discards old frames based on content analysis.
func (d *FrameDetector) discardWithSyncFramesContentAware(buffer *bytes.Buffer, data []byte, boundary FrameBoundary) int {
	// Find all frame boundaries
	startPositions := findAllPositions(data, syncOutputStartSeq)
	endPositions := findAllPositions(data, syncOutputEndSeq)

	if len(startPositions) == 0 {
		return 0
	}

	// Find the last "full redraw" frame - we can discard everything before it
	lastFullRedrawStart := -1

	for i := len(startPositions) - 1; i >= 0; i-- {
		startPos := startPositions[i]

		// Find the corresponding end
		endPos := -1
		for _, ep := range endPositions {
			if ep > startPos {
				endPos = ep
				break
			}
		}

		if endPos < 0 {
			// This is an incomplete frame at the end - keep it
			continue
		}

		// Check if this frame is a full redraw
		frameData := data[startPos : endPos+len(syncOutputEndSeq)]
		if d.isFullRedrawFrame(frameData) {
			lastFullRedrawStart = startPos
			break
		}
	}

	// If no full redraw frame found, keep everything
	if lastFullRedrawStart <= 0 {
		return 0
	}

	// Discard everything before the last full redraw frame
	discardLen := lastFullRedrawStart
	newData := make([]byte, len(data)-discardLen)
	copy(newData, data[discardLen:])
	buffer.Reset()
	buffer.Write(newData)

	logger.Terminal().Debug("FrameDetector: discarded old frames (content-aware)",
		"discarded_bytes", discardLen, "kept_bytes", len(newData))

	return discardLen
}

// discardWithSyncFrames handles discard logic when sync output frames are present.
func (d *FrameDetector) discardWithSyncFrames(buffer *bytes.Buffer, data []byte, boundary FrameBoundary) int {
	// Determine what to keep:
	// 1. If there's an incomplete frame, keep from the PREVIOUS complete frame start (for context)
	//    or from the incomplete frame start if no previous complete frame
	// 2. If all frames are complete, keep from the last frame start

	var keepFrom int

	if boundary.IncompleteStart >= 0 {
		// There's an incomplete frame - we need to keep it
		// But we also want to keep the last complete frame before it for context
		startPositions := findAllPositions(data, syncOutputStartSeq)

		// Find the second-to-last start (last complete frame's start)
		if len(startPositions) >= 2 {
			// Keep from second-to-last start (last complete frame)
			keepFrom = startPositions[len(startPositions)-2]
		} else {
			// Only the incomplete frame's start exists, keep from there
			keepFrom = boundary.IncompleteStart
		}
	} else if boundary.CompleteEnd >= 0 {
		// All frames are complete - keep only the last frame
		// Find the start of the last complete frame
		startPositions := findAllPositions(data, syncOutputStartSeq)
		if len(startPositions) > 0 {
			keepFrom = startPositions[len(startPositions)-1]
		} else {
			return 0
		}
	} else {
		// No complete frames, keep everything
		return 0
	}

	if keepFrom <= 0 {
		return 0
	}

	discardLen := keepFrom
	newData := make([]byte, len(data)-keepFrom)
	copy(newData, data[keepFrom:])
	buffer.Reset()
	buffer.Write(newData)

	logger.Terminal().Debug("FrameDetector: discarded old frames (sync output)",
		"discarded_bytes", discardLen, "kept_bytes", len(newData),
		"incomplete_frame", boundary.IncompleteStart >= 0)

	return discardLen
}

// FindFlushBoundary determines how much data can be safely flushed.
// It ensures we don't flush in the middle of an incomplete frame.
//
// Returns:
// - flushEnd: position up to which data can be safely flushed
// - keepFrom: position from which data should be kept in buffer
//
// If there's an incomplete frame at the end, we flush up to where the
// incomplete frame starts, keeping the incomplete frame in the buffer.
func (d *FrameDetector) FindFlushBoundary(data []byte) (flushEnd, keepFrom int) {
	if len(data) == 0 {
		return 0, 0
	}

	boundary := d.AnalyzeFrameBoundaries(data)

	// If no sync frames, flush everything
	if !boundary.HasSyncFrames {
		return len(data), len(data)
	}

	// If there's an incomplete frame, don't flush it
	if boundary.IncompleteStart >= 0 {
		// Flush everything up to the incomplete frame start
		// Keep the incomplete frame in buffer
		return boundary.IncompleteStart, boundary.IncompleteStart
	}

	// All frames are complete, flush everything
	return len(data), len(data)
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
// 1. Sync frames already provide atomic updates (no need to clear first)
// 2. After resize, Claude Code sends ESC[2J + ESC[H with every frame, causing xterm.js
//    to continuously jump to top, making scrolling impossible
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
