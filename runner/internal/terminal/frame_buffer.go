// Package terminal provides terminal management for PTY sessions.
package terminal

import (
	"bytes"
	"unicode/utf8"

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
		logger.Terminal().Debug("FrameBuffer: keeping incomplete data",
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
		logger.Terminal().Debug("FrameBuffer: keeping incomplete UTF-8",
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

// enforceLimit ensures buffer doesn't exceed maxSize after adding newDataLen bytes.
func (b *FrameBuffer) enforceLimit(newDataLen int) {
	targetLen := b.buffer.Len() + newDataLen
	if targetLen <= b.maxSize {
		return
	}

	// First try to discard old frames
	b.detector.DiscardOldFrames(&b.buffer)

	// Check again after discarding frames
	targetLen = b.buffer.Len() + newDataLen
	if targetLen <= b.maxSize {
		return
	}

	// Still over limit - discard oldest data from head
	excess := targetLen - b.maxSize
	if excess > 0 && excess < b.buffer.Len() {
		data := b.buffer.Bytes()
		// Adjust offset to UTF-8 character boundary
		offset := alignToUTF8Boundary(data, excess)
		newData := make([]byte, len(data)-offset)
		copy(newData, data[offset:])
		b.buffer.Reset()
		b.buffer.Write(newData)
	} else if excess >= b.buffer.Len() {
		// New data alone exceeds limit - clear buffer entirely
		b.buffer.Reset()
	}
}

// enforceLimitAfterWrite truncates buffer if it exceeds maxSize after a write.
// This handles the case where the written data itself exceeds the limit.
//
// IMPORTANT: We must align truncation to frame boundaries, not just UTF-8 boundaries.
// If we truncate in the middle of a frame (after [?2026h but before [?2026l]),
// we'll have orphan frame ends that break the TUI rendering.
func (b *FrameBuffer) enforceLimitAfterWrite() {
	if b.buffer.Len() <= b.maxSize {
		return
	}

	data := b.buffer.Bytes()
	excess := b.buffer.Len() - b.maxSize

	// First, try to find a safe truncation point that respects frame boundaries.
	// A safe point is either:
	// 1. Right after a complete frame end (ESC[?2026l)
	// 2. At a position where there's no incomplete frame before it
	boundary := b.detector.AnalyzeFrameBoundaries(data)

	if boundary.HasSyncFrames {
		// If there are sync frames, we need to be careful
		// Find all frame starts and ends to determine safe truncation points
		startPositions := findAllPositions(data, syncOutputStartSeq)
		endPositions := findAllPositions(data, syncOutputEndSeq)

		// Find the first frame start that would remain after truncation
		// We need to ensure we don't truncate to a position that's inside a frame
		safeOffset := -1
		for _, startPos := range startPositions {
			if startPos >= excess {
				// This frame start would remain - check if it's a good truncation point
				// Verify there's no unclosed frame before this position
				unclosedBefore := false
				for _, s := range startPositions {
					if s < startPos {
						// Check if this earlier start has a matching end before our truncation point
						hasEnd := false
						for _, e := range endPositions {
							if e > s && e < startPos {
								hasEnd = true
								break
							}
						}
						if !hasEnd {
							unclosedBefore = true
							break
						}
					}
				}
				if !unclosedBefore {
					safeOffset = startPos
					break
				}
			}
		}

		// If we found a safe frame-aligned offset, use it
		if safeOffset >= 0 {
			// Also align to UTF-8 boundary (should already be, but be safe)
			offset := alignToUTF8Boundary(data, safeOffset)
			if offset > 0 && offset < len(data) {
				newData := make([]byte, len(data)-offset)
				copy(newData, data[offset:])
				b.buffer.Reset()
				b.buffer.Write(newData)
				logger.Terminal().Debug("FrameBuffer: truncated at frame boundary",
					"excess", excess, "actual_offset", offset, "new_len", len(newData))
				return
			}
		}

		// No safe frame boundary found - we're inside a large frame
		// In this case, we need to truncate from the END to preserve frame START.
		// The frame START (ESC[?2026h) is critical - it tells the terminal to enter
		// synchronized output mode. Without it, the frame END becomes orphan.
		//
		// Strategy: Keep maxSize bytes from the START of the data (preserving frame start)
		if len(data) > b.maxSize {
			// Truncate from end, but try to end at a frame boundary or UTF-8 boundary
			truncateAt := b.maxSize

			// Find the last complete frame end within maxSize
			for _, endPos := range endPositions {
				completeEndPos := endPos + len(syncOutputEndSeq)
				if completeEndPos <= b.maxSize {
					truncateAt = completeEndPos
				}
			}

			// Align to UTF-8 boundary
			truncateAt = findLastValidUTF8Boundary(data[:truncateAt])

			if truncateAt > 0 && truncateAt < len(data) {
				newData := make([]byte, truncateAt)
				copy(newData, data[:truncateAt])
				b.buffer.Reset()
				b.buffer.Write(newData)
				logger.Terminal().Debug("FrameBuffer: truncated from end to preserve frame start",
					"original_len", len(data), "new_len", truncateAt, "max_size", b.maxSize)
				return
			}
		}

		logger.Terminal().Warn("FrameBuffer: truncating inside frame (no safe boundary found)",
			"excess", excess, "buffer_len", len(data), "max_size", b.maxSize)
	}

	// Fallback: truncate at UTF-8 boundary only
	offset := alignToUTF8Boundary(data, excess)
	newData := make([]byte, len(data)-offset)
	copy(newData, data[offset:])
	b.buffer.Reset()
	b.buffer.Write(newData)
}

// alignToUTF8Boundary adjusts an offset to the next valid UTF-8 character boundary.
// This prevents truncating in the middle of a multi-byte UTF-8 character.
func alignToUTF8Boundary(data []byte, offset int) int {
	if offset >= len(data) {
		return len(data)
	}
	// If we're at the start of a valid UTF-8 character, we're done
	if utf8.RuneStart(data[offset]) {
		return offset
	}
	// Otherwise, advance until we find the start of a valid UTF-8 character
	for offset < len(data) && !utf8.RuneStart(data[offset]) {
		offset++
	}
	return offset
}

// findLastValidUTF8Boundary finds the last position in data that ends on a valid UTF-8 boundary.
// This is used to avoid sending incomplete multi-byte characters at the end of a message.
func findLastValidUTF8Boundary(data []byte) int {
	if len(data) == 0 {
		return 0
	}

	// Check if data already ends on a valid UTF-8 boundary
	for i := len(data) - 1; i >= 0 && i >= len(data)-4; i-- {
		if utf8.RuneStart(data[i]) {
			// Found the start of a UTF-8 character
			// Check if the remaining bytes form a complete character
			r, size := utf8.DecodeRune(data[i:])
			if r != utf8.RuneError || size == len(data)-i {
				// Complete character or valid single byte
				return len(data)
			}
			// Incomplete character - truncate before it
			return i
		}
	}

	// All bytes in the last 4 positions are continuation bytes
	for i := len(data) - 1; i >= 0; i-- {
		if utf8.RuneStart(data[i]) {
			return i
		}
	}

	// No valid UTF-8 start byte found - return all data
	return len(data)
}
