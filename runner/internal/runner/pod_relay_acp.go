package runner

import (
	"encoding/json"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

// ACPPodRelay implements PodRelay for ACP-mode pods.
type ACPPodRelay struct {
	podKey    string
	acpClient *acp.ACPClient
	onCommand func([]byte) // closure bound to pod ref at creation
}

// NewACPPodRelay creates a PodRelay for ACP mode.
func NewACPPodRelay(podKey string, acpClient *acp.ACPClient, onCommand func([]byte)) *ACPPodRelay {
	return &ACPPodRelay{
		podKey:    podKey,
		acpClient: acpClient,
		onCommand: onCommand,
	}
}

func (r *ACPPodRelay) SetupHandlers(rc relay.RelayClient) {
	rc.SetMessageHandler(relay.MsgTypeAcpCommand, r.onCommand)
}

func (r *ACPPodRelay) SendSnapshot(rc relay.RelayClient) {
	if r.acpClient == nil {
		return
	}
	snapshot := r.acpClient.GetSessionSnapshot()
	data, err := json.Marshal(snapshot)
	if err != nil {
		logger.Pod().Error("Failed to marshal ACP snapshot", "pod_key", r.podKey, "error", err)
		return
	}
	if err := rc.Send(relay.MsgTypeAcpSnapshot, data); err != nil {
		logger.Pod().Warn("Failed to send ACP snapshot via relay", "pod_key", r.podKey, "error", err)
	}
}

func (r *ACPPodRelay) OnRelayConnected(rc relay.RelayClient) {
	// No-op: ACP mode has no aggregator to wire
}

func (r *ACPPodRelay) OnRelayDisconnected() {
	// No-op: ACP mode has no aggregator to clear
}

// Compile-time interface check.
var _ PodRelay = (*ACPPodRelay)(nil)
