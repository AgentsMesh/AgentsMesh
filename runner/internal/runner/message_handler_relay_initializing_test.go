package runner

import (
	"log/slog"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

// TestOnSubscribePod_RejectsStoppedPod verifies that subscribe is silently ignored
// for pods in terminal states.
func TestOnSubscribePod_RejectsStoppedPod(t *testing.T) {
	for _, status := range []string{PodStatusStopped, PodStatusFailed} {
		t.Run(status, func(t *testing.T) {
			store := NewInMemoryPodStore()
			mockConn := client.NewMockConnection()
			runner := &Runner{cfg: &config.Config{}}
			handler := NewRunnerMessageHandler(runner, store, mockConn)

			pod := &Pod{PodKey: "pod-terminal", Status: status}
			store.Put(pod.PodKey, pod)

			err := handler.OnSubscribePod(client.SubscribePodRequest{
				PodKey:      pod.PodKey,
				RelayURL:    "wss://relay.example.com",
				RunnerToken: "token-123",
			})

			if err != nil {
				t.Errorf("expected nil error for %s pod, got: %v", status, err)
			}
			if pod.GetRelayClient() != nil {
				t.Errorf("no relay client should be set for %s pod", status)
			}
		})
	}
}

// TestOnSubscribePod_AllowsInitializingPod verifies that subscribe is accepted
// for pods still in initializing state (backend may send subscribe before process starts).
func TestOnSubscribePod_AllowsInitializingPod(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()
	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, mockConn)

	handler.relayClientFactory = func(url, podKey, token string, logger *slog.Logger) relay.RelayClient {
		mc := relay.NewMockClient(url)
		return mc
	}

	pod := &Pod{
		PodKey: "pod-init", Status: PodStatusInitializing,
		Relay: &orderedLifecycleRelay{events: &lifecycleEvents{}},
	}
	store.Put(pod.PodKey, pod)

	err := handler.OnSubscribePod(client.SubscribePodRequest{
		PodKey:      pod.PodKey,
		RelayURL:    "wss://relay.example.com",
		RunnerToken: "token-123",
	})

	// Should proceed (Connect will succeed with mock), not be rejected
	if err != nil {
		t.Errorf("initializing pod should accept subscribe, got: %v", err)
	}
}
