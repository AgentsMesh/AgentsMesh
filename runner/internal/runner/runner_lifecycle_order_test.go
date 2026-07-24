package runner

import (
	"context"
	"sync"
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

type immediateExitIO struct {
	*orderedLifecycleIO
	exitHandler func(int)
}

func (o *immediateExitIO) SetExitHandler(handler func(int)) { o.exitHandler = handler }
func (o *immediateExitIO) Start() error {
	go o.exitHandler(0)
	return nil
}

type lifecycleEvents struct {
	mu     sync.Mutex
	events []string
}

func (e *lifecycleEvents) add(event string) {
	e.mu.Lock()
	e.events = append(e.events, event)
	e.mu.Unlock()
}

func (e *lifecycleEvents) snapshot() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.events...)
}

type orderedLifecycleIO struct{ events *lifecycleEvents }

func (o *orderedLifecycleIO) Mode() string                              { return InteractionModePTY }
func (o *orderedLifecycleIO) SendInput(string) error                    { return nil }
func (o *orderedLifecycleIO) GetSnapshot(int) (string, error)           { return "", nil }
func (o *orderedLifecycleIO) GetAgentStatus() string                    { return "idle" }
func (o *orderedLifecycleIO) SubscribeStateChange(string, func(string)) {}
func (o *orderedLifecycleIO) UnsubscribeStateChange(string)             {}
func (o *orderedLifecycleIO) GetPID() int                               { return 0 }
func (o *orderedLifecycleIO) Start() error                              { return nil }
func (o *orderedLifecycleIO) Stop()                                     { o.events.add("stop") }
func (o *orderedLifecycleIO) Teardown() string                          { o.events.add("teardown"); return "" }
func (o *orderedLifecycleIO) SetExitHandler(func(int))                  {}
func (o *orderedLifecycleIO) SetIOErrorHandler(func(error))             {}
func (o *orderedLifecycleIO) Detach()                                   { o.events.add("detach") }

type blockingTeardownIO struct {
	*orderedLifecycleIO
	started chan struct{}
	release chan struct{}
}

func (o *blockingTeardownIO) Teardown() string {
	close(o.started)
	<-o.release
	o.events.add("teardown")
	return ""
}

type orderedLifecycleRelay struct{ events *lifecycleEvents }

func (r *orderedLifecycleRelay) SetupHandlers(relay.RelayClient, RelayInboundGuard) {}
func (r *orderedLifecycleRelay) SendSnapshot(relay.RelayClient)                     {}
func (r *orderedLifecycleRelay) RecoverSnapshot(relay.RelayClient)                  {}
func (r *orderedLifecycleRelay) OnRelayConnected(relay.RelayClient)                 { r.events.add("relay-connect") }
func (r *orderedLifecycleRelay) OnRelayDisconnected(relay.RelayClient) {
	r.events.add("relay-disconnect")
}
func (r *orderedLifecycleRelay) BroadcastEvent(relay.RelayClient, byte, []byte) {}

type orderedRelayClient struct {
	*relay.MockClient
	events *lifecycleEvents
}

func (c *orderedRelayClient) Stop() {
	c.events.add("client-stop")
	c.MockClient.Stop()
}

type orderedLocalBroker struct {
	events *lifecycleEvents
}

func (b *orderedLocalBroker) RegisterPod(string, string)                           {}
func (b *orderedLocalBroker) UnregisterPod(string)                                 { b.events.add("local-unregister") }
func (b *orderedLocalBroker) SetMessageHandler(string, byte, func([]byte))         {}
func (b *orderedLocalBroker) SetRequestHandler(string, byte, relay.RequestHandler) {}
func (b *orderedLocalBroker) Send(string, byte, []byte) error                      { return nil }
func (b *orderedLocalBroker) Flush(context.Context, string) error                  { return nil }
func (b *orderedLocalBroker) IsPodConnected(string) bool                           { return false }
func (b *orderedLocalBroker) URL() string                                          { return "ws://local" }

type localOnlyHandlerContext struct {
	MessageHandlerContext
	local LocalRelayBroker
}

func (c localOnlyHandlerContext) GetLocalRelayServer() LocalRelayBroker { return c.local }

func TestTerminateCleanupOrdersProducerDrainAndSinks(t *testing.T) {
	events := &lifecycleEvents{}
	store := NewInMemoryPodStore()
	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, client.NewMockConnection())
	pod := orderedLifecyclePod("terminate-order", events)
	store.Put(pod.PodKey, pod)

	handler.cleanupPodExit(pod.PodKey, -1, true)
	assertLifecycleEvents(t, events, "stop", "teardown", "relay-disconnect", "client-stop")
}

