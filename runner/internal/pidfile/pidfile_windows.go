//go:build windows

package pidfile

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/process"
)

// CleanupStaleProcess finds and kills any leftover runner from a previous run.
// Mirrors the Unix logic: checks if the process is alive, guards against PID reuse,
// kills the process tree, and waits for exit before removing the PID file.
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
		return nil
	}

	inspector := process.DefaultInspector()

	// Check if the process is still alive.
	if !inspector.IsRunning(pid) {
		slog.Info("Removing stale PID file (process not running)", "pid", pid)
		os.Remove(pidPath)
		return nil
	}

	// PID reuse guard — is this actually our runner or an unrelated process?
	// Normalize both names: lowercase + strip .exe suffix, because:
	// - GetProcessName (via exeName) returns lowercase without .exe
	// - recordedExec from os.Args[0] may include .exe and mixed case on Windows
	actualName := normalizeExeName(inspector.GetProcessName(pid))
	expected := normalizeExeName(recordedExec)
	if actualName != expected {
		slog.Info("PID reused by different process", "pid", pid,
			"expected", expected, "actual", actualName)
		os.Remove(pidPath)
		return nil
	}

	// It's our stale runner — kill the process tree.
	slog.Warn("Found stale runner process, terminating it",
		"pid", pid,
		"executable", recordedExec,
		"pid_file", pidPath)
	if err := process.KillProcessTree(pid); err != nil {
		return fmt.Errorf("cannot kill stale runner (PID %d): %w", pid, err)
	}

	// Wait up to 5s for the process to exit.
	if waitForExit(inspector, pid, 5*time.Second) {
		slog.Info("Stale runner terminated", "pid", pid)
		os.Remove(pidPath)
		return nil
	}

	return fmt.Errorf("cannot kill stale runner (PID %d) — please kill it manually and remove %s", pid, pidPath)
}

// waitForExit polls every 200ms until the process exits or timeout is reached.
func waitForExit(inspector process.Inspector, pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !inspector.IsRunning(pid) {
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

// normalizeExeName normalizes an executable name for comparison:
// strips .exe suffix (Windows) and converts to lowercase (NTFS is case-insensitive).
func normalizeExeName(name string) string {
	name = filepath.Base(name)
	name = strings.TrimSuffix(strings.ToLower(name), ".exe")
	return name
}
