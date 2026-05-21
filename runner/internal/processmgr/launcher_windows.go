//go:build windows

package processmgr

import (
	"os/exec"
	"syscall"
)

// configureDaemonSysProcAttr applies DETACHED_PROCESS so the daemon does not
// share the launcher's console. Combined with the launcher's
// CREATE_NEW_PROCESS_GROUP this yields the same end state as Unix Setsid:
// the daemon cannot be killed by signals targeting the runner.
func configureDaemonSysProcAttr(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	const detachedProcess uint32 = 0x00000008
	cmd.SysProcAttr.CreationFlags |= detachedProcess
}
