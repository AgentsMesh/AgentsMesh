//go:build !windows

package processmgr

import (
	"os/exec"
	"syscall"
)

// configureDaemonSysProcAttr is invoked inside the launcher when it spawns
// the real daemon. Setsid here is the second of the double-fork pair — it
// puts the daemon in its own session so closing the launcher's controlling
// terminal cannot deliver SIGHUP to the daemon.
func configureDaemonSysProcAttr(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setsid = true
}
