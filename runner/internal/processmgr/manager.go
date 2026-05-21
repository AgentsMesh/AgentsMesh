package processmgr

import (
	"context"
	"errors"
	"sync"
)

// Manager is the registry of every long-lived child the runner owns. Init must
// be called once at runner startup so the reaper goroutine and metrics get
// installed; subsequent Global() returns the same singleton.
//
// StopAll deliberately leaves ModeDaemon processes alone so that PodDaemon's
// "survive across runner upgrade" semantic stays intact. Callers that really
// want to terminate daemons (e.g. systemctl stop with a confirmed shutdown)
// must call StopDaemons explicitly.
type Manager interface {
	Start(ctx context.Context, spec Spec) (Handle, error)
	List() []Handle
	StopAll(ctx context.Context) error
	StopDaemons(ctx context.Context) error
}

// ErrManagerNotInitialized is returned by Global before Init has been called.
// In tests, use New to obtain an isolated manager instead.
var ErrManagerNotInitialized = errors.New("processmgr: Init has not been called")

var (
	globalMu sync.RWMutex
	global   *manager
)

// Init constructs the singleton manager with the given policy options and
// starts its reaper goroutine. Pass Options{} for defaults; pass a partially-
// filled Options to override individual knobs. Calling Init twice in the same
// process is a programmer error and panics — short-lived subcommands like
// `webconsole` or `update` that do not need processmgr must not invoke it.
func Init(ctx context.Context, opts Options) {
	globalMu.Lock()
	defer globalMu.Unlock()
	if global != nil {
		panic("processmgr: Init called twice")
	}
	global = newManager(ctx, opts.withDefaults())
}

// Global returns the singleton manager set up by Init. Before Init runs it
// returns a sentinel whose Start fails with ErrManagerNotInitialized, so
// forgetting to call Init surfaces immediately on the first Start instead of
// crashing on nil deref.
func Global() Manager {
	globalMu.RLock()
	defer globalMu.RUnlock()
	if global == nil {
		return uninitializedManager{}
	}
	return global
}

// uninitializedManager makes the "forgot to call Init" failure explicit at the
// first Start call instead of crashing on nil deref.
type uninitializedManager struct{}

func (uninitializedManager) Start(context.Context, Spec) (Handle, error) {
	return nil, ErrManagerNotInitialized
}
func (uninitializedManager) List() []Handle                    { return nil }
func (uninitializedManager) StopAll(context.Context) error     { return ErrManagerNotInitialized }
func (uninitializedManager) StopDaemons(context.Context) error { return ErrManagerNotInitialized }
