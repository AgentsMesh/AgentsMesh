//go:build !windows

package process

import "os"

// KillProcessTree terminates a process.
// On Unix, PTY-based processes already use process groups (SIGKILL to -pgid),
// so only the root process needs to be killed here. The kernel handles the rest.
func KillProcessTree(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}
