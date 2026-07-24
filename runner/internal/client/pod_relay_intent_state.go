package client

func (i *podRelayIntent) offerSubscribe(req SubscribePodRequest) (relayIntentAdmission, uint64) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.phase == relayIntentAborted {
		return relayIntentRetry, 0
	}
	i.clearPendingLocked()
	i.revision++
	i.subscribed = true
	req.IntentValid = nil
	reqCopy := req
	i.pending = &reqCopy
	switch i.phase {
	case relayIntentOpen:
		i.phase = relayIntentDraining
		return relayIntentStart, 0
	case relayIntentDraining:
		return relayIntentInvalidate, i.beginTeardownLocked()
	default:
		return relayIntentQueued, 0
	}
}

func (i *podRelayIntent) offerUnsubscribe() (relayIntentAdmission, uint64) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.phase == relayIntentAborted {
		return relayIntentRetry, 0
	}
	i.clearPendingLocked()
	i.revision++
	i.subscribed = false
	if i.phase == relayIntentDraining {
		return relayIntentInvalidate, i.beginTeardownLocked()
	}
	if i.phase == relayIntentPending {
		return relayIntentQueued, 0
	}
	return relayIntentDirect, 0
}

func (i *podRelayIntent) claimDrain() (uint64, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.phase == relayIntentAborted || i.drainOwned {
		return 0, false
	}
	if i.phase == relayIntentPending {
		i.phase = relayIntentDraining
	}
	if i.phase != relayIntentDraining {
		return 0, false
	}
	i.drainEpoch++
	i.drainOwned = true
	return i.drainEpoch, true
}

func (i *podRelayIntent) takeLatest(
	drainEpoch uint64,
) (SubscribePodRequest, uint64, <-chan struct{}, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.phase != relayIntentDraining || i.drainEpoch != drainEpoch {
		return SubscribePodRequest{}, 0, nil, false
	}
	if i.teardownDone != nil {
		return SubscribePodRequest{}, 0, i.teardownDone, false
	}
	if i.pending == nil {
		i.phase = relayIntentOpen
		i.drainOwned = false
		return SubscribePodRequest{}, 0, nil, false
	}
	req := *i.pending
	i.clearPendingLocked()
	return req, i.revision, nil, true
}

func (i *podRelayIntent) current(drainEpoch, revision uint64) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.phase == relayIntentDraining && i.drainEpoch == drainEpoch && i.revision == revision
}

func (i *podRelayIntent) abort() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	invalidate := i.phase == relayIntentDraining || (i.phase == relayIntentOpen && i.subscribed)
	i.clearPendingLocked()
	i.revision++
	i.subscribed = false
	i.phase = relayIntentAborted
	i.drainEpoch++
	i.drainOwned = false
	i.closeTeardownLocked()
	return invalidate
}

func (i *podRelayIntent) beginTeardownLocked() uint64 {
	if i.teardownDone == nil {
		i.teardownEpoch++
		i.teardownDone = make(chan struct{})
	}
	i.teardownPending++
	return i.teardownEpoch
}

func (i *podRelayIntent) endTeardown(teardownEpoch uint64) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.teardownDone == nil || i.teardownEpoch != teardownEpoch || i.teardownPending == 0 {
		return
	}
	i.teardownPending--
	if i.teardownPending == 0 {
		close(i.teardownDone)
		i.teardownDone = nil
	}
}

func (i *podRelayIntent) closeTeardownLocked() {
	if i.teardownDone != nil {
		close(i.teardownDone)
		i.teardownDone = nil
	}
	i.teardownPending = 0
}

// endDrain is generation-owned: an older goroutine cannot clear a request
// queued for a newer drain after the intent briefly returned to open.
func (i *podRelayIntent) endDrain(drainEpoch uint64) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.phase == relayIntentDraining && i.drainEpoch == drainEpoch {
		i.clearPendingLocked()
		i.revision++
		i.phase = relayIntentOpen
		i.drainOwned = false
	}
}

func (i *podRelayIntent) clearPendingLocked() {
	if i.pending == nil {
		return
	}
	i.pending.RunnerToken = ""
	i.pending.LocalToken = ""
	i.pending.IntentValid = nil
	i.pending = nil
}
