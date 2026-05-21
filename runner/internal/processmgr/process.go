package processmgr

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Handle is the value returned by Manager.Start. Callers never see the
// underlying *exec.Cmd because that would let them invoke Start without Wait
// — the original source of zombie accumulation. The name "Handle" is
// deliberate: this is a control surface for a child process, not the child
// process itself.
//
// Done() semantics differ slightly across modes:
//
//   - ModeNormal / ModePTY: Done closes when the child's reapLoop sees
//     cmd.Wait return — i.e. the kernel has reaped the process. Once Done is
//     closed, the child is guaranteed gone.
//
//   - ModeDaemon: a background watcher polls liveness (every
//     Options.DaemonAlivePoll) and closes Done when the daemon disappears.
//     Stop also closes Done — even when Stop fails (SIGKILL didn't take and
//     the daemon is stuck), Done still closes so callers stop blocking. In
//     ModeDaemon, Done closing only means "we stopped tracking"; callers who
//     must know the daemon is truly gone have to check Stop's return value
//     and/or call Alive afterwards.
type Handle interface {
	PID() int
	Owner() string
	Mode() Mode
	StartedAt() time.Time

	PTY() *os.File

	StdinWriter() io.WriteCloser
	StdoutReader() io.ReadCloser
	StderrReader() io.ReadCloser

	Signal(sig os.Signal) error
	Stop(ctx context.Context) error
	Wait(ctx context.Context) (ExitInfo, error)
	Done() <-chan struct{}
	ExitInfo() (ExitInfo, bool)

	Alive() bool
}

// ErrAlreadyExited is returned by Signal/Stop when the child has already been
// reaped. Callers normally treat this as success — the desired end state.
var ErrAlreadyExited = errors.New("processmgr: process already exited")

// baseProcess is the shared state machine used by every concrete handle. PID
// is stored as a plain int because it is set once during construction and
// never changes — there is no concurrent writer for an immutable identity.
type baseProcess struct {
	owner     string
	mode      Mode
	startedAt time.Time
	pid       int

	exitOnce sync.Once
	exit     atomic.Pointer[ExitInfo]
	doneCh   chan struct{}

	stopOnce sync.Once
	stopErr  atomic.Pointer[error]
}

func newBaseProcess(owner string, mode Mode, pid int) *baseProcess {
	return &baseProcess{
		owner:     owner,
		mode:      mode,
		startedAt: time.Now(),
		pid:       pid,
		doneCh:    make(chan struct{}),
	}
}

func (b *baseProcess) PID() int             { return b.pid }
func (b *baseProcess) Owner() string        { return b.owner }
func (b *baseProcess) Mode() Mode           { return b.mode }
func (b *baseProcess) StartedAt() time.Time { return b.startedAt }
func (b *baseProcess) Done() <-chan struct{} { return b.doneCh }

func (b *baseProcess) ExitInfo() (ExitInfo, bool) {
	if p := b.exit.Load(); p != nil {
		return *p, true
	}
	return ExitInfo{}, false
}

func (b *baseProcess) setExit(info ExitInfo) {
	b.exitOnce.Do(func() {
		b.exit.Store(&info)
		close(b.doneCh)
	})
}

func (b *baseProcess) Wait(ctx context.Context) (ExitInfo, error) {
	select {
	case <-b.doneCh:
		info, _ := b.ExitInfo()
		return info, nil
	case <-ctx.Done():
		return ExitInfo{}, ctx.Err()
	}
}

func (b *baseProcess) Alive() bool {
	select {
	case <-b.doneCh:
		return false
	default:
		return true
	}
}

// StdinWriter / StdoutReader / StderrReader return nil by default. Only
// cmdProcess (used by ModeNormal) overrides these when Spec.PipeStdin/Stdout/
// Stderr is set. Daemon and PTY handles have no caller-accessible stdio pipes
// by design.
func (b *baseProcess) StdinWriter() io.WriteCloser { return nil }
func (b *baseProcess) StdoutReader() io.ReadCloser { return nil }
func (b *baseProcess) StderrReader() io.ReadCloser { return nil }
