//go:build windows

package process

import (
	"errors"
	"os"
)

// KillProcessTree terminates a process and all its descendants (bottom-up).
// On Windows, child processes are NOT automatically killed when the parent dies,
// so we must walk the process tree via Toolhelp32 snapshots and kill each one.
func KillProcessTree(pid int) error {
	inspector := &windowsInspector{}
	killDescendants(inspector, pid)

	// Finally kill the root process itself.
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	err = p.Kill()
	// Ignore "process already finished" — the process may have exited
	// between our check and the kill call, which is the desired outcome.
	if err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}
	return nil
}

// killDescendants recursively kills all descendants of the given pid (depth-first).
func killDescendants(inspector *windowsInspector, pid int) {
	children := inspector.GetChildProcesses(pid)
	for _, child := range children {
		killDescendants(inspector, child)
		if p, err := os.FindProcess(child); err == nil {
			_ = p.Kill()
		}
	}
}
