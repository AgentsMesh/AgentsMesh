package runner

import (
	"github.com/anthropics/agentsmesh/backend/internal/infra/terminal"
)

// GetRecentOutput returns recent terminal output for observation
// If raw is true, returns raw scrollback data; otherwise returns processed output from virtual terminal
func (tr *TerminalRouter) GetRecentOutput(podKey string, lines int, raw bool) []byte {
	shard := tr.getShard(podKey)

	shard.mu.RLock()
	buffer := shard.scrollbackBuffers[podKey]
	vt := shard.virtualTerminals[podKey]
	shard.mu.RUnlock()

	if raw {
		// Return raw scrollback data
		if buffer == nil {
			return nil
		}
		return buffer.GetRecentLines(lines)
	}

	// Try to return processed output from virtual terminal
	if vt != nil {
		output := vt.GetOutput(lines)
		if output != "" {
			return []byte(output)
		}
	}

	// Fallback: if virtual terminal has no data, strip ANSI from raw scrollback
	if buffer == nil {
		return nil
	}

	rawData := buffer.GetRecentLines(lines)
	if rawData == nil {
		return nil
	}

	// Strip ANSI escape sequences as fallback
	return []byte(terminal.StripANSI(string(rawData)))
}

// GetScreenSnapshot returns the current screen snapshot for agent observation
func (tr *TerminalRouter) GetScreenSnapshot(podKey string) string {
	shard := tr.getShard(podKey)

	shard.mu.RLock()
	vt := shard.virtualTerminals[podKey]
	buffer := shard.scrollbackBuffers[podKey]
	shard.mu.RUnlock()

	if vt != nil {
		display := vt.GetDisplay()
		if display != "" {
			return display
		}
	}

	// Fallback: strip ANSI from raw scrollback and return last screen worth of lines
	if buffer == nil {
		return ""
	}

	// Get approximately one screen worth of lines (default 24 lines)
	rawData := buffer.GetRecentLines(DefaultTerminalRows)
	if rawData == nil {
		return ""
	}

	return terminal.StripANSI(string(rawData))
}

// GetCursorPosition returns the current cursor position (row, col) for a pod
func (tr *TerminalRouter) GetCursorPosition(podKey string) (row, col int) {
	shard := tr.getShard(podKey)

	shard.mu.RLock()
	vt := shard.virtualTerminals[podKey]
	shard.mu.RUnlock()

	if vt == nil {
		return 0, 0
	}
	return vt.CursorPosition()
}

// GetAllScrollbackData returns all scrollback buffer data
func (tr *TerminalRouter) GetAllScrollbackData(podKey string) []byte {
	shard := tr.getShard(podKey)

	shard.mu.RLock()
	buffer := shard.scrollbackBuffers[podKey]
	shard.mu.RUnlock()

	if buffer == nil {
		return nil
	}

	return buffer.GetData()
}

// ClearScrollback clears the scrollback buffer for a pod
func (tr *TerminalRouter) ClearScrollback(podKey string) {
	shard := tr.getShard(podKey)

	shard.mu.RLock()
	buffer := shard.scrollbackBuffers[podKey]
	shard.mu.RUnlock()

	if buffer != nil {
		buffer.Clear()
	}
}
