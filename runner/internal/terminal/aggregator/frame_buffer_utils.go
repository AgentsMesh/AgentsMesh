package aggregator

import "unicode/utf8"

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
			// FullRune distinguishes an incomplete lead byte from an invalid byte.
			// Invalid bytes remain part of the raw terminal stream, while an incomplete
			// rune stays owned by the next PTY chunk.
			if !utf8.FullRune(data[i:]) {
				return i
			}
			return len(data)
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

// trailingIncompleteUTF8 returns only the incomplete rune prefix at the end of
// the concatenated stream. A UTF-8 rune can own at most three pending bytes, so
// inspecting the final UTFMax-1 bytes is sufficient without copying the stream.
func trailingIncompleteUTF8(buffer, data []byte) []byte {
	const maxPending = utf8.UTFMax - 1
	tail := make([]byte, 0, maxPending)
	if len(data) < maxPending {
		needed := maxPending - len(data)
		start := len(buffer) - needed
		if start < 0 {
			start = 0
		}
		tail = append(tail, buffer[start:]...)
	}
	start := len(data) - maxPending
	if start < 0 {
		start = 0
	}
	tail = append(tail, data[start:]...)

	boundary := findLastValidUTF8Boundary(tail)
	if boundary == len(tail) {
		return nil
	}
	return append([]byte(nil), tail[boundary:]...)
}
