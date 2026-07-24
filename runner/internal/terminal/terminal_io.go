package terminal

import (
	"io"
	"os"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// readOutput reads output from the PTY and sends to handler.
// Implements ttyd-style backpressure: when paused, blocks until resumed.
// This prevents unbounded memory growth when consumer can't keep up.
func (t *Terminal) readOutput() {
	defer t.signalReadDone()

	log := logger.TerminalTrace()
	label := t.label
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
			log.Warn("PTY read loop BLOCKED by backpressure", "label", label, "read_count", readCount)
			select {
			case <-t.resumeCh:
				// Resumed, continue reading
				log.Trace("PTY read loop resumed from backpressure")
			case <-time.After(100 * time.Millisecond):
				// A naturally exited process must bypass backpressure so buffered
				// tail bytes can drain before the exit callback runs.
				t.mu.Lock()
				closed := t.closed
				processExited := t.processExited
				t.mu.Unlock()
				if closed {
					return
				}
				if !processExited {
					continue // Re-check paused state
				}
			}
		}

		// Check if terminal is closed before reading
		t.mu.Lock()
		closed := t.closed
		proc := t.proc
		t.mu.Unlock()

		if closed || proc == nil {
			log.Debug("PTY read loop exiting", "label", label, "closed", closed, "proc_nil", proc == nil, "read_count", readCount)
			return
		}

		// Read and dispatch are one ordered transaction. Resize and teardown use
		// the same external boundary, so bytes cannot cross a geometry or
		// lifecycle transition after the PTY has handed them to this reader.
		var n int
		var err error
		var stopped bool
		t.withReadOrder(func() {
			t.mu.Lock()
			stopped = t.closed || t.proc != proc
			t.mu.Unlock()
			if stopped {
				return
			}

			// A short deadline lets the loop observe backpressure and shutdown.
			t.signalReadActive()
			_ = proc.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, err = proc.Read(buf)
			if n > 0 {
				readCount++
				timeoutCount = 0
				lastOutputTime = time.Now()
				data := append([]byte(nil), buf[:n]...)
				t.dispatchOutput(data, readCount)
				t.signalReadProgress()
			}
		})
		if stopped {
			return
		}

		if err != nil {
			// Check if it's just a timeout (expected during backpressure checks)
			if os.IsTimeout(err) {
				timeoutCount++
				// Log every 50 timeouts (5 seconds of no output) to track idle state
				if timeoutCount%50 == 0 {
					idleDuration := time.Since(lastOutputTime)
					log.Debug("PTY read loop idle heartbeat",
						"label", label,
						"timeout_count", timeoutCount,
						"idle_duration", idleDuration,
						"total_reads", readCount)
				}
				continue // Normal timeout, re-check pause state
			}

			if err != io.EOF {
				// Fatal PTY I/O error (not a normal close)
				t.mu.Lock()
				expectedClose := t.closed || t.processExited
				ptyErrorHandler := t.onPTYError
				t.mu.Unlock()
				if !expectedClose {
					log.Error("PTY read error", "label", label, "error", err, "read_count", readCount)

					// Notify the runner about the fatal PTY error so it can
					// send a visible error message to the frontend via relay.
					if ptyErrorHandler != nil {
						ptyErrorHandler(err)
					}

					// Kill the process to trigger clean exit through waitExit/exitHandler.
					// Without a working PTY, the user cannot interact with the process,
					// so keeping it alive would only cause a frozen terminal.
					if proc != nil {
						pid := proc.Pid()
						log.Info("Killing process after PTY read error", "label", label, "pid", pid)
						proc.Kill()
					}
				}
			} else {
				log.Debug("PTY EOF received", "label", label, "read_count", readCount)
			}
			break
		}
	}
}

// waitExit waits for the process to exit
func (t *Terminal) waitExit() {
	log := logger.Terminal()

	exitCode, err := t.proc.Wait()
	if err != nil {
		log.Error("Process wait error", "label", t.label, "error", err)
	}

	pid := t.proc.Pid()
	log.Info("Process exited", "label", t.label, "pid", pid, "exit_code", exitCode)

	t.mu.Lock()
	t.processExited = true
	t.mu.Unlock()
	// Backpressure no longer has a consumer-protection purpose after process
	// exit; release it so all already-buffered PTY output can drain promptly.
	t.ResumeRead()

	// The exit callback owns runner teardown. It must not overtake buffered PTY
	// output or an output handler that is still using the stream pipeline.
	t.drainReadOutput()
	t.closePTY()

	t.mu.Lock()
	t.closed = true
	handler := t.onExit
	t.mu.Unlock()

	// Signal only after the reader is joined, so Stop cannot return while an
	// output callback is still running.
	close(t.doneCh)

	if handler != nil {
		handler(exitCode)
	}
}
