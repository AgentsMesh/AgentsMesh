//go:build !windows

package process

import (
	"fmt"
	"os"
	"syscall"
)

// IsAlive checks whether the process with the given PID is still running.
// On Unix, it sends signal 0 which checks process existence without side effects.
func IsAlive(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process not found: %w", err)
	}

	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return fmt.Errorf("process not running: %w", err)
	}

	return nil
}
