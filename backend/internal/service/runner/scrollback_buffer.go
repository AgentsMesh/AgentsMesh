package runner

import (
	"bytes"
	"sync"
	"unicode/utf8"
)

// ScrollbackBuffer stores terminal output for reconnection
type ScrollbackBuffer struct {
	data    []byte
	maxSize int
	mu      sync.RWMutex
}

// NewScrollbackBuffer creates a new scrollback buffer
func NewScrollbackBuffer(maxSize int) *ScrollbackBuffer {
	return &ScrollbackBuffer{
		data:    make([]byte, 0, maxSize),
		maxSize: maxSize,
	}
}

// Write appends data to the buffer, trimming old data if necessary
func (sb *ScrollbackBuffer) Write(data []byte) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	sb.data = append(sb.data, data...)

	// Trim if exceeded max size
	if len(sb.data) > sb.maxSize {
		// Keep only the last maxSize bytes
		sb.data = sb.data[len(sb.data)-sb.maxSize:]
		// Ensure we start at a valid UTF-8 boundary
		sb.data = trimToValidUTF8Start(sb.data)
	}
}

// trimToValidUTF8Start ensures data starts with a valid UTF-8 character.
// If the data begins with continuation bytes (10xxxxxx pattern), it skips them
// to find the start of a valid UTF-8 sequence.
func trimToValidUTF8Start(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	// Check up to utf8.UTFMax (4) bytes for a valid start
	for i := 0; i < len(data) && i < utf8.UTFMax; i++ {
		// Check if remaining data is valid UTF-8
		if utf8.Valid(data[i:]) {
			return data[i:]
		}
		// Also check if this byte starts a valid UTF-8 sequence
		// (not a continuation byte: 10xxxxxx)
		if data[i]&0xC0 != 0x80 {
			// This is a leading byte, check if the sequence starting here is valid
			if r, _ := utf8.DecodeRune(data[i:]); r != utf8.RuneError {
				return data[i:]
			}
		}
	}

	// Fallback: return original data (shouldn't normally reach here)
	return data
}

// GetData returns a copy of the buffer data
func (sb *ScrollbackBuffer) GetData() []byte {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	result := make([]byte, len(sb.data))
	copy(result, sb.data)
	return result
}

// GetRecentLines returns the last N lines from the buffer
func (sb *ScrollbackBuffer) GetRecentLines(lines int) []byte {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	if len(sb.data) == 0 {
		return nil
	}

	// Split by newlines and return last N lines
	allLines := bytes.Split(sb.data, []byte("\n"))
	if len(allLines) <= lines {
		return sb.data
	}

	recentLines := allLines[len(allLines)-lines:]
	return bytes.Join(recentLines, []byte("\n"))
}

// Clear clears the buffer
func (sb *ScrollbackBuffer) Clear() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.data = sb.data[:0]
}
