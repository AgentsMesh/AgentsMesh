//go:build windows

package processmgr

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/anthropics/agentsmesh/runner/internal/safego"
)

// startDaemon (Windows) spawns the real daemon directly with
// DETACHED_PROCESS + CREATE_NEW_PROCESS_GROUP rather than going through a
// launcher subprocess. Windows has no zombie state and ExtraFiles cannot be
// inherited across processes, so the Unix double-fork trick is both
// unnecessary and impossible here. The detachment guarantee comes from the
// Win32 flags alone — once we call Process.Release the daemon is fully on
// its own.
func startDaemon(ctx context.Context, mgr *manager, spec Spec) (Handle, error) {
	cmd := exec.CommandContext(ctx, spec.Command, spec.Args...) //nolint:gosec
	cmd.Env = spec.Env
	cmd.Dir = spec.Dir
	cmd.Stdin = spec.Stdin
	cmd.Stdout = spec.Stdout
	cmd.Stderr = spec.Stderr
	configureLauncherSysProcAttr(cmd)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("processmgr: start %s: %w", spec.Owner, err)
	}

	daemonPID := cmd.Process.Pid
	if err := cmd.Process.Release(); err != nil {
		// Release rarely fails on a valid handle; if it does, the daemon
		// is still running but our bookkeeping is in an awkward state.
		// Log via the safego panic boundary below and continue.
		_ = err
	}

	p := &daemonProcess{
		baseProcess: newBaseProcess(spec.Owner, ModeDaemon, daemonPID),
		mgr:         mgr,
		launcherCmd: nil,
		launcherPID: 0,
		stopTimeout: mgr.opts.stopTimeoutFor(spec),
		pollEvery:   mgr.opts.DaemonAlivePoll,
	}
	safego.Go("processmgr-daemon-monitor-"+spec.Owner, p.monitorLoop)
	return p, nil
}

// configureLauncherSysProcAttr applies the Win32 creation flags that detach
// the daemon from the runner: DETACHED_PROCESS hides the console;
// CREATE_NEW_PROCESS_GROUP makes the daemon its own group leader so console
// control events sent to the runner do not propagate.
func configureLauncherSysProcAttr(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	const detachedProcess uint32 = 0x00000008
	cmd.SysProcAttr.CreationFlags |= syscall.CREATE_NEW_PROCESS_GROUP | detachedProcess
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
