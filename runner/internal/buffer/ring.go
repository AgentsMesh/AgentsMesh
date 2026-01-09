// Package buffer provides buffer utilities.
package buffer

import (
	"sync"
	"unicode/utf8"
)

// Ring is a circular buffer for storing recent data.
// It is thread-safe and efficiently stores the most recent N bytes,
// discarding older data as new data is written.
type Ring struct {
	data  []byte
	size  int
	start int
	end   int
	full  bool
	mu    sync.Mutex
}

// NewRing creates a new ring buffer with the specified size.
func NewRing(size int) *Ring {
	return &Ring{
		data: make([]byte, size),
		size: size,
	}
}

// Write writes data to the ring buffer.
// Implements io.Writer interface.
func (rb *Ring) Write(p []byte) (n int, err error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	for _, b := range p {
		rb.data[rb.end] = b
		rb.end = (rb.end + 1) % rb.size
		if rb.full {
			rb.start = (rb.start + 1) % rb.size
		}
		if rb.end == rb.start {
			rb.full = true
		}
	}
	return len(p), nil
}

// Bytes returns all data in the buffer.
// The returned slice is a copy of the internal data.
// When the buffer wraps, the result is aligned to UTF-8 character boundaries.
func (rb *Ring) Bytes() []byte {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if !rb.full && rb.start == rb.end {
		return nil
	}

	if rb.full {
		result := make([]byte, rb.size)
		copy(result, rb.data[rb.start:])
		copy(result[rb.size-rb.start:], rb.data[:rb.end])
		// Ensure result starts at a valid UTF-8 boundary
		return trimToValidUTF8Start(result)
	}

	if rb.end > rb.start {
		result := make([]byte, rb.end-rb.start)
		copy(result, rb.data[rb.start:rb.end])
		return result
	}

	return nil
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

// Reset clears the buffer.
func (rb *Ring) Reset() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.start = 0
	rb.end = 0
	rb.full = false
}

// Len returns the current number of bytes in the buffer.
func (rb *Ring) Len() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.full {
		return rb.size
	}
	if rb.end >= rb.start {
		return rb.end - rb.start
	}
	return rb.size - rb.start + rb.end
}

// Cap returns the capacity of the buffer.
func (rb *Ring) Cap() int {
	return rb.size
}
