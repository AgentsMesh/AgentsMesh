// Package terminal provides terminal management for PTY sessions.
package aggregator

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
	}

	if len(data) == 0 {
		return result
	}

	// Find all start and end positions
	startPositions := findAllPositions(data, syncOutputStartSeq)
	endPositions := findAllPositions(data, syncOutputEndSeq)

	result.HasSyncFrames = len(startPositions) > 0 || len(endPositions) > 0

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
