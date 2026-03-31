//go:build !windows

package main

import (
	"fmt"
	"log/slog"
	"os"
	"syscall"
)

// execRestartFunc returns a restart function that exec-replaces the current
// process with the (updated) binary on disk. Because the runner is a single
// binary, the updater can safely overwrite the file while we are still
// running — the OS keeps the old inode open. syscall.Exec then loads the
// new binary in-place, preserving the PID.
//
// execPath must be the canonical binary path resolved at startup — before any
// self-upgrade renames the file. After an upgrade, /proc/self/exe follows the
// old inode (now renamed to .old or deleted), so os.Executable() would return
// a stale or "(deleted)" path. Using the startup-resolved path avoids this.
func execRestartFunc(execPath string) func() (int, error) {
	return func() (int, error) {
		slog.Info("Exec-replacing process with updated binary", "path", execPath)

		err := syscall.Exec(execPath, os.Args, os.Environ())
		// If Exec succeeds, this line is never reached.
		return 0, fmt.Errorf("exec failed: %w", err)
	}
}
