package runner

import "sync"

// AckTracker tracks whether a Runner has acknowledged receipt of a CreatePod
// command. The tracker is purely event-driven — no timers or goroutines.
//
// Lifecycle:
//  1. Register(podKey) — called when Backend sends CreatePod
//  2. Resolve(podKey) — called when Runner sends "received" progress, PodCreated, or PodError
//  3. Remove(podKey)  — called on disconnect cleanup or pod termination
//
// The heartbeat reconciler uses IsPending to distinguish "Runner is still
// initializing" (ACK not yet received) from "PodCreated message was lost"
// (ACK received but pod still shows as initializing in DB).
type AckTracker struct {
	mu      sync.Mutex
	pending map[string]struct{} // podKeys awaiting ACK
}

// NewAckTracker creates a new AckTracker.
func NewAckTracker() *AckTracker {
	return &AckTracker{
		pending: make(map[string]struct{}),
	}
}

// Register marks a podKey as awaiting ACK.
func (t *AckTracker) Register(podKey string) {
	t.mu.Lock()
	t.pending[podKey] = struct{}{}
	t.mu.Unlock()
}

// Resolve marks the ACK as received. Safe to call for unknown keys.
func (t *AckTracker) Resolve(podKey string) {
	t.mu.Lock()
	delete(t.pending, podKey)
	t.mu.Unlock()
}

// Remove cancels tracking without marking as resolved.
func (t *AckTracker) Remove(podKey string) {
	t.mu.Lock()
	delete(t.pending, podKey)
	t.mu.Unlock()
}
