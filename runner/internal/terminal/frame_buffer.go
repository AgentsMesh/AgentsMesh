// Package terminal provides terminal management for PTY sessions.
package terminal

import (
	"bytes"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// FrameBuffer manages terminal output buffering with frame-aware operations.
// It uses FrameDetector to ensure frame integrity during discard and flush operations.
type FrameBuffer struct {
	buffer   bytes.Buffer
	maxSize  int
	detector *FrameDetector
}

// NewFrameBuffer creates a new frame buffer.
//
// Parameters:
// - maxSize: maximum buffer size (hard cap to prevent unbounded memory growth)
func NewFrameBuffer(maxSize int) *FrameBuffer {
	return &FrameBuffer{
		maxSize:  maxSize,
		detector: NewFrameDetector(),
	}
}

// Write adds data to the buffer.
// Uses content-aware discard strategy and enforces size limits.
//
// The discard strategy is intelligent:
// - Full redraw frames (contain ESC[2J or ESC[H): discard everything before them
// - Incremental frames (small, relative cursor movement): keep them all
//
// This is critical for Claude Code which uses both patterns.
func (b *FrameBuffer) Write(data []byte) {
	if len(data) == 0 {
		return
	}

	// Enforce buffer size limit before adding new data
	b.enforceLimit(len(data))

	b.buffer.Write(data)

	// Content-aware discard: only discard if there's a full redraw frame
	// Incremental frames are preserved
	b.detector.DiscardOldFrames(&b.buffer)

	// Enforce limit again after write (handles case where data itself exceeds limit)
	b.enforceLimitAfterWrite()
}

// FlushComplete returns data that can be safely flushed (complete frames only).
// Incomplete frames are kept in the buffer for next flush.
//
// Returns:
// - data: bytes to be flushed
// - remaining: bytes kept in buffer
func (b *FrameBuffer) FlushComplete() (data []byte, remaining int) {
	if b.buffer.Len() == 0 {
		return nil, 0
	}

	allData := b.buffer.Bytes()

	// Find flush boundary (don't flush incomplete frames)
	flushEnd, keepFrom := b.detector.FindFlushBoundary(allData)

	// Also ensure we don't break UTF-8 characters
	if flushEnd > 0 {
		adjustedFlushEnd := findLastValidUTF8Boundary(allData[:flushEnd])
		// IMPORTANT: When flushEnd is adjusted backwards for UTF-8 boundary,
		// we must also adjust keepFrom to avoid losing data.
		// The bytes between adjustedFlushEnd and original flushEnd must be kept.
		if adjustedFlushEnd < flushEnd && adjustedFlushEnd < keepFrom {
			keepFrom = adjustedFlushEnd
		}
		flushEnd = adjustedFlushEnd
	}

	if flushEnd == 0 {
		// Nothing to flush (only incomplete frame or incomplete UTF-8)
		return nil, b.buffer.Len()
	}

	// Copy data to flush
	data = make([]byte, flushEnd)
	copy(data, allData[:flushEnd])

	// Strip redundant sequences (ESC[2J, ESC[H) from inside sync frames
	// This prevents xterm.js from jumping to top after resize
	data = b.detector.StripRedundantSequencesInFrames(data)

	// Keep remaining data in buffer
	if keepFrom < len(allData) {
		remainingData := make([]byte, len(allData)-keepFrom)
		copy(remainingData, allData[keepFrom:])
		b.buffer.Reset()
		b.buffer.Write(remainingData)
		logger.TerminalTrace().Trace("FrameBuffer: keeping incomplete data",
			"flushed", flushEnd, "remaining", len(remainingData))
	} else {
		b.buffer.Reset()
	}

	return data, b.buffer.Len()
}

// FlushAll returns all buffered data, handling UTF-8 boundaries.
// Use this for forced flushes (like Stop).
//
// Returns:
// - data: bytes to be flushed
// - remaining: bytes kept in buffer (incomplete UTF-8 only)
func (b *FrameBuffer) FlushAll() (data []byte, remaining int) {
	if b.buffer.Len() == 0 {
		return nil, 0
	}

	allData := b.buffer.Bytes()

	// Find last valid UTF-8 boundary
	validLen := findLastValidUTF8Boundary(allData)

	if validLen == 0 {
		return nil, b.buffer.Len()
	}

	// Copy valid data for sending
	data = make([]byte, validLen)
	copy(data, allData[:validLen])

	// Strip redundant sequences (ESC[2J, ESC[H) from inside sync frames
	// This prevents xterm.js from jumping to top after resize
	data = b.detector.StripRedundantSequencesInFrames(data)

	// Keep any trailing incomplete UTF-8 bytes
	if validLen < len(allData) {
		remainingData := make([]byte, len(allData)-validLen)
		copy(remainingData, allData[validLen:])
		b.buffer.Reset()
		b.buffer.Write(remainingData)
		logger.TerminalTrace().Trace("FrameBuffer: keeping incomplete UTF-8",
			"flushed", validLen, "remaining", len(remainingData))
	} else {
		b.buffer.Reset()
	}

	return data, b.buffer.Len()
}

// Len returns current buffer length.
func (b *FrameBuffer) Len() int {
	return b.buffer.Len()
}

// Reset clears the buffer.
func (b *FrameBuffer) Reset() {
	b.buffer.Reset()
}

// Bytes returns the current buffer contents (for testing/debugging).
func (b *FrameBuffer) Bytes() []byte {
	return b.buffer.Bytes()
}

// MaxSize returns the configured max buffer size.
func (b *FrameBuffer) MaxSize() int {
	return b.maxSize
}

// SetMaxSize updates the max buffer size.
func (b *FrameBuffer) SetMaxSize(size int) {
	b.maxSize = size
}

// IsLastFrameFullRedraw checks if the last complete frame in the buffer is a full-screen redraw.
// This is used by FullRedrawThrottler to detect high-frequency redraw patterns.
//
// Returns true if:
//   - The buffer contains sync frames (ESC[?2026h ... ESC[?2026l)
//   - The last complete frame is a full redraw (contains ESC[2J, starts with ESC[H, or is large)
//
// Returns false if:
//   - Buffer is empty
//   - No sync frames in buffer
//   - Last frame is not a full redraw (e.g., incremental update)
func (b *FrameBuffer) IsLastFrameFullRedraw() bool {
	data := b.buffer.Bytes()
	if len(data) == 0 {
		return false
	}

	boundary := b.detector.AnalyzeFrameBoundaries(data)
	if !boundary.HasSyncFrames {
		return false
	}

	// Find all frame boundaries
	startPositions := findAllPositions(data, syncOutputStartSeq)
	endPositions := findAllPositions(data, syncOutputEndSeq)

	if len(startPositions) == 0 {
		return false
	}

	// Find the last complete frame (match starts with ends)
	var lastCompleteFrameStart int = -1
	var lastCompleteFrameEnd int = -1
	usedEnds := make(map[int]bool)

	for _, startPos := range startPositions {
		for _, endPos := range endPositions {
			if endPos > startPos && !usedEnds[endPos] {
				usedEnds[endPos] = true
				lastCompleteFrameStart = startPos
				lastCompleteFrameEnd = endPos + len(syncOutputEndSeq)
				break
			}
		}
	}

	if lastCompleteFrameStart < 0 || lastCompleteFrameEnd <= lastCompleteFrameStart {
		// No complete frame found
		return false
	}

	// Check if this frame is a full redraw
	frameData := data[lastCompleteFrameStart:lastCompleteFrameEnd]
	return b.detector.IsFullRedrawFrame(frameData)
}

// Note: enforceLimit and enforceLimitAfterWrite are in frame_buffer_limit.go
