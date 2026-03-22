//go:build !windows

package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
)

// execRestartFunc returns a restart function that exec-replaces the current
// process with the (updated) binary on disk. Because the runner is a single
// binary, the updater can safely overwrite the file while we are still
// running — the OS keeps the old inode open. syscall.Exec then loads the
// new binary in-place, preserving the PID.
func execRestartFunc() func() (int, error) {
	return func() (int, error) {
		execPath, err := os.Executable()
		if err != nil {
			return 0, fmt.Errorf("get executable path: %w", err)
		}
		execPath, err = filepath.EvalSymlinks(execPath)
		if err != nil {
			return 0, fmt.Errorf("resolve symlinks: %w", err)
		}

		slog.Info("Exec-replacing process with updated binary", "path", execPath)

		err = syscall.Exec(execPath, os.Args, os.Environ())
		// If Exec succeeds, this line is never reached.
		return 0, fmt.Errorf("exec failed: %w", err)
	}
}
