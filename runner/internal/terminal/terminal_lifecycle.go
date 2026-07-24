package terminal

import (
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// Stop gracefully terminates the process and joins the PTY reader.
func (t *Terminal) Stop() {
	t.lifecycleMu.Lock()
	defer t.lifecycleMu.Unlock()

	log := logger.Terminal()
	log.Info("Terminal stopping")

	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		t.waitForReadDone()
		return
	}
	t.stopping = true
	proc := t.proc
	if proc == nil {
		t.closed = true
	}
	t.mu.Unlock()

	if proc != nil {
		pid := proc.Pid()
		if err := proc.GracefulStop(); err != nil {
			log.Debug("Graceful stop failed (process may have already exited)", "error", err)
		}
		exited := false
		select {
		case <-t.doneCh:
			exited = true
		case <-time.After(gracefulStopTimeout):
			log.Warn("Process did not exit after graceful stop, killing",
				"pid", pid, "timeout", gracefulStopTimeout)
			if err := proc.Kill(); err != nil {
				log.Debug("Kill failed (process may have already exited)", "error", err)
			}
			select {
			case <-t.doneCh:
				exited = true
			case <-time.After(time.Second):
				log.Warn("Process did not exit after kill", "pid", pid)
			}
		}
		if !exited {
			t.mu.Lock()
			t.closed = true
			t.mu.Unlock()
		}
	}

	t.closePTY()
	t.waitForReadDone()
	t.mu.Lock()
	t.closed = true
	t.mu.Unlock()
	log.Info("Terminal stopped")
}

func (t *Terminal) closePTY() {
	t.ptyCloseOnce.Do(func() {
		if t.proc != nil {
			_ = t.proc.Close()
		}
	})
}

// Detach closes and joins this reader without intentionally killing a
// daemon-backed child, allowing the next Runner process to recover it.
func (t *Terminal) Detach() {
	t.lifecycleMu.Lock()
	defer t.lifecycleMu.Unlock()

	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		t.waitForReadDone()
		return
	}
	t.stopping = true
	t.closed = true
	t.mu.Unlock()

	logger.Terminal().Info("Terminal detaching (daemon stays alive)")
	t.closePTY()
	t.waitForReadDone()
}
