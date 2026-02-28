package terminal

import (
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// readOutput reads output from the PTY and sends to handler.
// Implements ttyd-style backpressure: when paused, blocks until resumed.
// This prevents unbounded memory growth when consumer can't keep up.
func (t *Terminal) readOutput() {
	log := logger.TerminalTrace()
	buf := make([]byte, 4096)
	readCount := 0
	timeoutCount := 0            // Track consecutive timeouts
	lastOutputTime := time.Now() // Track when we last received output

	for {
		// Check if we should pause (backpressure from consumer)
		t.readPauseMu.RLock()
		paused := t.readPaused
		t.readPauseMu.RUnlock()

		if paused {
			// Block until resume signal or terminal closes
			// This is the key to ttyd-style backpressure:
			// we stop reading from PTY when consumer is overwhelmed
			log.Warn("PTY read loop BLOCKED by backpressure", "read_count", readCount)
			select {
			case <-t.resumeCh:
				// Resumed, continue reading
				log.Trace("PTY read loop resumed from backpressure")
			case <-time.After(100 * time.Millisecond):
				// Periodic check - verify terminal isn't closed
				t.mu.Lock()
				closed := t.closed
				t.mu.Unlock()
				if closed {
					return
				}
				continue // Re-check paused state
			}
		}

		// Check if terminal is closed before reading
		t.mu.Lock()
		closed := t.closed
		ptyFile := t.pty
		t.mu.Unlock()

		if closed || ptyFile == nil {
			log.Debug("PTY read loop exiting", "closed", closed, "ptyFile_nil", ptyFile == nil, "read_count", readCount)
			return
		}

		// Read from PTY with timeout to allow periodic backpressure checks
		// This ensures we can respond to pause signals even during slow output
		ptyFile.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, err := ptyFile.Read(buf)

		if err != nil {
			// Check if it's just a timeout (expected during backpressure checks)
			if os.IsTimeout(err) {
				timeoutCount++
				// Log every 50 timeouts (5 seconds of no output) to track idle state
				if timeoutCount%50 == 0 {
					idleDuration := time.Since(lastOutputTime)
					log.Debug("PTY read loop idle heartbeat",
						"timeout_count", timeoutCount,
						"idle_duration", idleDuration,
						"total_reads", readCount)
				}
				continue // Normal timeout, re-check pause state
			}

			if err != io.EOF {
				// Fatal PTY I/O error (not a normal close)
				t.mu.Lock()
				closed := t.closed
				ptyErrorHandler := t.onPTYError
				t.mu.Unlock()
				if !closed {
					log.Error("PTY read error", "error", err, "read_count", readCount)

					// Notify the runner about the fatal PTY error so it can
					// send a visible error message to the frontend via relay.
					if ptyErrorHandler != nil {
						ptyErrorHandler(err)
					}

					// Kill the process to trigger clean exit through waitExit/exitHandler.
					// Without a working PTY, the user cannot interact with the process,
					// so keeping it alive would only cause a frozen terminal.
					if t.cmd != nil && t.cmd.Process != nil {
						log.Info("Killing process after PTY read error", "pid", t.cmd.Process.Pid)
						t.cmd.Process.Kill()
					}
				}
			} else {
				log.Debug("PTY EOF received", "read_count", readCount)
			}
			break
		}

		readCount++
		timeoutCount = 0            // Reset timeout counter on successful read
		lastOutputTime = time.Now() // Update last output time
		if n > 0 {
			// Log every read for debugging (Trace level - high frequency)
			log.Trace("PTY read SUCCESS",
				"read_num", readCount,
				"bytes", n)

			// Make a copy of the data
			data := make([]byte, n)
			copy(data, buf[:n])

			// Get handler with lock to prevent race condition
			t.mu.Lock()
			handler := t.onOutput
			t.mu.Unlock()

			if handler != nil {
				log.Trace("PTY calling handler",
					"read_num", readCount,
					"bytes", n)
				startHandler := time.Now()
				handler(data)
				handlerTime := time.Since(startHandler)
				log.Trace("PTY handler returned",
					"read_num", readCount,
					"bytes", n,
					"handler_time", handlerTime)
				if handlerTime > 50*time.Millisecond {
					log.Warn("PTY output handler slow", "read_num", readCount, "bytes", n, "handler_time", handlerTime)
				}
			} else {
				log.Warn("No output handler set", "read_num", readCount)
			}
		}
	}
}

// waitExit waits for the process to exit
func (t *Terminal) waitExit() {
	log := logger.Terminal()
	exitCode := 0
	if err := t.cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	log.Info("Process exited", "pid", t.cmd.Process.Pid, "exit_code", exitCode)

	// Signal that the process has exited (unblocks Stop() if waiting)
	close(t.doneCh)

	t.mu.Lock()
	t.closed = true
	t.mu.Unlock()

	// Close PTY via sync.Once (safe if Stop() also calls closePTY)
	t.closePTY()

	// Get handler with lock to prevent race condition
	t.mu.Lock()
	handler := t.onExit
	t.mu.Unlock()

	if handler != nil {
		handler(exitCode)
	}
}
