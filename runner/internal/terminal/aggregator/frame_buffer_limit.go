// Package terminal provides terminal management for PTY sessions.
package aggregator

import "github.com/AgentsMesh/AgentsMesh/runner/internal/logger"

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
