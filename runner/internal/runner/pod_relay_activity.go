package runner

import "sync"

// podActivity pins one admitted generation while arbitrary application code
// executes without relayTransitionMu. Retirement rejects new acquisitions and
// exposes a channel that closes after the last pinned callback returns.
type podActivity struct {
	mu        sync.Mutex
	accepting bool
	active    int
	drained   chan struct{}
	closed    bool
}

func newPodActivity() *podActivity {
	return &podActivity{accepting: true, drained: make(chan struct{})}
}

func (a *podActivity) acquire() (func(), bool) {
	if a == nil {
		return nil, false
	}
	a.mu.Lock()
	if !a.accepting {
		a.mu.Unlock()
		return nil, false
	}
	a.active++
	a.mu.Unlock()
	return func() { a.release() }, true
}

func (a *podActivity) release() {
	a.mu.Lock()
	if a.active > 0 {
		a.active--
	}
	if !a.accepting && a.active == 0 {
		a.closeDrainedLocked()
	}
	a.mu.Unlock()
}

func (a *podActivity) retire() <-chan struct{} {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	a.accepting = false
	if a.active == 0 {
		a.closeDrainedLocked()
	}
	drained := a.drained
	a.mu.Unlock()
	return drained
}

func (a *podActivity) closeDrainedLocked() {
	if !a.closed {
		a.closed = true
		close(a.drained)
	}
}

func waitPodActivities(drains ...<-chan struct{}) {
	for _, drain := range drains {
		if drain != nil {
			<-drain
		}
	}
}
