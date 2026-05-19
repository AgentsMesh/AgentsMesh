package runner

import "sync"

type AckTracker struct {
	mu      sync.Mutex
	pending map[string]struct{} // podKeys awaiting ACK
}

func NewAckTracker() *AckTracker {
	return &AckTracker{
		pending: make(map[string]struct{}),
	}
}

func (t *AckTracker) Register(podKey string) {
	t.mu.Lock()
	t.pending[podKey] = struct{}{}
	t.mu.Unlock()
}

func (t *AckTracker) Resolve(podKey string) {
	t.mu.Lock()
	delete(t.pending, podKey)
	t.mu.Unlock()
}

func (t *AckTracker) Remove(podKey string) {
	t.mu.Lock()
	delete(t.pending, podKey)
	t.mu.Unlock()
}
