package runner

import (
	"fmt"
	"log/slog"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

// OnSubscribePod handles one browser's request to observe a pod. IntentValid
// binds deferred network work to the latest gRPC subscription revision.
func (h *RunnerMessageHandler) OnSubscribePod(req client.SubscribePodRequest) error {
	log := logger.Pod()
	relayURL := h.runner.GetConfig().RewriteRelayURL(req.RelayURL)
	if relayURL != req.RelayURL {
		log.Info("Relay URL rewritten", "pod_key", req.PodKey,
			"original", req.RelayURL, "rewritten", relayURL)
		req.RelayURL = relayURL
	}

	pod, ok := h.podStore.Get(req.PodKey)
	if !ok {
		return fmt.Errorf("pod not found: %s", req.PodKey)
	}
	return h.subscribePodRuntime(pod, req, req.IntentValid)
}

// subscribePodRuntime performs network work without Pod lifecycle locks. When
// current is non-nil, each ticket is followed by an intent check: a newer intent
// either fails that check or invalidates the ticket before installation.
func (h *RunnerMessageHandler) subscribePodRuntime(
	pod *Pod,
	req client.SubscribePodRequest,
	current func() bool,
) error {
	log := logger.Pod()
	relayURL := req.RelayURL
	log.Info("Subscribing to pod via Relay", "pod_key", req.PodKey, "relay_url", relayURL)

	if status := pod.GetStatus(); status == PodStatusStopped || status == PodStatusFailed {
		log.Info("Pod is not active, ignoring subscribe", "pod_key", req.PodKey, "status", status)
		return nil
	}

	ticket, accepting := pod.RelayLifecycle()
	if !accepting || !relayIntentCurrent(current) {
		log.Info("Pod relay runtime or subscription intent changed", "pod_key", req.PodKey)
		return nil
	}

	existingClient := pod.GetRelayClient()
	if existingClient != nil {
		if existingClient.IsConnected() && existingClient.GetRelayURL() == relayURL {
			_, updated := pod.WithRelayHandlerGeneration(ticket, existingClient, func(modeRelay PodRelay, guard RelayInboundGuard) {
				h.registerLocalRelayPod(req)
				h.setupRelayClientHandlers(existingClient, pod, modeRelay, guard, req)
				existingClient.UpdateToken(req.RunnerToken)
			})
			if updated {
				log.Info("Already connected to same relay, updated token",
					"pod_key", req.PodKey, "relay_url", relayURL)
			}
			return nil
		}
		log.Info("Disconnecting existing relay connection", "pod_key", req.PodKey,
			"old_relay_url", existingClient.GetRelayURL(), "new_relay_url", relayURL,
			"was_connected", existingClient.IsConnected())
		if pod.ClearRelayClientIf(existingClient) {
			existingClient.Stop()
		}
	}

	ticket, accepting = pod.RelayLifecycle()
	if !accepting || !relayIntentCurrent(current) {
		return nil
	}
	relayClient := h.relayClientFactory(relayURL, req.PodKey, req.RunnerToken,
		slog.Default().With("pod_key", req.PodKey))

	generation, prepared := pod.WithRelayHandlerGeneration(ticket, relayClient,
		func(modeRelay PodRelay, guard RelayInboundGuard) {
			h.registerLocalRelayPod(req)
			h.setupRelayClientHandlers(relayClient, pod, modeRelay, guard, req)
		})
	if !prepared {
		relayClient.Stop()
		return nil
	}
	if !relayIntentCurrent(current) {
		return retireRelayCandidate(relayClient, generation)
	}

	if err := relayClient.Connect(); err != nil {
		generation.retire()
		relayClient.Stop()
		return fmt.Errorf("failed to connect to relay: %w", err)
	}
	if !relayIntentCurrent(current) {
		return retireRelayCandidate(relayClient, generation)
	}
	if !relayClient.Start() {
		generation.retire()
		relayClient.Stop()
		return fmt.Errorf("failed to start relay client: client already stopped")
	}
	if !relayIntentCurrent(current) || !pod.TryInstallRelayClient(relayClient, generation) {
		generation.retire()
		log.Info("Pod or subscription changed while connecting; discarding candidate",
			"pod_key", req.PodKey, "status", pod.GetStatus())
		relayClient.Stop()
		return nil
	}

	log.Info("Successfully subscribed to pod via Relay", "pod_key", req.PodKey, "mode", pod.InteractionMode)
	return nil
}

func relayIntentCurrent(current func() bool) bool {
	return current == nil || current()
}

func retireRelayCandidate(client relay.RelayClient, generation RelayHandlerGeneration) error {
	generation.retire()
	client.Stop()
	return nil
}