func TestFastExitCannotOvertakePodCreatedCommit(t *testing.T) {
	events := &lifecycleEvents{}
	store := NewInMemoryPodStore()
	conn := client.NewMockConnection()
	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, conn)
	io := &immediateExitIO{orderedLifecycleIO: &orderedLifecycleIO{events: events}}
	pod := &Pod{
		ID:              "fast-exit",
		PodKey:          "fast-exit",
		InteractionMode: InteractionModePTY,
		Status:          PodStatusInitializing,
	}
	pod.installRuntime(PodRuntime{
		IO:    io,
		Relay: &orderedLifecycleRelay{events: events},
	})
	store.Put(pod.PodKey, pod)

	if err := handler.wireAndStartPTYPod(pod, &runnerv1.CreatePodCommand{
		PodKey: "fast-exit",
	}, 80, 24); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, exists := store.Get(pod.PodKey); !exists {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("fast exit did not finalize")
		}
		time.Sleep(time.Millisecond)
	}

	created, terminated := -1, -1
	for i, event := range conn.GetEvents() {
		switch event.Type {
		case client.MsgTypePodCreated:
			created = i
		case client.MsgTypePodTerminated:
			terminated = i
		}
	}
	if created < 0 || terminated < 0 || created >= terminated {
		t.Fatalf("event order created=%d terminated=%d", created, terminated)
	}
}

func TestPerpetualRetirementDrainsBeforeClosingViewers(t *testing.T) {
	events := &lifecycleEvents{}
	broker := &orderedLocalBroker{events: events}
	handler := &RunnerMessageHandler{runner: localOnlyHandlerContext{local: broker}}
	pod := orderedLifecyclePod("perpetual-order", events)

	handler.retirePerpetualRuntime(pod)
	assertLifecycleEvents(t, events, "teardown", "relay-disconnect", "local-unregister", "client-stop")
}

func TestRunnerShutdownOrdersPTYDetachBeforeDrain(t *testing.T) {
	events := &lifecycleEvents{}
	runner, _ := NewTestRunner(t)
	pod := orderedLifecyclePod("shutdown-pty", events)
	runner.podStore.Put(pod.PodKey, pod)

	runner.stopAllPods()
	assertLifecycleEvents(t, events, "detach", "teardown", "relay-disconnect", "client-stop")
}

func TestRunnerShutdownStopsACPBeforeTeardown(t *testing.T) {
	events := &lifecycleEvents{}
	runner, _ := NewTestRunner(t)
	pod := orderedLifecyclePod("shutdown-acp", events)
	pod.InteractionMode = InteractionModeACP
	runner.podStore.Put(pod.PodKey, pod)

	runner.stopAllPods()
	assertLifecycleEvents(t, events, "stop", "teardown", "relay-disconnect", "client-stop")
}

func TestPerpetualRestartSerializesConcurrentTerminate(t *testing.T) {
	events := &lifecycleEvents{}
	store := NewInMemoryPodStore()
	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, client.NewMockConnection())
	pod := orderedLifecyclePod("perpetual-race", events)
	pod.Perpetual = true
	blockingIO := &blockingTeardownIO{
		orderedLifecycleIO: &orderedLifecycleIO{events: events},
		started:            make(chan struct{}),
		release:            make(chan struct{}),
	}
	pod.IO = blockingIO
	store.Put(pod.PodKey, pod)

	restartDone := make(chan struct{})
	go func() {
		handler.cleanupPodExit(pod.PodKey, 0, false)
		close(restartDone)
	}()
	select {
	case <-blockingIO.started:
	case <-time.After(2 * time.Second):
		t.Fatal("perpetual retirement did not start")
	}

	terminateDone := make(chan error, 1)
	go func() {
		terminateDone <- handler.OnTerminatePod(client.TerminatePodRequest{PodKey: pod.PodKey})
	}()
	select {
	case <-terminateDone:
		close(blockingIO.release)
		t.Fatal("terminate crossed an in-progress perpetual lifecycle")
	case <-time.After(50 * time.Millisecond):
	}
	close(blockingIO.release)

	select {
	case <-restartDone:
	case <-time.After(2 * time.Second):
		t.Fatal("failed perpetual restart cleanup did not finish")
	}
	select {
	case err := <-terminateDone:
		if err != nil {
			t.Fatalf("concurrent terminate returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("terminate did not finish after lifecycle owner released")
	}
}

func orderedLifecyclePod(podKey string, events *lifecycleEvents) *Pod {
	client := &orderedRelayClient{MockClient: relay.NewMockClient("wss://relay"), events: events}
	client.SetConnected(true)
	pod := &Pod{
		PodKey: podKey, Status: PodStatusRunning, InteractionMode: InteractionModePTY,
		IO: &orderedLifecycleIO{events: events}, Relay: &orderedLifecycleRelay{events: events},
	}
	pod.SetRelayClient(client)
	return pod
}

func assertLifecycleEvents(t *testing.T, events *lifecycleEvents, want ...string) {
	t.Helper()
	got := events.snapshot()
	if len(got) != len(want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("events = %v, want %v", got, want)
		}
	}
}
