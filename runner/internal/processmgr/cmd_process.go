package processmgr

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// cmdProcess is the *exec.Cmd-backed lifecycle shared by ModeNormal and
// ModePTY. The two modes differ only in how the child is launched
// (exec.Cmd.Start vs pty.StartWithSize) and what is exposed alongside
// (stdio pipes vs *os.File for the PTY master); the SIGTERM → grace →
// SIGKILL escalation, the single reapLoop, and the Signal plumbing are
// identical. Embedding cmdProcess eliminates ~80 lines of duplicated Stop
// logic between process_normal.go and process_pty_unix.go.
type cmdProcess struct {
	*baseProcess
	mgr         *manager
	cmd         *exec.Cmd
	stopTimeout time.Duration
}

func (p *cmdProcess) Signal(sig os.Signal) error {
	if !p.Alive() {
		return ErrAlreadyExited
	}
	return signalProcessGroup(p.cmd.Process, sig)
}

// Stop drives the cooperative-then-forceful termination protocol. Because
// reapLoop runs unconditionally as a separate goroutine, Stop never calls
// cmd.Wait itself — it only signals and waits for the doneCh that reapLoop
// closes. That decoupling is what removes the historical race where a Stop
// timeout could leave Wait in-flight and the child momentarily a zombie.
func (p *cmdProcess) Stop(ctx context.Context) error {
	p.stopOnce.Do(func() {
		err := p.doStop(ctx)
		if err != nil && !errors.Is(err, ErrAlreadyExited) {
			p.stopErr.Store(&err)
		}
	})
	if errPtr := p.stopErr.Load(); errPtr != nil {
		return *errPtr
	}
	return nil
}

func (p *cmdProcess) doStop(ctx context.Context) error {
	if !p.Alive() {
		return nil
	}
	if err := signalProcessGroup(p.cmd.Process, syscall.SIGTERM); err != nil {
		logger.Runner().Warn("processmgr: SIGTERM failed, escalating", "owner", p.Owner(), "err", err)
	}

	graceful := time.NewTimer(p.stopTimeout)
	defer graceful.Stop()
	select {
	case <-p.Done():
		return nil
	case <-graceful.C:
	case <-ctx.Done():
	}

	if !p.Alive() {
		return nil
	}
	if err := signalProcessGroup(p.cmd.Process, syscall.SIGKILL); err != nil {
		logger.Runner().Error("processmgr: SIGKILL failed", "owner", p.Owner(), "err", err)
	}

	select {
	case <-p.Done():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// reapLoopBody is the body each concrete handle (normal, pty) runs in its
// reaper goroutine. The optional onWait callback runs after cmd.Wait returns
// but before doneCh closes, giving ptyProcess a place to close its PTY fd.
// The self parameter is the concrete Handle (normalProcess or ptyProcess);
// unregister needs it because *cmdProcess on its own does not implement
// Handle.PTY().
func (p *cmdProcess) reapLoopBody(self Handle, onWait func()) {
	startedAt := p.StartedAt()
	err := p.cmd.Wait()
	info := buildExitInfo(p.cmd, err, startedAt)
	if onWait != nil {
		onWait()
	}
	p.setExit(info)
	p.mgr.unregister(self)
}

func buildExitInfo(cmd *exec.Cmd, waitErr error, startedAt time.Time) ExitInfo {
	info := ExitInfo{Duration: time.Since(startedAt), Err: waitErr}
	if cmd.ProcessState != nil {
		info.Code = cmd.ProcessState.ExitCode()
		if ws, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
			if sig, ok := waitSignal(ws); ok {
				info.Signal = sig
			}
		}
	}
	return info
}
