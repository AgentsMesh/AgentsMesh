package terminal

import (
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// Resize resizes the terminal.
// Parameters follow the standard convention: cols (width) first, then rows (height).
// This matches xterm.js, ANSI standards, and most terminal libraries.
func (t *Terminal) Resize(cols, rows int) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed || t.stopping || t.proc == nil {
		return fmt.Errorf("terminal is not running")
	}
	if cols == t.cols && rows == t.rows {
		return nil
	}

	logger.Terminal().Debug("Terminal resize", "cols", cols, "rows", rows)

	if err := t.proc.Resize(cols, rows); err != nil {
		return err
	}
	t.cols = cols
	t.rows = rows
	return nil
}

// PauseRead pauses PTY reading (backpressure signal from consumer).
// This implements ttyd-style flow control: when the consumer can't keep up,
// we stop reading from the PTY to prevent unbounded memory growth.
// The readOutput goroutine will block until ResumeRead is called.
func (t *Terminal) PauseRead() {
	t.readPauseMu.Lock()
	wasPaused := t.readPaused
	t.readPaused = true
	t.readPauseMu.Unlock()

	if !wasPaused {
		logger.TerminalTrace().Trace("PTY read paused (backpressure)")
	}
}

// ResumeRead resumes PTY reading after backpressure is released.
// This signals the readOutput goroutine to continue reading.
func (t *Terminal) ResumeRead() {
	t.readPauseMu.Lock()
	wasPaused := t.readPaused
	t.readPaused = false
	t.readPauseMu.Unlock()

	if wasPaused {
		// Signal the resume channel (non-blocking)
		select {
		case t.resumeCh <- struct{}{}:
		default:
			// Channel already has a signal pending
		}
		logger.TerminalTrace().Trace("PTY read resumed")
	}
}

// IsReadPaused returns whether PTY reading is currently paused.
func (t *Terminal) IsReadPaused() bool {
	t.readPauseMu.RLock()
	defer t.readPauseMu.RUnlock()
	return t.readPaused
}
