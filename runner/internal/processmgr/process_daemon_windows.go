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

// daemonProcessAlive probes a detached daemon on Windows. proc.Signal(0)
// returns "not supported by windows" — that error is not "process gone", so
// using it as a liveness probe (as the Unix path does) would make this
// function always return false and break daemon Stop. The Win32-correct
// probe is OpenProcess + GetExitCodeProcess: exit code 259 (STILL_ACTIVE)
// means the process is running; anything else means it has exited.
func daemonProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	const processQueryLimitedInformation = 0x1000
	const stillActive = 259
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(h)
	var code uint32
	if err := syscall.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == stillActive
}

// signalDaemon translates Unix-style termination signals to their Win32
// equivalents. syscall.SIGTERM is not deliverable through os.Process.Signal
// on Windows (returns "not supported by windows"), but the caller's intent
// is "terminate this process" — so we map both SIGTERM and SIGKILL to
// TerminateProcess via proc.Kill(). Signal(0) is unsupported too, but
// liveness probing flows through daemonProcessAlive (OpenProcess) above,
// so we never reach this function with sig == Signal(0).
func signalDaemon(pid int, sig os.Signal) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if sysSig, ok := sig.(syscall.Signal); ok {
		switch sysSig {
		case syscall.SIGTERM, syscall.SIGKILL:
			return proc.Kill()
		}
	}
	return proc.Signal(sig)
}
