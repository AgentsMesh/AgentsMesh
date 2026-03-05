//go:build windows

package pidfile

import (
	"log/slog"
	"os"
)

// CleanupStaleProcess on Windows removes stale PID files.
// Signal-based process termination is not supported on Windows;
// the service manager handles process lifecycle instead.
func CleanupStaleProcess() error {
	pidPath := GetPath()
	if pidPath == "" {
		return nil
	}

	pid, _, err := parsePIDFile(pidPath)
	if err != nil {
		return err
	}
	if pid == 0 {
		return nil
	}

	// On Windows, we only remove the stale PID file.
	// The Windows service manager (SCM) handles process lifecycle.
	slog.Info("Removing stale PID file", "pid", pid)
	os.Remove(pidPath)
	return nil
}
