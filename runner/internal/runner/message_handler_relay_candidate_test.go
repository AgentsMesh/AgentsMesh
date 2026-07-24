package runner

import (
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

func TestOnSubscribePodConnectFailureStopsCandidate(t *testing.T) {
	store := NewInMemoryPodStore()
	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, client.NewMockConnection())
	candidate := relay.NewMockClient("wss://relay.example.com")
	candidate.ConnectError = errors.New("dial failed")
	handler.relayClientFactory = func(string, string, string, *slog.Logger) relay.RelayClient {
		return candidate
	}
	pod := newRelayReadyTestPod("connect-failure", PodStatusRunning)
	store.Put(pod.PodKey, pod)

	err := handler.OnSubscribePod(client.SubscribePodRequest{
		PodKey: pod.PodKey, RelayURL: "wss://relay.example.com", RunnerToken: "token",
	})
	if err == nil {
		t.Fatal("expected relay connect error")
	}
	if !candidate.StopCalled {
		t.Fatal("failed relay candidate was not stopped")
	}
	if got := pod.GetRelayClient(); got != nil {
		t.Fatalf("pod retained failed relay candidate: %T", got)
	}
}

func TestOnSubscribePodRewritesRelayURL(t *testing.T) {
	store := NewInMemoryPodStore()
	runner := &Runner{cfg: &config.Config{RelayBaseURL: "ws://127.0.0.1:19001"}}
	handler := NewRunnerMessageHandler(runner, store, client.NewMockConnection())
	pod := newRelayReadyTestPod("rewrite", PodStatusRunning)
	store.Put(pod.PodKey, pod)
	candidate := relay.NewMockClient("ws://127.0.0.1:19001/pod/rewrite")
	var factoryURL string
	handler.relayClientFactory = func(relayURL, _, _ string, _ *slog.Logger) relay.RelayClient {
		factoryURL = relayURL
		return candidate
	}

	err := handler.OnSubscribePod(client.SubscribePodRequest{
		PodKey: pod.PodKey, RelayURL: "wss://remote.example/pod/rewrite", RunnerToken: "token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if factoryURL != "ws://127.0.0.1:19001/pod/rewrite" {
		t.Fatalf("relay factory URL = %q", factoryURL)
	}
	pod.TeardownRelayTransports(nil)
}

func TestSubscribePodRuntimeIntentInvalidationStages(t *testing.T) {
	t.Run("before first ticket", func(t *testing.T) {
		handler, pod, candidate := newCandidateStageTest(t, "first")
		if err := handler.subscribePodRuntime(pod, client.SubscribePodRequest{
			PodKey: pod.PodKey, RelayURL: candidate.GetRelayURL(),
		}, func() bool { return false }); err != nil {
			t.Fatal(err)
		}
		if candidate.ConnectCalled {
			t.Fatal("invalid intent reached Connect")
		}
	})

	t.Run("after old client teardown", func(t *testing.T) {
		handler, pod, candidate := newCandidateStageTest(t, "after-teardown")
		old := relay.NewMockClient("wss://old.example")
		old.SetConnected(true)
		pod.SetRelayClient(old)
		calls := 0
		current := func() bool {
			calls++
			return calls == 1
		}
		if err := handler.subscribePodRuntime(pod, client.SubscribePodRequest{
			PodKey: pod.PodKey, RelayURL: candidate.GetRelayURL(),
		}, current); err != nil {
			t.Fatal(err)
		}
		if !old.StopCalled || candidate.ConnectCalled {
			t.Fatal("second intent fence did not stop after old transport teardown")
		}
	})

	t.Run("after candidate preparation", func(t *testing.T) {
		handler, pod, candidate := newCandidateStageTest(t, "after-prepare")
		calls := 0
		current := func() bool {
			calls++
			return calls <= 2
		}
		if err := handler.subscribePodRuntime(pod, client.SubscribePodRequest{
			PodKey: pod.PodKey, RelayURL: candidate.GetRelayURL(),
		}, current); err != nil {
			t.Fatal(err)
		}
		if candidate.ConnectCalled || !candidate.StopCalled {
			t.Fatal("prepared stale candidate was not retired before Connect")
		}
	})
}

func TestSubscribePodRuntimeCandidatePreparationAndStartFailures(t *testing.T) {
	t.Run("runtime epoch changes during factory", func(t *testing.T) {
		handler, pod, candidate := newCandidateStageTest(t, "prepare")
		handler.relayClientFactory = func(string, string, string, *slog.Logger) relay.RelayClient {
			pod.BeginRelayRuntimeTransition()
			return candidate
		}
		if err := handler.subscribePodRuntime(pod, client.SubscribePodRequest{
			PodKey: pod.PodKey, RelayURL: candidate.GetRelayURL(),
		}, nil); err != nil {
			t.Fatal(err)
		}
		if !candidate.StopCalled || candidate.ConnectCalled {
			t.Fatal("stale runtime ticket did not reject candidate")
		}
	})

	t.Run("client start rejected", func(t *testing.T) {
		handler, pod, candidate := newCandidateStageTest(t, "start")
		candidate.StartResult = false
		err := handler.subscribePodRuntime(pod, client.SubscribePodRequest{
			PodKey: pod.PodKey, RelayURL: candidate.GetRelayURL(),
		}, nil)
		if err == nil || !candidate.ConnectCalled || !candidate.StartCalled || !candidate.StopCalled {
			t.Fatalf("start rejection was not fully retired: err=%v candidate=%+v", err, candidate)
		}
	})
}

func newCandidateStageTest(t *testing.T, suffix string) (*RunnerMessageHandler, *Pod, *relay.MockClient) {
	t.Helper()
	store := NewInMemoryPodStore()
	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, client.NewMockConnection())
	pod := newRelayReadyTestPod("stage-"+suffix, PodStatusRunning)
	store.Put(pod.PodKey, pod)
	candidate := relay.NewMockClient("wss://relay.example/" + suffix)
	handler.relayClientFactory = func(string, string, string, *slog.Logger) relay.RelayClient {
		return candidate
	}
	return handler, pod, candidate
}

type gatedConnectRelayClient struct {
	*relay.MockClient
	started chan struct{}
	release chan struct{}
}

type eagerStartRelayClient struct {
	*relay.MockClient
	msgType byte
	payload []byte
}

func (c *eagerStartRelayClient) Start() bool {
	started := c.MockClient.Start()
	if started {
		c.SimulateMessage(c.msgType, c.payload)
	}
	return started
}

func newRelayReadyTestPod(podKey, status string) *Pod {
	return &Pod{
		PodKey: podKey, Status: status,
		Relay: &orderedLifecycleRelay{events: &lifecycleEvents{}},
	}
}

func prepareRelayHandlerGeneration(
	t *testing.T,
	pod *Pod,
	client relay.RelayClient,
) RelayHandlerGeneration {
	t.Helper()
	ticket, accepting := pod.RelayLifecycle()
	if !accepting {
		t.Fatal("pod did not accept relay handler generation")
	}
	generation, prepared := pod.WithRelayHandlerGeneration(
		ticket,
		client,
		func(PodRelay, RelayInboundGuard) {},
	)
	if !prepared {
		t.Fatal("failed to prepare relay handler generation")
	}
	return generation
}

func (c *gatedConnectRelayClient) Connect() error {
	close(c.started)
	<-c.release
	return c.MockClient.Connect()
}

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
