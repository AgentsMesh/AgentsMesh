package processmgr

import "time"

// Options configures the Manager. All fields have sensible defaults; pass an
// empty Options{} to use them. These are the policy knobs the runner can tune
// without touching mechanism code (separating mechanism from policy is the
// whole point — the manager's job is "how to spawn/reap", not "how often").
type Options struct {
	// ReaperInterval is how often the safety-net wait4(WNOHANG) sweep runs.
	// In steady state it finds nothing; raise this for less-frequent sweeps
	// if metric runner_zombie_reaped_total stays at zero in production.
	ReaperInterval time.Duration

	// DefaultStopTimeout is the SIGTERM-to-SIGKILL grace period applied when
	// Spec.StopTimeout is zero. Tighten for chatty short-lived children;
	// raise for processes that need time to flush state.
	DefaultStopTimeout time.Duration

	// DaemonAlivePoll is how often a detached daemon's liveness is checked
	// via kill(pid, 0). Used by Wait() polling and the background watcher
	// that closes Done() on natural death.
	DaemonAlivePoll time.Duration

	// LauncherStartTimeout bounds how long Start blocks waiting for the
	// launcher subprocess to report the daemon PID via launcherPIDFd. Anything
	// slower than this is treated as a fork failure.
	LauncherStartTimeout time.Duration
}

// withDefaults fills in any zero-valued fields with the package defaults.
// Callers pass Options{} or a partially-set value; this is where the
// "what's the real value" decision happens.
func (o Options) withDefaults() Options {
	if o.ReaperInterval <= 0 {
		o.ReaperInterval = 30 * time.Second
	}
	if o.DefaultStopTimeout <= 0 {
		o.DefaultStopTimeout = 5 * time.Second
	}
	if o.DaemonAlivePoll <= 0 {
		o.DaemonAlivePoll = 500 * time.Millisecond
	}
	if o.LauncherStartTimeout <= 0 {
		o.LauncherStartTimeout = 10 * time.Second
	}
	return o
}
