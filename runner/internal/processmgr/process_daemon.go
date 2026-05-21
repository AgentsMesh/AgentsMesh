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
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
)

// LauncherSubcommand is the first argument the runner inspects in main(). When
// it matches, the process re-enters as a one-shot launcher that double-forks
// the real daemon binary, reports its PID via launcherPIDFd, and exits. See
// RunLauncher.
const LauncherSubcommand = "__processmgr_launcher__"

// launcherPIDFd is the file descriptor the launcher writes the daemon's real
// PID to. The kernel assigns ExtraFiles[i] to fd 3+i in the child, so this
// MUST match the index used in startDaemon's cmd.ExtraFiles slice. Define
// the protocol here in one place rather than scattering the magic number 3.
const launcherPIDFd = 3

// daemonProcess wraps a child that has been detached via double-fork. The
// underlying *exec.Cmd is the launcher, not the daemon itself; the daemon's
// PID lives on baseProcess and is what PID() exposes.
//
// Done() semantics are kept consistent with normalProcess: a background
// monitor goroutine polls liveness via kill(pid, 0) at Options.DaemonAlivePoll
// cadence and calls setExit when the daemon disappears, which closes doneCh.
// Stop() can also trigger setExit directly — even when Stop fails (SIGKILL
// didn't take), Done still closes so callers stop waiting. The contract is
// that Done means "we stopped tracking", not "the daemon is dead"; callers
// who need certainty must check Stop's return value.
type daemonProcess struct {
	*baseProcess
	mgr         *manager
	launcherCmd *exec.Cmd
	launcherPID int
	stopTimeout time.Duration
	pollEvery   time.Duration
}

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
	// ExtraFiles[0] becomes fd launcherPIDFd in the launcher subprocess.
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

// monitorLoop is what makes Done() semantics consistent across modes. Without
// it, a daemon dying of its own accord would never close doneCh — the runner
// would have to poll Alive() manually. Polling kill(pid, 0) here gives us at
// most one DaemonAlivePoll interval of latency before Done fires.
//
// The loop is intentionally NOT tied to manager.ctx: a runner shutdown leaves
// detached daemons running (PodDaemon's "survive across runner upgrade"
// semantic), so the monitor must outlive manager.ctx. The goroutine is freed
// by process exit if the runner restarts; otherwise it ends when either the
// daemon dies or Stop closes doneCh.
func (p *daemonProcess) monitorLoop() {
	defer p.mgr.unregister(p)
	t := time.NewTicker(p.pollEvery)
	defer t.Stop()
	for {
		select {
		case <-p.doneCh:
			return
		case <-t.C:
			if !daemonProcessAlive(p.PID()) {
				p.setExit(ExitInfo{Duration: time.Since(p.StartedAt())})
				return
			}
		}
	}
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

func (p *daemonProcess) PTY() *os.File { return nil }
