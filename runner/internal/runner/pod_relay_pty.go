package runner

import (
	"sync"
	"sync/atomic"

	"github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
)

// PTYPodRelay implements PodRelay for PTY-mode pods.
type PTYPodRelay struct {
	podKey      string
	io          PodIO
	components  *PTYComponents
	localServer LocalRelayBroker
	localWriter *localOutputWriter

	cloudMu     sync.RWMutex
	cloudClient relay.RelayClient
	cloudEpoch  atomic.Uint64
	cloudRepair outputRepairState
	localRepair outputRepairState
}

type outputRepairState struct {
	active         atomic.Bool
	requestedEpoch atomic.Uint64
}

// NewPTYPodRelay constructs a PodRelay for PTY mode.
// localServer == nil disables local-side fanout (e.g. when 127.0.0.1 binding failed).
func NewPTYPodRelay(podKey string, io PodIO, comps *PTYComponents, localServer LocalRelayBroker) *PTYPodRelay {
	r := &PTYPodRelay{podKey: podKey, io: io, components: comps, localServer: localServer}
	if comps == nil || comps.Aggregator == nil {
		return r
	}
	comps.Aggregator.SetOutputHealthHandler(r.outputHealthHandler())
	if localServer != nil {
		r.localWriter = &localOutputWriter{server: localServer, podKey: podKey}
		comps.Aggregator.SetOutputDestination(aggregator.OutputDestinationLocal, r.localWriter)
	}
	return r
}

func (r *PTYPodRelay) SetupHandlers(rc relay.RelayClient, guard RelayInboundGuard) {
	input := r.inputHandler()
	resize := r.resizeHandler()
	rc.SetMessageHandler(relay.MsgTypeInput, func(payload []byte) {
		guard.runCloud(func(RelayInboundContext) { input(payload) })
	})
	rc.SetMessageHandler(relay.MsgTypeResize, func(payload []byte) {
		guard.runCloud(func(RelayInboundContext) { resize(payload) })
	})
	rc.SetMessageHandler(relay.MsgTypeSnapshotRequest, func(_ []byte) {
		guard.runCloud(func(RelayInboundContext) { r.SendSnapshot(rc) })
	})
	r.installLocalHandlers(guard)
}

func (r *PTYPodRelay) OnRelayConnected(rc relay.RelayClient) {
	if rc == nil || !rc.IsConnected() || r.components == nil {
		return
	}
	r.components.streamMu.Lock()
	defer r.components.streamMu.Unlock()
	r.setCloudClient(rc)
	r.cloudEpoch.Add(1)
	if agg := r.components.Aggregator; agg != nil {
		agg.SetOutputHealthHandler(r.outputHealthHandler())
		agg.SetOutputDestination(
			aggregator.OutputDestinationCloud,
			&cloudOutputWriter{client: rc},
		)
	}
	// A newly constructed Runner cannot know whether this relay channel already
	// has ready viewers from an earlier publisher process. Publisher attach is
	// therefore always destination recovery; requester baselines use SendSnapshot.
	r.deliverCloudSnapshotLocked(true, rc)
}

func (r *PTYPodRelay) OnRelayDisconnected(rc relay.RelayClient) {
	if rc == nil || r.components == nil {
		return
	}
	r.components.streamMu.Lock()
	defer r.components.streamMu.Unlock()
	if current := r.currentCloudClient(); rc != nil && current != rc {
		return
	}
	r.setCloudClient(nil)
	r.cloudEpoch.Add(1)
	if agg := r.components.Aggregator; agg != nil {
		agg.RemoveOutputDestination(aggregator.OutputDestinationCloud)
	}
}

// BroadcastEvent is a no-op for PTY pods; PTY output flows through the
// aggregator's destination lanes, not via discrete events.
func (r *PTYPodRelay) BroadcastEvent(_ relay.RelayClient, _ byte, _ []byte) {}

func (r *PTYPodRelay) setCloudClient(client relay.RelayClient) {
	r.cloudMu.Lock()
	r.cloudClient = client
	r.cloudMu.Unlock()
}

func (r *PTYPodRelay) currentCloudClient() relay.RelayClient {
	r.cloudMu.RLock()
	defer r.cloudMu.RUnlock()
	return r.cloudClient
}

var _ PodRelay = (*PTYPodRelay)(nil)
