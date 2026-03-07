package terminal

import (
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// Resize resizes the terminal.
// Parameters follow the standard convention: cols (width) first, then rows (height).
// This matches xterm.js, ANSI standards, and most terminal libraries.
func (t *Terminal) Resize(cols, rows int) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed || t.proc == nil {
		return fmt.Errorf("terminal is not running")
	}

	logger.Terminal().Debug("Terminal resize", "cols", cols, "rows", rows)

	return t.proc.Resize(cols, rows)
}

// Redraw triggers a terminal redraw by temporarily changing the terminal size.
// This is used to restore terminal state after server restart.
// We use resize +1/-1 instead of just SIGWINCH because some programs (like Claude Code)
// don't respond to SIGWINCH when they're in an idle/waiting state.
func (t *Terminal) Redraw() error {
	t.mu.Lock()
	if t.closed || t.proc == nil {
		t.mu.Unlock()
		return fmt.Errorf("terminal is not running")
	}

	// Get current size
	cols, rows, err := t.proc.GetSize()
	if err != nil {
		t.mu.Unlock()
		return fmt.Errorf("failed to get terminal size: %w", err)
	}

	// Resize to cols+1 to trigger redraw
	if err := t.proc.Resize(cols+1, rows); err != nil {
		t.mu.Unlock()
		return fmt.Errorf("failed to expand terminal: %w", err)
	}
	t.mu.Unlock()

	// Small delay to ensure the resize is processed — lock released so
	// other Terminal operations (Write, Resize) are not blocked.
	time.Sleep(50 * time.Millisecond)

	// Resize back to original size
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed || t.proc == nil {
		return fmt.Errorf("terminal closed during redraw")
	}

	if err := t.proc.Resize(cols, rows); err != nil {
		return fmt.Errorf("failed to restore terminal size: %w", err)
	}

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
