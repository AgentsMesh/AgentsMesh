package aggregator

import (
	"bytes"
	"unicode/utf8"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// FrameBuffer preserves an exact, bounded prefix of the raw terminal stream.
type FrameBuffer struct {
	buffer   bytes.Buffer
	maxSize  int
	detector *FrameDetector
}

// NewFrameBuffer creates a new frame buffer.
//
// Parameters:
//   - maxSize: bulk buffer cap; an incomplete UTF-8 boundary may use up to three
//     additional bytes so a later chunk can complete the owned rune
func NewFrameBuffer(maxSize int) *FrameBuffer {
	return &FrameBuffer{
		maxSize:  maxSize,
		detector: NewFrameDetector(),
	}
}

// Write appends the complete chunk or clears all recoverable delta bytes on
// overflow. A trailing incomplete UTF-8 rune is retained because its suffix is
// owned by a later PTY chunk. The buffer may use at most utf8.UTFMax-1 bytes of
// boundary headroom beyond maxSize to complete that rune.
//
// False means the caller must recover every downstream stream from an
// authoritative snapshot; no suffix of a truncated ANSI stream is usable.
func (b *FrameBuffer) Write(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	capacity := b.maxSize
	if findLastValidUTF8Boundary(b.buffer.Bytes()) < b.buffer.Len() {
		capacity += utf8.UTFMax - 1
	}
	if b.maxSize <= 0 || len(data) > capacity-b.buffer.Len() {
		continuation := trailingIncompleteUTF8(b.buffer.Bytes(), data)
		b.buffer.Reset()
		b.buffer.Write(continuation)
		return false
	}
	b.buffer.Write(data)
	return true
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
