package runner

import "github.com/anthropics/agentsmesh/runner/internal/relay"

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
	IsPodConnected(podKey string) bool
	URL() string
}

// PodRelay abstracts mode-specific relay behavior.
// PTY and ACP pods implement this interface to encapsulate
// their relay wiring differences, eliminating IsACPMode() branches
// from the relay layer (OCP).
type PodRelay interface {
	// SetupHandlers registers mode-specific handlers on the relay client.
	// PTY: SetInputHandler + SetResizeHandler
	// ACP: SetAcpCommandHandler
	SetupHandlers(rc relay.RelayClient)

	// SendSnapshot sends the current state snapshot via the relay client.
	// PTY: VT snapshot + alt-screen redraw
	// ACP: ACPClient session snapshot JSON
	SendSnapshot(rc relay.RelayClient)

	// OnRelayConnected wires the relay client to mode-specific output routing.
	// PTY: Aggregator.SetRelayClient(rc)
	// ACP: no-op
	OnRelayConnected(rc relay.RelayClient)

	// OnRelayDisconnected clears the relay client from mode-specific components.
	// PTY: Aggregator.SetRelayClient(nil)
	// ACP: no-op
	OnRelayDisconnected()

	// BroadcastEvent fans out an out-of-band event payload to every active
	// transport (cloud client + local server). Used by ACP for session events
	// that don't flow through the aggregator. PTY-mode pods should no-op.
	BroadcastEvent(rc relay.RelayClient, msgType byte, payload []byte)
}
