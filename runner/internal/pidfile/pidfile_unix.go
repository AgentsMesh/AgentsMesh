//go:build !windows

package pidfile

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/process"
)

// CleanupStaleProcess finds and kills any leftover runner from a previous run.
// Safe to call when no stale process exists (no-op).
// Returns error only if it cannot kill the stale process.
func CleanupStaleProcess() error {
	pidPath := GetPath()
	if pidPath == "" {
		return nil
	}

	pid, recordedExec, err := parsePIDFile(pidPath)
	if err != nil {
		return err
	}
	if pid == 0 {
		return nil // No PID file or corrupt file (already cleaned up)
	}

	// After exec-replace the PID stays the same, so the new process
	// would see its own PID in the file and try to kill itself.
	if pid == os.Getpid() {
		return nil
	}

	// Check if process is alive (signal 0 = existence check)
	if err := syscall.Kill(pid, 0); err != nil {
		if errors.Is(err, syscall.EPERM) {
			// EPERM from signal 0 means the process exists but is owned by another user.
			return fmt.Errorf("stale runner (PID %d) is owned by another user — try running with sudo or as the same user that started it", pid)
		}
		// ESRCH = no such process → stale PID file from a dead process
		slog.Info("Removing stale PID file (process not running)", "pid", pid)
		os.Remove(pidPath)
		return nil
	}

	// PID reuse guard — is this actually our runner or an unrelated process?
	actualName := process.DefaultInspector().GetProcessName(pid)
	if filepath.Base(actualName) != recordedExec {
		// PID was recycled by the OS for a different program
		slog.Info("PID reused by different process", "pid", pid,
			"expected", recordedExec, "actual", actualName)
		os.Remove(pidPath)
		return nil
	}

	// It's our stale runner — kill it gracefully
	slog.Warn("Found stale runner process, terminating it",
		"pid", pid,
		"executable", recordedExec,
		"pid_file", pidPath)
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		if errors.Is(err, syscall.EPERM) {
			return fmt.Errorf("cannot kill stale runner (PID %d): permission denied — try running with sudo or as the same user that started it", pid)
		}
		return fmt.Errorf("cannot send SIGTERM to stale runner (PID %d): %w", pid, err)
	}

	// Wait up to 5s for graceful exit
	if waitForExit(pid, 5*time.Second) {
		slog.Info("Stale runner terminated gracefully", "pid", pid)
		os.Remove(pidPath)
		return nil
	}

	// Escalate to SIGKILL
	slog.Warn("SIGTERM ignored, sending SIGKILL", "pid", pid)
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
		if errors.Is(err, syscall.EPERM) {
			return fmt.Errorf("cannot force-kill stale runner (PID %d): permission denied — try running with sudo or as the same user that started it", pid)
		}
		return fmt.Errorf("cannot send SIGKILL to stale runner (PID %d): %w", pid, err)
	}

	if waitForExit(pid, 2*time.Second) {
		slog.Info("Stale runner force-killed", "pid", pid)
		os.Remove(pidPath)
		return nil
	}

	// Give up — something is very wrong
	return fmt.Errorf("cannot kill stale runner (PID %d) — please kill it manually and remove %s", pid, pidPath)
}

// waitForExit polls every 200ms until the process exits or timeout is reached.
func waitForExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); err != nil {
			return true // Process is gone
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}
