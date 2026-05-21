package processmgr

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// manager is the concrete singleton. It owns the registry, options, and
// dispatches Start to the per-mode constructors. The opts field is the
// canonical source of policy (timeouts, intervals) — modes read it through
// the manager rather than from package-level constants.
type manager struct {
	ctx    context.Context
	cancel context.CancelFunc
	opts   Options

	mu        sync.Mutex
	processes map[Handle]struct{}
}

func newManager(parent context.Context, opts Options) *manager {
	ctx, cancel := context.WithCancel(parent)
	m := &manager{
		ctx:       ctx,
		cancel:    cancel,
		opts:      opts,
		processes: make(map[Handle]struct{}),
	}
	m.startReaper()
	return m
}

// New constructs an isolated manager for tests. Production code uses Init +
// Global; tests want independent instances so registry state cannot leak
// between test cases.
func New(parent context.Context, opts Options) Manager {
	return newManager(parent, opts.withDefaults())
}

func (m *manager) Start(ctx context.Context, spec Spec) (Handle, error) {
	if err := validateSpec(spec); err != nil {
		return nil, err
	}

	p, err := m.dispatch(ctx, spec)
	if err != nil {
		return nil, err
	}
	m.register(p)
	observeStart(spec.Mode, spec.Owner)
	return p, nil
}

func (m *manager) dispatch(ctx context.Context, spec Spec) (Handle, error) {
	switch spec.Mode {
	case ModeNormal:
		return startNormal(ctx, m, spec)
	case ModePTY:
		return startPTY(ctx, m, spec)
	case ModeDaemon:
		return startDaemon(ctx, m, spec)
	default:
		return nil, fmt.Errorf("processmgr: unknown mode %d", spec.Mode)
	}
}

func (m *manager) register(p Handle) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processes[p] = struct{}{}
}

func (m *manager) unregister(p Handle) {
	m.mu.Lock()
	delete(m.processes, p)
	m.mu.Unlock()
	observeExit(p)
}

func (m *manager) List() []Handle {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Handle, 0, len(m.processes))
	for p := range m.processes {
		out = append(out, p)
	}
	return out
}

func (m *manager) StopAll(ctx context.Context) error {
	return m.stopMatching(ctx, func(p Handle) bool { return p.Mode() != ModeDaemon })
}

func (m *manager) StopDaemons(ctx context.Context) error {
	return m.stopMatching(ctx, func(p Handle) bool { return p.Mode() == ModeDaemon })
}

func (m *manager) stopMatching(ctx context.Context, want func(Handle) bool) error {
	var targets []Handle
	for _, p := range m.List() {
		if want(p) {
			targets = append(targets, p)
		}
	}

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)
	for _, p := range targets {
		wg.Add(1)
		go func(p Handle) {
			defer wg.Done()
			if err := p.Stop(ctx); err != nil && !errors.Is(err, ErrAlreadyExited) {
				mu.Lock()
				errs = append(errs, fmt.Errorf("stop %s pid=%d: %w", p.Owner(), p.PID(), err))
				mu.Unlock()
			}
		}(p)
	}
	wg.Wait()
	return errors.Join(errs...)
}

func validateSpec(spec Spec) error {
	if spec.Owner == "" {
		return errors.New("processmgr: Spec.Owner is required")
	}
	if spec.Command == "" {
		return errors.New("processmgr: Spec.Command is required")
	}
	if spec.Mode == ModePTY && spec.PTYSize == nil {
		return errors.New("processmgr: ModePTY requires Spec.PTYSize")
	}
	if spec.PipeStdin && spec.Stdin != nil {
		return errors.New("processmgr: Spec.PipeStdin conflicts with Spec.Stdin")
	}
	if spec.PipeStdout && spec.Stdout != nil {
		return errors.New("processmgr: Spec.PipeStdout conflicts with Spec.Stdout")
	}
	if spec.PipeStderr && spec.Stderr != nil {
		return errors.New("processmgr: Spec.PipeStderr conflicts with Spec.Stderr")
	}
	if spec.Mode != ModeNormal && (spec.PipeStdin || spec.PipeStdout || spec.PipeStderr) {
		return errors.New("processmgr: stdio pipes are only supported in ModeNormal")
	}
	return nil
}

// stopTimeoutFor returns the per-call StopTimeout, falling back to the
// manager's default. Keeping this on Options (not on Spec) means callers
// usually don't have to think about timeouts at all — only the runner's
// startup code decides the policy.
func (o Options) stopTimeoutFor(spec Spec) time.Duration {
	if spec.StopTimeout > 0 {
		return spec.StopTimeout
	}
	return o.DefaultStopTimeout
}
