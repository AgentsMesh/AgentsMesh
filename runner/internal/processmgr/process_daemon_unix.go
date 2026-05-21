//go:build !windows

package processmgr

import (
	"os"
	"os/exec"
	"syscall"
)

// configureLauncherSysProcAttr ensures the launcher process has Setsid set.
// That alone does not detach the eventual daemon — that requires the launcher
// itself to exit while the daemon keeps running, which is what RunLauncher
// does. Setsid here also detaches the launcher from the runner's controlling
// terminal so that Ctrl-C in the runner does not propagate to the daemon.
func configureLauncherSysProcAttr(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setsid = true
}

// daemonProcessAlive uses kill(pid, 0) which returns ESRCH if the process is
// gone. EPERM means the process exists but we lack permission to signal it —
// still "alive" for our purposes.
func daemonProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}
	return err == syscall.EPERM
}

func signalDaemon(pid int, sig os.Signal) error {
	sysSig, ok := sig.(syscall.Signal)
	if !ok {
		proc, err := os.FindProcess(pid)
		if err != nil {
			return err
		}
		return proc.Signal(sig)
	}
	return syscall.Kill(pid, sysSig)
}
