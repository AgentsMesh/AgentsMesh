package runner

import (
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

func TestOnSubscribePodRejectsInboundFrameBeforeCandidateInstall(t *testing.T) {
	const podKey = "candidate-inbound-before-install"
	store := NewInMemoryPodStore()
	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, client.NewMockConnection())

	inputs := make(chan string, 2)
	io := &stubPodIO{onSendInput: func(text string) error {
		inputs <- text
		return nil
	}}
	pod := &Pod{PodKey: podKey, Status: PodStatusRunning, IO: io}
	pod.Relay = NewPTYPodRelay(podKey, io, nil, nil)
	store.Put(podKey, pod)

	candidate := &eagerStartRelayClient{
		MockClient: relay.NewMockClient("wss://relay.example.com"),
		msgType:    relay.MsgTypeInput,
		payload:    []byte("before-install"),
	}
	handler.relayClientFactory = func(string, string, string, *slog.Logger) relay.RelayClient {
		return candidate
	}

	if err := handler.OnSubscribePod(client.SubscribePodRequest{
		PodKey: podKey, RelayURL: candidate.GetRelayURL(), RunnerToken: "token",
	}); err != nil {
		t.Fatalf("OnSubscribePod failed: %v", err)
	}
	select {
	case got := <-inputs:
		t.Fatalf("candidate executed inbound frame before ownership commit: %q", got)
	default:
	}

	candidate.SimulateMessage(relay.MsgTypeInput, []byte("after-install"))
	select {
	case got := <-inputs:
		if got != "after-install" {
			t.Fatalf("installed owner delivered %q, want after-install", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("installed relay owner did not execute inbound frame")
	}
	pod.TeardownRelayTransports(nil)
}

func TestOnSubscribePodSlowConnectCannotInstallAfterStop(t *testing.T) {
	pod, candidate, done := startGatedSubscribe(t)
	pod.SetStatus(PodStatusStopped)
	pod.DisconnectRelay()
	finishGatedSubscribe(t, pod, candidate, done)
}

func TestOnSubscribePodSlowConnectCannotCrossRuntimeEpoch(t *testing.T) {
	pod, candidate, done := startGatedSubscribe(t)
	transition := pod.BeginRelayRuntimeTransition()
	if !pod.EndRelayRuntimeTransition(transition) {
		t.Fatal("failed to complete test runtime transition")
	}
	finishGatedSubscribe(t, pod, candidate, done)
}

func TestOnSubscribePodInvalidatedIntentCannotInstall(t *testing.T) {
	store := NewInMemoryPodStore()
	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, client.NewMockConnection())
	candidate := &gatedConnectRelayClient{
		MockClient: relay.NewMockClient("wss://relay.example.com"),
		started:    make(chan struct{}),
		release:    make(chan struct{}),
	}
	handler.relayClientFactory = func(string, string, string, *slog.Logger) relay.RelayClient {
		return candidate
	}
	pod := newRelayReadyTestPod("intent-invalidated", PodStatusRunning)
	store.Put(pod.PodKey, pod)

	var current atomic.Bool
	current.Store(true)
	done := make(chan error, 1)
	go func() {
		done <- handler.OnSubscribePod(client.SubscribePodRequest{
			PodKey: pod.PodKey, RelayURL: candidate.GetRelayURL(), RunnerToken: "stale-token",
			IntentValid: current.Load,
		})
	}()
	waitSignalChannel(t, candidate.started, "subscribe did not enter Connect")
	current.Store(false)
	pod.TeardownRelayTransports(nil)
	close(candidate.release)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("OnSubscribePod returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("invalidated subscribe did not finish")
	}
	if !candidate.StopCalled {
		t.Fatal("invalidated relay candidate was not stopped")
	}
	if got := pod.GetRelayClient(); got != nil {
		t.Fatalf("invalidated relay candidate was installed: %T", got)
	}
}

func waitSignalChannel(t *testing.T, ch <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal(message)
	}
}

func TestClearRelayClientIfDoesNotClearReplacement(t *testing.T) {
	pod := &Pod{
		PodKey: "relay-owner", Status: PodStatusRunning,
		Relay: &orderedLifecycleRelay{events: &lifecycleEvents{}},
	}
	oldClient := relay.NewMockClient("wss://old.example.com")
	newClient := relay.NewMockClient("wss://new.example.com")
	oldClient.SetConnected(true)
	newClient.SetConnected(true)

	oldGeneration := prepareRelayHandlerGeneration(t, pod, oldClient)
	if !pod.TryInstallRelayClient(oldClient, oldGeneration) {
		t.Fatal("failed to install old client")
	}
	if !pod.ClearRelayClientIf(oldClient) {
		t.Fatal("old owner failed to clear itself")
	}
	newGeneration := prepareRelayHandlerGeneration(t, pod, newClient)
	if !pod.TryInstallRelayClient(newClient, newGeneration) {
		t.Fatal("failed to install replacement client")
	}
	epochBeforeStaleClear, _ := pod.RelayLifecycle()
	if pod.ClearRelayClientIf(oldClient) {
		t.Fatal("stale owner cleared replacement")
	}
	epochAfterStaleClear, _ := pod.RelayLifecycle()
	if epochAfterStaleClear != epochBeforeStaleClear {
		t.Fatal("stale clear advanced the active relay generation")
	}
	if got := pod.GetRelayClient(); got != newClient {
		t.Fatal("replacement relay client was not preserved")
	}
}

func startGatedSubscribe(t *testing.T) (*Pod, *gatedConnectRelayClient, <-chan error) {
	t.Helper()
	store := NewInMemoryPodStore()
	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, client.NewMockConnection())
	candidate := &gatedConnectRelayClient{
		MockClient: relay.NewMockClient("wss://relay.example.com"),
		started:    make(chan struct{}),
		release:    make(chan struct{}),
	}
	handler.relayClientFactory = func(string, string, string, *slog.Logger) relay.RelayClient {
		return candidate
	}
	pod := &Pod{
		PodKey: "slow-subscribe", Status: PodStatusRunning,
		Relay: &orderedLifecycleRelay{events: &lifecycleEvents{}},
	}
	store.Put(pod.PodKey, pod)
	done := make(chan error, 1)
	go func() {
		done <- handler.OnSubscribePod(client.SubscribePodRequest{
			PodKey: pod.PodKey, RelayURL: "wss://relay.example.com", RunnerToken: "token",
		})
	}()
	select {
	case <-candidate.started:
	case <-time.After(2 * time.Second):
		t.Fatal("subscribe did not enter Connect")
	}
	return pod, candidate, done
}

func finishGatedSubscribe(
	t *testing.T,
	pod *Pod,
	candidate *gatedConnectRelayClient,
	done <-chan error,
) {
	t.Helper()
	close(candidate.release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("OnSubscribePod returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("subscribe did not finish")
	}
	if !candidate.StopCalled {
		t.Fatal("stale candidate was not stopped")
	}
	if got := pod.GetRelayClient(); got != nil {
		t.Fatalf("pod retained ghost relay client: %T", got)
	}
}
