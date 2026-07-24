package runner

// RelayRuntimeTransition identifies one exclusive runtime replacement window.
type RelayRuntimeTransition uint64

// TryBeginRelayRuntimeTransition starts one replacement only when no other
// replacement or terminal cleanup already owns the pod runtime.
func (p *Pod) TryBeginRelayRuntimeTransition() (RelayRuntimeTransition, bool) {
	p.relayTransitionMu.Lock()
	p.relayMu.Lock()

	status := p.GetStatus()
	if p.relayRuntimeBlocked || status == PodStatusStopped || status == PodStatusFailed {
		p.relayMu.Unlock()
		p.relayTransitionMu.Unlock()
		return 0, false
	}
	lease, runtimeDrain, cloudDrain, localDrain := p.beginRelayRuntimeTransitionLocked()
	p.relayMu.Unlock()
	p.relayTransitionMu.Unlock()
	waitPodActivities(runtimeDrain, cloudDrain, localDrain)
	return lease, true
}

// BeginRelayRuntimeTransition rejects candidates until the replacement runtime
// is fully assembled. Terminal cleanup uses this form to supersede an in-flight
// replacement; ordinary restarts use TryBeginRelayRuntimeTransition.
func (p *Pod) BeginRelayRuntimeTransition() RelayRuntimeTransition {
	p.relayTransitionMu.Lock()
	p.relayMu.Lock()
	lease, runtimeDrain, cloudDrain, localDrain := p.beginRelayRuntimeTransitionLocked()
	p.relayMu.Unlock()
	p.relayTransitionMu.Unlock()
	waitPodActivities(runtimeDrain, cloudDrain, localDrain)
	return lease
}

func (p *Pod) beginRelayRuntimeTransitionLocked() (
	RelayRuntimeTransition,
	<-chan struct{},
	<-chan struct{},
	<-chan struct{},
) {
	p.relayRuntimeBlocked = true
	p.relayRuntimeEpoch++
	p.relayEpoch++
	p.relayLocalEpoch++
	p.relayLocalActive = false
	runtimeDrain := p.runtimeActivity.retire()
	cloudDrain := p.cloudActivity.retire()
	localDrain := p.localActivity.retire()
	p.runtimeActivity = nil
	p.cloudActivity = nil
	p.localActivity = nil
	return RelayRuntimeTransition(p.relayRuntimeEpoch), runtimeDrain, cloudDrain, localDrain
}

// EndRelayRuntimeTransition commits only the Begin call that created lease.
// A later terminate/restart invalidates the lease and keeps relay blocked.
func (p *Pod) EndRelayRuntimeTransition(lease RelayRuntimeTransition) bool {
	p.relayTransitionMu.Lock()
	defer p.relayTransitionMu.Unlock()
	p.relayMu.Lock()
	status := p.GetStatus()
	valid := p.relayRuntimeBlocked && uint64(lease) == p.relayRuntimeEpoch &&
		status != PodStatusStopped && status != PodStatusFailed && p.Relay != nil
	if !valid {
		p.relayMu.Unlock()
		return false
	}
	p.relayRuntimeBlocked = false
	p.relayEpoch++
	p.runtimeActivity = newPodActivity()
	p.relayMu.Unlock()
	p.SetStatus(PodStatusRunning)
	return true
}

func (p *Pod) RelayRuntimeTransitionCurrent(lease RelayRuntimeTransition) bool {
	p.relayTransitionMu.Lock()
	defer p.relayTransitionMu.Unlock()
	p.relayMu.RLock()
	current := p.relayRuntimeBlocked && uint64(lease) == p.relayRuntimeEpoch
	p.relayMu.RUnlock()
	status := p.GetStatus()
	return current && status != PodStatusStopped && status != PodStatusFailed
}

func (p *Pod) installRuntime(runtime PodRuntime) bool {
	if !runtime.valid() {
		return false
	}
	p.relayTransitionMu.Lock()
	p.relayMu.Lock()
	runtimeDrain := p.runtimeActivity.retire()
	cloudDrain := p.cloudActivity.retire()
	localDrain := p.localActivity.retire()
	p.relayRuntimeEpoch++
	p.relayLocalEpoch++
	p.relayLocalActive = false
	p.installRuntimeLocked(runtime)
	p.runtimeActivity = newPodActivity()
	p.cloudActivity = nil
	p.localActivity = nil
	p.relayEpoch++
	p.relayMu.Unlock()
	p.relayTransitionMu.Unlock()
	waitPodActivities(runtimeDrain, cloudDrain, localDrain)
	return true
}

// installRuntimeDuringTransition publishes a fully assembled candidate without
// unblocking Relay subscribers. The caller starts it and then ends the same
// transition while holding the pod lifecycle commit lock.
func (p *Pod) installRuntimeDuringTransition(
	lease RelayRuntimeTransition,
	runtime PodRuntime,
) bool {
	if !runtime.valid() {
		return false
	}
	p.relayTransitionMu.Lock()
	defer p.relayTransitionMu.Unlock()
	p.relayMu.Lock()
	defer p.relayMu.Unlock()

	status := p.GetStatus()
	if !p.relayRuntimeBlocked || uint64(lease) != p.relayRuntimeEpoch ||
		status == PodStatusStopped || status == PodStatusFailed {
		return false
	}
	p.installRuntimeLocked(runtime)
	p.relayEpoch++
	return true
}

func (p *Pod) clearRuntimeDuringTransition(lease RelayRuntimeTransition) bool {
	p.relayTransitionMu.Lock()
	defer p.relayTransitionMu.Unlock()
	p.relayMu.Lock()
	defer p.relayMu.Unlock()
	if !p.relayRuntimeBlocked || uint64(lease) != p.relayRuntimeEpoch {
		return false
	}
	p.IO = nil
	p.Relay = nil
	p.vtProvider = nil
	p.relayEpoch++
	return true
}

// installedRuntime returns one coherent runtime generation. Lifecycle owners
// call it only after blocking new relay/control-plane work and before draining
// that generation.
func (p *Pod) installedRuntime() PodRuntime {
	p.relayTransitionMu.Lock()
	defer p.relayTransitionMu.Unlock()
	p.relayMu.RLock()
	defer p.relayMu.RUnlock()
	return PodRuntime{
		IO:         p.IO,
		Relay:      p.Relay,
		vtProvider: p.vtProvider,
	}
}

func (p *Pod) installRuntimeLocked(runtime PodRuntime) {
	p.IO = runtime.IO
	p.Relay = runtime.Relay
	p.vtProvider = runtime.vtProvider
}

// WithActiveIO orders one control-plane operation against runtime replacement.
// The callback must not retain io after it returns.
func (p *Pod) WithActiveIO(fn func(PodIO)) bool {
	if fn == nil {
		return false
	}
	p.relayTransitionMu.Lock()
	p.relayMu.Lock()
	blocked := p.relayRuntimeBlocked
	io := p.IO
	if !blocked && io != nil && p.runtimeActivity == nil {
		p.runtimeActivity = newPodActivity()
	}
	activity := p.runtimeActivity
	status := p.GetStatus()
	valid := !blocked && status != PodStatusInitializing && status != PodStatusStopped &&
		status != PodStatusFailed && io != nil && activity != nil
	var release func()
	if valid {
		release, valid = activity.acquire()
	}
	p.relayMu.Unlock()
	p.relayTransitionMu.Unlock()
	if !valid {
		return false
	}
	defer release()
	fn(io)
	return true
}
