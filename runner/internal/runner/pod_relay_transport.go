package runner

import (
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

func (p *Pod) SetRelayClient(client relay.RelayClient) {
	p.relayTransitionMu.Lock()
	p.relayMu.Lock()
	p.RelayClient = client
	p.relayEpoch++
	p.relayMu.Unlock()
	p.relayTransitionMu.Unlock()
}

func (p *Pod) GetRelayClient() relay.RelayClient {
	p.relayMu.RLock()
	defer p.relayMu.RUnlock()
	return p.RelayClient
}

// BroadcastRelayEvent sends one event through the currently active relay
// generation. Runtime replacement cannot retire that generation mid-send.
func (p *Pod) BroadcastRelayEvent(msgType byte, payload []byte) bool {
	ticket, ok := p.RelayLifecycle()
	if !ok {
		return false
	}
	return p.WithRelayLifecycle(ticket, func(modeRelay PodRelay) {
		modeRelay.BroadcastEvent(p.GetRelayClient(), msgType, payload)
	})
}

// TryInstallRelayClient atomically turns a prepared candidate into the current
// owner and publishes that candidate's activity lease in the same serialized
// transition. A competing candidate cannot retire the winning generation while
// it is still connecting.
func (p *Pod) TryInstallRelayClient(client relay.RelayClient, generation RelayHandlerGeneration) bool {
	if client == nil || generation.client != client || generation.cloudActivity == nil || !generation.installable {
		return false
	}
	p.relayTransitionMu.Lock()
	defer p.relayTransitionMu.Unlock()

	p.relayMu.Lock()
	status := p.GetStatus()
	valid := uint64(generation.ticket) == p.relayEpoch &&
		generation.runtimeEpoch == p.relayRuntimeEpoch && !p.relayRuntimeBlocked &&
		!p.relayTransportBlock &&
		p.RelayClient == nil && p.cloudActivity == nil && client.IsConnected() &&
		status != PodStatusStopped && status != PodStatusFailed
	if !valid {
		p.relayMu.Unlock()
		return false
	}
	p.RelayClient = client
	p.cloudActivity = generation.cloudActivity
	p.relayEpoch++
	p.relayMu.Unlock()

	if p.Relay != nil {
		p.Relay.OnRelayConnected(client)
	}
	return true
}

// ClearRelayClientIf retires exactly one owner and its publisher generation.
func (p *Pod) ClearRelayClientIf(expected relay.RelayClient) bool {
	if expected == nil {
		return false
	}
	p.relayTransitionMu.Lock()

	p.relayMu.Lock()
	if p.RelayClient != expected {
		p.relayMu.Unlock()
		p.relayTransitionMu.Unlock()
		return false
	}
	p.RelayClient = nil
	p.relayEpoch++
	cloudDrain := p.cloudActivity.retire()
	p.cloudActivity = nil
	modeRelay := p.Relay
	p.relayMu.Unlock()
	if modeRelay != nil {
		modeRelay.OnRelayDisconnected(expected)
	}
	p.relayTransitionMu.Unlock()
	waitPodActivities(cloudDrain)
	return true
}

// LockRelay and UnlockRelay remain for low-level race tests. Production
// ownership transitions use the ticketed methods above.
func (p *Pod) LockRelay()   { p.relayMu.Lock() }
func (p *Pod) UnlockRelay() { p.relayMu.Unlock() }

func (p *Pod) HasRelayClient() bool {
	p.relayMu.RLock()
	defer p.relayMu.RUnlock()
	return p.RelayClient != nil && p.RelayClient.IsConnected()
}

// TeardownRelayTransports is the linearization point for retiring the cloud
// publisher and the local viewer generation. The epoch is invalidated before
// local registration is removed, so a subscriber holding an older lifecycle
// ticket cannot recreate the local token or handlers after teardown.
//
// The network client is stopped after releasing lifecycle locks because Stop
// may synchronously invoke callbacks that need to inspect relay ownership.
func (p *Pod) TeardownRelayTransports(local LocalRelayBroker) {
	p.relayTransitionMu.Lock()
	p.relayMu.Lock()
	rc := p.RelayClient
	modeRelay := p.Relay
	p.RelayClient = nil
	p.relayEpoch++
	p.relayTransportEpoch++
	transition := p.relayTransportEpoch
	p.relayTransportBlock = true
	cloudDrain := p.cloudActivity.retire()
	p.cloudActivity = nil
	var localDrain <-chan struct{}
	if local != nil {
		p.relayLocalEpoch++
		p.relayLocalActive = false
		localDrain = p.localActivity.retire()
		p.localActivity = nil
	}
	p.relayMu.Unlock()
	p.relayTransitionMu.Unlock()

	if modeRelay != nil && rc != nil {
		modeRelay.OnRelayDisconnected(rc)
	}
	waitPodActivities(cloudDrain, localDrain)

	p.relayTransitionMu.Lock()
	p.relayMu.RLock()
	current := transition == p.relayTransportEpoch
	p.relayMu.RUnlock()
	if current && local != nil {
		local.UnregisterPod(p.PodKey)
	}
	if current {
		p.relayMu.Lock()
		if transition == p.relayTransportEpoch {
			p.relayTransportBlock = false
		}
		p.relayMu.Unlock()
	}
	p.relayTransitionMu.Unlock()

	if rc != nil {
		logger.Pod().Debug("Disconnecting relay client", "pod_key", p.PodKey)
		rc.Stop()
	}
}

// DisconnectRelay retires only the cloud relay transport. Callers that also
// own a local viewer generation should use TeardownRelayTransports.
func (p *Pod) DisconnectRelay() {
	p.TeardownRelayTransports(nil)
}
