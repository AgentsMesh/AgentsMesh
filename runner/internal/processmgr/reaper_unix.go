//go:build !windows

package processmgr

import (
	"errors"
	"syscall"
)

// reapOrphans drains any zombies in the runner's process table that nobody
// else has Wait'd on. waitpid(-1, WNOHANG) returns immediately when there is
// no work to do, so this is cheap to call on a timer.
//
// The expected steady-state return value is 0: every child started through
// processmgr is reaped by its own reapLoop. A non-zero return means a fork
// happened outside processmgr — which is exactly what we want this safety
// net to surface.
func reapOrphans() int {
	count := 0
	for {
		var ws syscall.WaitStatus
		pid, err := syscall.Wait4(-1, &ws, syscall.WNOHANG, nil)
		if pid <= 0 {
			return count
		}
		if err != nil && !errors.Is(err, syscall.EINTR) {
			return count
		}
		count++
	}
}
