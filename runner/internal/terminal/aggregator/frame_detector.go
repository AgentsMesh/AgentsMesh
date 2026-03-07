// Package terminal provides terminal management for PTY sessions.
package aggregator

import (
	"bytes"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
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
	var lastCompleteEnd = -1
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
		logger.TerminalTrace().Trace("FrameDetector: discarded old frames (clear screen)",
			"discarded_bytes", discardLen, "kept_bytes", len(newData))
		return discardLen
	}

	return 0
}

// IsFullRedrawFrame checks if a frame contains sequences that indicate a full screen redraw.
// Full redraw frames contain ESC[2J (clear screen) or ESC[H (cursor home at start).
//
// Detection criteria:
//   - Contains ESC[2J (clear entire screen)
//   - Starts with ESC[H or ESC[;H (cursor home at beginning of frame content)
//   - Frame size > 1KB (large frames are typically full redraws)
//
// This is used by:
//   - DiscardOldFrames: to determine which frames can be safely discarded
//   - FullRedrawThrottler: to detect high-frequency redraw patterns
func (d *FrameDetector) IsFullRedrawFrame(frameData []byte) bool {
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
		if d.IsFullRedrawFrame(frameData) {
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

	logger.TerminalTrace().Trace("FrameDetector: discarded old frames (content-aware)",
		"discarded_bytes", discardLen, "kept_bytes", len(newData))

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
