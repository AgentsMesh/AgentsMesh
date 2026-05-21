//go:build !windows

package processmgr

import (
	"os"
	"os/exec"
	"syscall"
)

// applyNewProcessGroup makes the child the leader of a fresh process group.
// signalProcessGroup later uses kill(-pgid, sig) to reach grandchildren that
// the immediate child has forked — without this, SIGTERM only reaches the
// top-level shell and leaks any inner subprocesses.
func applyNewProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

func signalProcessGroup(proc *os.Process, sig os.Signal) error {
	if proc == nil {
		return ErrAlreadyExited
	}
	sysSig, ok := sig.(syscall.Signal)
	if !ok {
		return proc.Signal(sig)
	}
	// Negative PID targets the entire process group.
	if err := syscall.Kill(-proc.Pid, sysSig); err == nil {
		return nil
	}
	// Fall back to single-process kill if the group is already gone.
	return proc.Signal(sig)
}

func waitSignal(ws syscall.WaitStatus) (os.Signal, bool) {
	if ws.Signaled() {
		return ws.Signal(), true
	}
	return nil, false
}
