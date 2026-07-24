package runner

import (
	"errors"
	"log/slog"
	"testing"

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
