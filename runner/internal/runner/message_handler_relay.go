package runner

import (
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

func (h *RunnerMessageHandler) registerLocalRelayPod(req client.SubscribePodRequest) {
	if local := h.runner.GetLocalRelayServer(); local != nil && req.LocalToken != "" {
		local.RegisterPod(req.PodKey, req.LocalToken)
	}
}

// setupRelayClientHandlers sets up all handlers for a relay client.
// Mode-specific behavior is delegated to PodRelay; shared handlers are wired directly.
func (h *RunnerMessageHandler) setupRelayClientHandlers(
	relayClient relay.RelayClient,
	pod *Pod,
	modeRelay PodRelay,
	guard RelayInboundGuard,
	req client.SubscribePodRequest,
) {
	log := logger.Pod()
	podKey := req.PodKey

	// Mode-specific handlers — delegated to PodRelay
	if modeRelay != nil {
		modeRelay.SetupHandlers(relayClient, guard)
	}

	// Shared: CloseHandler
	relayClient.SetCloseHandler(func() {
		log.Info("Relay connection closed permanently", "pod_key", podKey)
		if pod.ClearRelayClientIf(relayClient) {
		} else {
			log.Debug("Relay close handler skipped: client already replaced", "pod_key", podKey)
		}
	})

	// Shared: TokenExpiredHandler
	relayClient.SetTokenExpiredHandler(func() string {
		log.Info("Relay token expired, requesting new token", "pod_key", podKey)
		if err := h.conn.SendRequestRelayToken(podKey, relayClient.GetRelayURL()); err != nil {
			log.Error("Failed to send token refresh request", "pod_key", podKey, "error", err)
			return ""
		}
		newToken := pod.WaitForNewToken(30 * time.Second)
		if newToken == "" {
			log.Warn("Timeout waiting for new token", "pod_key", podKey)
		}
		return newToken
	})

	// Shared: ReconnectHandler
	relayClient.SetReconnectHandler(func() {
		log.Info("Relay reconnected, sending snapshot", "pod_key", podKey)
		pod.WithCurrentRelayClient(relayClient, func(currentRelay PodRelay) {
			currentRelay.RecoverSnapshot(relayClient)
		})
	})
}

// OnUnsubscribePod handles unsubscribe PTY command from server.
func (h *RunnerMessageHandler) OnUnsubscribePod(req client.UnsubscribePodRequest) error {
	log := logger.Pod()
	log.Info("Unsubscribing from terminal relay", "pod_key", req.PodKey)

	local := h.runner.GetLocalRelayServer()
	pod, ok := h.podStore.Get(req.PodKey)
	if !ok {
		if local != nil {
			local.UnregisterPod(req.PodKey)
		}
		return nil
	}

	pod.TeardownRelayTransports(local)
	log.Info("Successfully unsubscribed from terminal relay", "pod_key", req.PodKey)
	return nil
}
