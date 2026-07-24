package aggregator

func (l *outputLane) markDesynced(generation uint64, reason OutputDesyncReason, err error) {
	l.mu.Lock()
	if generation != l.generation || l.stopped {
		l.mu.Unlock()
		return
	}
	health, changed := l.desyncLocked(reason, err)
	l.mu.Unlock()
	l.notifyDesync(health, changed)
}

func (l *outputLane) desyncLocked(reason OutputDesyncReason, err error) (OutputHealth, bool) {
	if l.desynced {
		return l.healthLocked(), false
	}
	l.desynced = true
	l.reason = reason
	l.err = err
	return l.healthLocked(), true
}

func (l *outputLane) notifyDesync(health OutputHealth, changed bool) {
	if changed && l.onDesync != nil {
		l.onDesync(health)
	}
}

func (l *outputLane) forceDesync(reason OutputDesyncReason, err error) {
	l.mu.Lock()
	if l.stopped || l.writer == nil {
		l.mu.Unlock()
		return
	}
	health, changed := l.desyncLocked(reason, err)
	l.mu.Unlock()
	l.notifyDesync(health, changed)
}

func (l *outputLane) replaceWriter(writer RelayWriter, generation uint64) {
	l.mu.Lock()
	l.writer = writer
	l.generation = generation
	l.desynced = false
	l.snapshotPending = false
	l.reason = ""
	l.err = nil
	l.mu.Unlock()
}

func (l *outputLane) beginSnapshot(resetGeneration uint64) (*outputBarrierResult, uint64, bool) {
	result := newOutputBarrierResult()
	l.mu.Lock()
	if l.stopped || l.writer == nil || l.snapshotPending {
		l.mu.Unlock()
		result.complete(false)
		return result, 0, false
	}
	l.snapshotPending = true
	recoveryRequired := l.desynced
	if recoveryRequired {
		l.generation = resetGeneration
		l.desynced = false
		l.reason = ""
		l.err = nil
	}
	generation := l.generation
	select {
	case l.queue <- queuedOutput{generation: generation, barrier: result}:
		l.mu.Unlock()
		return result, generation, recoveryRequired
	default:
		health, changed := l.desyncLocked(OutputDesyncOverflow, nil)
		l.mu.Unlock()
		l.notifyDesync(health, changed)
		result.complete(false)
		return result, generation, recoveryRequired
	}
}

func (l *outputLane) commitSnapshot(generation uint64) bool {
	l.mu.Lock()
	if l.stopped || generation == 0 || generation != l.generation || !l.snapshotPending {
		l.mu.Unlock()
		return false
	}
	l.snapshotPending = false
	l.desynced = false
	l.reason = ""
	l.err = nil
	l.mu.Unlock()
	return true
}

func (l *outputLane) abortSnapshot(generation uint64, err error) {
	l.mu.Lock()
	if generation == 0 || generation != l.generation || !l.snapshotPending || l.stopped {
		l.mu.Unlock()
		return
	}
	l.snapshotPending = false
	health, changed := l.desyncLocked(OutputDesyncSnapshot, err)
	l.mu.Unlock()
	l.notifyDesync(health, changed)
}

func (l *outputLane) health() OutputHealth {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.healthLocked()
}

func (l *outputLane) healthLocked() OutputHealth {
	return OutputHealth{
		Destination:     l.name,
		Desynced:        l.desynced,
		SnapshotPending: l.snapshotPending,
		Reason:          l.reason,
		Err:             l.err,
		QueuedBytes:     l.queuedBytes,
		ByteBudget:      l.byteBudget,
	}
}
