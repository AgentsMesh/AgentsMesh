package runner

import "github.com/anthropics/agentsmesh/runner/internal/relay"

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
}
