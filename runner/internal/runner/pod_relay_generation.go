package runner

import "github.com/anthropics/agentsmesh/runner/internal/relay"

// RelayLifecycleTicket binds asynchronous connection work to one pod runtime.
type RelayLifecycleTicket uint64

// RelayHandlerGeneration is the private cloud activity lease prepared for one
// relay client. Candidate generations are not published on Pod until that
// exact client wins TryInstallRelayClient.
type RelayHandlerGeneration struct {
	ticket        RelayLifecycleTicket
	client        relay.RelayClient
	runtimeEpoch  uint64
	cloudActivity *podActivity
	installable   bool
}

func (g RelayHandlerGeneration) retire() {
	waitPodActivities(g.cloudActivity.retire())
}

// RelayLifecycle returns a ticket only while relay candidates may be prepared.
func (p *Pod) RelayLifecycle() (RelayLifecycleTicket, bool) {
	p.relayTransitionMu.Lock()
	defer p.relayTransitionMu.Unlock()
	p.relayMu.RLock()
	epoch := p.relayEpoch
	blocked := p.relayRuntimeBlocked || p.relayTransportBlock
	p.relayMu.RUnlock()
	status := p.GetStatus()
	active := status != PodStatusStopped && status != PodStatusFailed
	return RelayLifecycleTicket(epoch), active && !blocked && p.Relay != nil
}

// WithCurrentRelayClient runs fn only for the active owner in the current
// runtime. Reconnect callbacks use this to avoid touching a retired pipeline.
func (p *Pod) WithCurrentRelayClient(expected relay.RelayClient, fn func(PodRelay)) bool {
	if expected == nil || fn == nil {
		return false
	}
	p.relayTransitionMu.Lock()
	p.relayMu.RLock()
	activity := p.cloudActivity
	valid := p.RelayClient == expected && !p.relayRuntimeBlocked &&
		!p.relayTransportBlock && p.Relay != nil && activity != nil
	modeRelay := p.Relay
	p.relayMu.RUnlock()
	var release func()
	if valid {
		release, valid = activity.acquire()
	}
	p.relayTransitionMu.Unlock()
	if !valid {
		return false
	}
	defer release()
	fn(modeRelay)
	return true
}

// WithRelayLifecycle prepares handlers and local registration while the ticket
// still identifies the current runtime. Runtime replacement cannot cross fn.
func (p *Pod) WithRelayLifecycle(ticket RelayLifecycleTicket, fn func(PodRelay)) bool {
	p.relayTransitionMu.Lock()
	defer p.relayTransitionMu.Unlock()

	p.relayMu.RLock()
	valid := uint64(ticket) == p.relayEpoch && !p.relayRuntimeBlocked &&
		!p.relayTransportBlock
	p.relayMu.RUnlock()
	status := p.GetStatus()
	if !valid || status == PodStatusStopped || status == PodStatusFailed {
		return false
	}
	fn(p.Relay)
	return true
}

// WithRelayHandlerGeneration installs one generation of cloud/local inbound
// handlers while the lifecycle ticket is current. Cloud callbacks become
// executable only after expected becomes the installed owner. Local callbacks
// become executable immediately, but only until local teardown or runtime
// replacement invalidates the returned generation.
func (p *Pod) WithRelayHandlerGeneration(
	ticket RelayLifecycleTicket,
	expected relay.RelayClient,
	fn func(PodRelay, RelayInboundGuard),
) (RelayHandlerGeneration, bool) {
	if expected == nil || fn == nil {
		return RelayHandlerGeneration{}, false
	}

	p.relayTransitionMu.Lock()

	p.relayMu.Lock()
	status := p.GetStatus()
	valid := uint64(ticket) == p.relayEpoch && !p.relayRuntimeBlocked &&
		!p.relayTransportBlock &&
		status != PodStatusStopped && status != PodStatusFailed &&
		p.Relay != nil
	if !valid {
		p.relayMu.Unlock()
		p.relayTransitionMu.Unlock()
		return RelayHandlerGeneration{}, false
	}
	cloudActivity := newPodActivity()
	ownerRefresh := p.RelayClient == expected
	var previousCloud <-chan struct{}
	if ownerRefresh {
		previousCloud = p.cloudActivity.retire()
		p.cloudActivity = cloudActivity
	}
	previousLocal := p.localActivity.retire()
	localActivity := newPodActivity()
	p.localActivity = localActivity
	p.relayLocalEpoch++
	p.relayLocalActive = true
	runtimeEpoch := p.relayRuntimeEpoch
	localEpoch := p.relayLocalEpoch
	modeRelay := p.Relay
	generation := RelayHandlerGeneration{
		ticket:        ticket,
		client:        expected,
		runtimeEpoch:  runtimeEpoch,
		cloudActivity: cloudActivity,
		installable:   !ownerRefresh,
	}
	p.relayMu.Unlock()

	guard := RelayInboundGuard{
		cloud: func(callback func(RelayInboundContext)) bool {
			return p.withRelayInbound(expected, runtimeEpoch, 0, cloudActivity, callback)
		},
		local: func(callback func(RelayInboundContext)) bool {
			return p.withRelayInbound(nil, runtimeEpoch, localEpoch, localActivity, callback)
		},
	}
	fn(modeRelay, guard)
	p.relayTransitionMu.Unlock()
	waitPodActivities(previousCloud, previousLocal)
	return generation, true
}

func (p *Pod) withRelayInbound(
	expected relay.RelayClient,
	runtimeEpoch uint64,
	localEpoch uint64,
	activity *podActivity,
	fn func(RelayInboundContext),
) bool {
	if fn == nil {
		return false
	}
	p.relayTransitionMu.Lock()

	p.relayMu.RLock()
	valid := !p.relayRuntimeBlocked && p.relayRuntimeEpoch == runtimeEpoch &&
		p.Relay != nil && p.IO != nil && activity != nil
	if expected != nil {
		valid = valid && !p.relayTransportBlock &&
			p.RelayClient == expected && p.cloudActivity == activity
	} else {
		valid = valid && p.relayLocalActive && p.relayLocalEpoch == localEpoch &&
			p.localActivity == activity
	}
	ctx := RelayInboundContext{IO: p.IO, Relay: p.Relay, Client: p.RelayClient}
	p.relayMu.RUnlock()
	status := p.GetStatus()
	var release func()
	if valid && status != PodStatusStopped && status != PodStatusFailed {
		release, valid = activity.acquire()
	} else {
		valid = false
	}
	p.relayTransitionMu.Unlock()
	if !valid {
		return false
	}
	defer release()
	fn(ctx)
	return true
}
