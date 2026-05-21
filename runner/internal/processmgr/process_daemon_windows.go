//go:build windows

package processmgr

import (
	"os"
	"os/exec"
	"syscall"
)

// configureLauncherSysProcAttr applies CREATE_NEW_PROCESS_GROUP so the
// launcher (and consequently the daemon it spawns) does not receive console
// control events meant for the runner. Windows does not have setsid; process
// detachment is achieved by the launcher closing its handle to the daemon
// before exiting, which is the standard pattern.
func configureLauncherSysProcAttr(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= syscall.CREATE_NEW_PROCESS_GROUP
}

func daemonProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Windows, signal 0 is not supported; instead test via OpenProcess
	// inside FindProcess. A successful FindProcess does not guarantee
	// liveness, so we additionally call Signal(syscall.Signal(0)) which
	// returns ErrFinished if the process is gone.
	return proc.Signal(syscall.Signal(0)) == nil
}

func signalDaemon(pid int, sig os.Signal) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(sig)
}
