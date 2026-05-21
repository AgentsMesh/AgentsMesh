package processmgr

import (
	"os"
	"os/exec"
	"time"
)

// LauncherSubcommand is the first argument the runner inspects in main(). When
// it matches, the process re-enters as a one-shot launcher that double-forks
// the real daemon binary on Unix, reports its PID via launcherPIDFd, and
// exits. Windows skips the launcher path entirely — see
// process_daemon_windows.go — because DETACHED_PROCESS already gives us the
// same parent-detachment guarantees without needing a middleman.
const LauncherSubcommand = "__processmgr_launcher__"

// launcherPIDFd is the file descriptor the Unix launcher writes the daemon's
// real PID to. ExtraFiles[i] becomes fd 3+i in the child, so this MUST match
// the index used in cmd.ExtraFiles when wiring the pipe. Windows does not
// inherit ExtraFiles, hence the separate Windows code path.
const launcherPIDFd = 3

// daemonProcess wraps a child that has been detached from the runner. On
// Unix this happens via a launcher subprocess (double-fork); on Windows the
// daemon is spawned directly with DETACHED_PROCESS so launcherCmd stays nil.
//
// Done() semantics are kept consistent with normalProcess: a background
// monitor goroutine polls liveness via kill(pid, 0) (or Windows equivalent)
// at Options.DaemonAlivePoll cadence and calls setExit when the daemon
// disappears, which closes doneCh. Stop() can also trigger setExit directly
// — even when Stop fails, Done still closes so callers stop waiting. The
// contract is that Done means "we stopped tracking", not "the daemon is
// dead"; callers who need certainty must check Stop's return value.
type daemonProcess struct {
	*baseProcess
	mgr         *manager
	launcherCmd *exec.Cmd // nil on Windows; on Unix this is the short-lived launcher
	launcherPID int       // 0 on Windows
	stopTimeout time.Duration
	pollEvery   time.Duration
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

func (p *daemonProcess) PTY() *os.File { return nil }
