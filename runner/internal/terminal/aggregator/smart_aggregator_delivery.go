package aggregator

import "time"

// FlushAndWait force-flushes pending aggregation data and waits until every
// active destination has completed all output accepted before the barrier.
// Callers that need a snapshot cut must prevent concurrent Write calls while
// this method runs and while the snapshot is sent.
func (a *SmartAggregator) FlushAndWait(timeout time.Duration) bool {
	a.mu.Lock()
	if a.stopped {
		a.mu.Unlock()
		return false
	}
	a.forceFlushLocked()
	barrier := a.router.barrier()
	a.mu.Unlock()

	if timeout <= 0 {
		timeout = defaultOutputDrainTimeout
	}
	return barrier.wait(timeout)
}

// PrepareSnapshot force-flushes pending data, establishes an ordered boundary
// for one destination, and freezes that destination's new deltas until Commit
// or Abort. Other destination lanes continue independently. The caller must
// hold the PTY stream-ordering lock for the full prepare/send/commit transaction.
func (a *SmartAggregator) PrepareSnapshot(
	destination string,
	timeout time.Duration,
) (*OutputSnapshotBoundary, bool) {
	a.mu.Lock()
	if a.stopped {
		a.mu.Unlock()
		return nil, false
	}
	a.forceFlushLocked()
	boundary, found := a.router.prepareSnapshot(destination)
	a.mu.Unlock()
	if !found {
		return nil, false
	}

	if timeout <= 0 {
		timeout = defaultOutputDrainTimeout
	}
	if !boundary.wait(timeout) {
		boundary.Abort(errSnapshotBarrierTimeout)
		return nil, false
	}
	return boundary, true
}

// SetOutputHealthHandler observes destination overflow and send failures.
func (a *SmartAggregator) SetOutputHealthHandler(handler OutputHealthHandler) {
	a.router.SetHealthHandler(handler)
}

// OutputHealth returns the current state of every destination lane.
func (a *SmartAggregator) OutputHealth() []OutputHealth {
	return a.router.Health()
}
