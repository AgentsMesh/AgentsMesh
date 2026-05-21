package processmgr

import (
	"context"
	"errors"
	"os"
	"syscall"
	"time"
)

// daemonProcess Stop/Signal/Alive implementations. Split from
// process_daemon.go (which holds startup + monitor goroutine) so each file
// stays within the package's 200-line guideline; the split is by
// responsibility, not arbitrary line counting.

var _ Handle = (*daemonProcess)(nil)

func (p *daemonProcess) Alive() bool {
	select {
	case <-p.doneCh:
		return false
	default:
	}
	return daemonProcessAlive(p.PID())
}

func (p *daemonProcess) Signal(sig os.Signal) error {
	if !p.Alive() {
		return ErrAlreadyExited
	}
	return signalDaemon(p.PID(), sig)
}

// Stop drives the daemon's SIGTERM → poll → SIGKILL → poll protocol. Because
// the daemon's parent is init(1), not the runner, we cannot use cmd.Wait —
// liveness is observed by kill(pid, 0). monitorLoop will eventually close
// doneCh; doStop closes it earlier so the caller's Wait returns promptly.
func (p *daemonProcess) Stop(ctx context.Context) error {
	p.stopOnce.Do(func() {
		err := p.doStop(ctx)
		if err != nil && !errors.Is(err, ErrAlreadyExited) {
			p.stopErr.Store(&err)
		}
		p.setExit(ExitInfo{Duration: time.Since(p.StartedAt())})
	})
	if e := p.stopErr.Load(); e != nil {
		return *e
	}
	return nil
}

func (p *daemonProcess) doStop(ctx context.Context) error {
	if !p.Alive() {
		return nil
	}
	if err := signalDaemon(p.PID(), syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	if waitGone(ctx, p, p.stopTimeout) {
		return nil
	}
	if !p.Alive() {
		return nil
	}
	if err := signalDaemon(p.PID(), syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	if waitGone(ctx, p, p.stopTimeout) {
		return nil
	}
	return errors.New("processmgr: daemon did not exit after SIGKILL")
}

// waitGone polls Alive() until the daemon is gone, the timeout elapses, or
// the context is canceled. Returns true if the daemon is gone.
func waitGone(ctx context.Context, p *daemonProcess, timeout time.Duration) bool {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(p.pollEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return !p.Alive()
		case <-deadline.C:
			return !p.Alive()
		case <-ticker.C:
			if !p.Alive() {
				return true
			}
		}
	}
}
