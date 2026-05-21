//go:build windows

package processmgr

import (
	"os"
	"os/exec"
	"syscall"
)

// applyNewProcessGroup makes the child the root of a new Windows process group.
// CREATE_NEW_PROCESS_GROUP also lets the child receive CTRL_BREAK_EVENT
// independently of the runner's own console signals.
func applyNewProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= syscall.CREATE_NEW_PROCESS_GROUP
}

// signalProcessGroup on Windows cannot use kill(-pgid). The runner emulates
// process-group termination by killing just the leader; descendants are
// expected to react to handle closure. For a true tree kill the caller can
// fall back to taskkill /F /T, but that is not done by default because most
// runner children are well-behaved Go programs.
func signalProcessGroup(proc *os.Process, sig os.Signal) error {
	if proc == nil {
		return ErrAlreadyExited
	}
	return proc.Signal(sig)
}

func waitSignal(ws syscall.WaitStatus) (os.Signal, bool) {
	return nil, false
}
