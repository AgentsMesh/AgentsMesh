package runner

import (
	"context"

	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

// LocalRelayBroker is the subset of *relay.LocalServer that runner-internal
// consumers (PodRelay implementations + message handlers) depend on. Depending
// on the interface decouples those callers from the concrete server, lets us
// mock the local relay in tests, and keeps the runner-wide LocalServer's full
// API (Start/Stop/etc.) out of consumer reach.
type LocalRelayBroker interface {
	RegisterPod(podKey, expectedToken string)
	UnregisterPod(podKey string)
	SetMessageHandler(podKey string, msgType byte, handler func([]byte))
	SetRequestHandler(podKey string, msgType byte, handler relay.RequestHandler)
	Send(podKey string, msgType byte, payload []byte) error
	Flush(ctx context.Context, podKey string) error
	IsPodConnected(podKey string) bool
	URL() string
}

// RelayInboundContext is the runtime generation admitted for one inbound
// relay callback. Callbacks must use these references rather than resolving
// Pod state again while the generation guard is held.
type RelayInboundContext struct {
	IO     PodIO
	Relay  PodRelay
	Client relay.RelayClient
}

// RelayInboundGuard serializes cloud and local data-plane callbacks with Pod
// ownership, local registration, and runtime replacement. A zero guard rejects
// callbacks; production handler registration must receive one from Pod.
type RelayInboundGuard struct {
	cloud func(func(RelayInboundContext)) bool
	local func(func(RelayInboundContext)) bool
}

func (g RelayInboundGuard) runCloud(fn func(RelayInboundContext)) bool {
	if fn == nil || g.cloud == nil {
		return false
	}
	return g.cloud(fn)
}

func (g RelayInboundGuard) runLocal(fn func(RelayInboundContext)) bool {
	if fn == nil || g.local == nil {
		return false
	}
	return g.local(fn)
}

// PodRelay abstracts mode-specific relay behavior.
// PTY and ACP pods implement this interface to encapsulate
// their relay wiring differences, eliminating IsACPMode() branches
// from the relay layer (OCP).
type PodRelay interface {
	// SetupHandlers registers mode-specific handlers on the relay client.
	// PTY: SetInputHandler + SetResizeHandler
	// ACP: SetAcpCommandHandler
	SetupHandlers(rc relay.RelayClient, guard RelayInboundGuard)

	// SendSnapshot sends the current state snapshot via the relay client.
	// PTY: ordered VT snapshot
	// ACP: ACPClient session snapshot JSON
	SendSnapshot(rc relay.RelayClient)

	// RecoverSnapshot replaces every ready viewer after publisher-side data loss.
	// PTY snapshots carry reset_all; ACP snapshots are already authoritative.
	RecoverSnapshot(rc relay.RelayClient)

	// OnRelayConnected establishes the publisher and its authoritative baseline.
	// PTY installs the output writer and snapshot in one ordered transaction.
	// ACP sends its authoritative session snapshot.
	OnRelayConnected(rc relay.RelayClient)

	// OnRelayDisconnected clears the relay client from mode-specific components.
	// PTY: Aggregator.SetRelayClient(nil)
	// ACP: no-op
	OnRelayDisconnected(rc relay.RelayClient)

	// BroadcastEvent fans out an out-of-band event payload to every active
	// transport (cloud client + local server). Used by ACP for session events
	// that don't flow through the aggregator. PTY-mode pods should no-op.
	BroadcastEvent(rc relay.RelayClient, msgType byte, payload []byte)
}
