//go:build !windows

package processmgr

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
)

// startDaemon (Unix) spawns the runner's __processmgr_launcher__ subcommand
// which double-forks the real daemon, reports its PID through a pipe at fd
// launcherPIDFd, and exits. The launcher's exit reparents the daemon to
// init(1), giving us a zombie-proof detachment. See RunLauncher in
// launcher.go for the launcher half.
func startDaemon(ctx context.Context, mgr *manager, spec Spec) (Handle, error) {
	selfPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("processmgr: os.Executable: %w", err)
	}

	args := append([]string{LauncherSubcommand, spec.Command}, spec.Args...)
	cmd := exec.Command(selfPath, args...) //nolint:gosec
	cmd.Env = spec.Env
	cmd.Dir = spec.Dir
	cmd.Stdin = spec.Stdin
	cmd.Stdout = spec.Stdout
	cmd.Stderr = spec.Stderr
	configureLauncherSysProcAttr(cmd)

	pidR, pidW, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("processmgr: pipe: %w", err)
	}
	cmd.ExtraFiles = []*os.File{pidW}

	if err := cmd.Start(); err != nil {
		_ = pidR.Close()
		_ = pidW.Close()
		return nil, fmt.Errorf("processmgr: launcher start %s: %w", spec.Owner, err)
	}
	_ = pidW.Close()

	daemonPID, err := readDaemonPID(ctx, pidR, mgr.opts.LauncherStartTimeout)
	_ = pidR.Close()
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, err
	}

	p := &daemonProcess{
		baseProcess: newBaseProcess(spec.Owner, ModeDaemon, daemonPID),
		mgr:         mgr,
		launcherCmd: cmd,
		launcherPID: cmd.Process.Pid,
		stopTimeout: mgr.opts.stopTimeoutFor(spec),
		pollEvery:   mgr.opts.DaemonAlivePoll,
	}

	safego.Go("processmgr-launcher-wait-"+spec.Owner, func() {
		if err := cmd.Wait(); err != nil {
			logger.Runner().Warn("processmgr: launcher wait error",
				"owner", spec.Owner, "launcher_pid", p.launcherPID, "err", err)
		}
	})
	safego.Go("processmgr-daemon-monitor-"+spec.Owner, p.monitorLoop)
	return p, nil
}

func readDaemonPID(ctx context.Context, r *os.File, timeout time.Duration) (int, error) {
	type result struct {
		pid int
		err error
	}
	ch := make(chan result, 1)
	go func() {
		scanner := bufio.NewScanner(r)
		if !scanner.Scan() {
			ch <- result{err: fmt.Errorf("processmgr: launcher closed pipe before reporting PID: %w", scanner.Err())}
			return
		}
		pid, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
		if err != nil {
			ch <- result{err: fmt.Errorf("processmgr: parse launcher PID: %w", err)}
			return
		}
		ch <- result{pid: pid}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case res := <-ch:
		return res.pid, res.err
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-timer.C:
		return 0, errors.New("processmgr: launcher timed out before reporting daemon PID")
	}
}

// configureLauncherSysProcAttr ensures the launcher process has Setsid set so
// it becomes its own session leader. The eventual daemon detachment happens
// when the launcher exits (see RunLauncher); Setsid here also prevents Ctrl-C
// in the runner from propagating to the daemon.
func configureLauncherSysProcAttr(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setsid = true
}

// daemonProcessAlive uses kill(pid, 0). ESRCH = gone; EPERM = exists but we
// can't signal — both treated as alive vs dead respectively.
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
