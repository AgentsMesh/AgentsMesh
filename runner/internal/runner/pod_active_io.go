package runner

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
