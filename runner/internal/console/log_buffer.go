// Package console provides a local web console for managing the runner.
package console

import (
	"sync"
	"time"
)

// LogEntry represents a single log entry.
type LogEntry struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
}

// LogBuffer is a circular buffer for log entries.
type LogBuffer struct {
	entries []LogEntry
	maxSize int
	mu      sync.RWMutex
}

// NewLogBuffer creates a new log buffer.
func NewLogBuffer(maxSize int) *LogBuffer {
	return &LogBuffer{
		entries: make([]LogEntry, 0, maxSize),
		maxSize: maxSize,
	}
}

// Add adds a log entry to the buffer.
func (b *LogBuffer) Add(level, message string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	entry := LogEntry{
		Time:    time.Now(),
		Level:   level,
		Message: message,
	}

	if len(b.entries) >= b.maxSize {
		// Remove oldest entry
		b.entries = b.entries[1:]
	}
	b.entries = append(b.entries, entry)
}

// GetAll returns all log entries.
func (b *LogBuffer) GetAll() []LogEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]LogEntry, len(b.entries))
	copy(result, b.entries)
	return result
}

// GetRecent returns the most recent n entries.
func (b *LogBuffer) GetRecent(n int) []LogEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if n >= len(b.entries) {
		result := make([]LogEntry, len(b.entries))
		copy(result, b.entries)
		return result
	}

	start := len(b.entries) - n
	result := make([]LogEntry, n)
	copy(result, b.entries[start:])
	return result
}
