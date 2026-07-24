package runner

import (
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

func TestOnUnsubscribePodLinearizesLocalGenerationWithRelayEpoch(t *testing.T) {
	const podKey = "unsubscribe-local-race"
	store := NewInMemoryPodStore()
	baseRunner := &Runner{cfg: &config.Config{}}
	local := newGatedUnregisterLocalBroker()
	defer func() {
		select {
		case <-local.unregisterRelease:
		default:
			close(local.unregisterRelease)
		}
	}()
	handler := NewRunnerMessageHandler(
		gatedLocalHandlerContext{MessageHandlerContext: baseRunner, local: local},
		store,
		client.NewMockConnection(),
	)

	components := &PTYComponents{}
	pod := &Pod{PodKey: podKey, Status: PodStatusRunning}
	pod.Relay = NewPTYPodRelay(podKey, nil, components, local)
	store.Put(podKey, pod)

	cloud := relay.NewMockClient("wss://relay.example.com")
	cloud.SetConnected(true)
	initialRequest := client.SubscribePodRequest{
		PodKey: podKey, RelayURL: "wss://relay.example.com",
		RunnerToken: "initial-runner-token", LocalToken: "initial-local-token",
	}
	initialPrepareTicket, ok := pod.RelayLifecycle()
	initialGeneration, prepared := pod.WithRelayHandlerGeneration(initialPrepareTicket, cloud, func(modeRelay PodRelay, guard RelayInboundGuard) {
		handler.registerLocalRelayPod(initialRequest)
		handler.setupRelayClientHandlers(cloud, pod, modeRelay, guard, initialRequest)
	})
	if !ok || !prepared {
		t.Fatal("failed to prepare initial local relay generation")
	}
	if !pod.TryInstallRelayClient(cloud, initialGeneration) {
		t.Fatal("failed to install initial relay owner")
	}
	if tokens, handlers, requestHandlers := local.registrationCounts(podKey); tokens != 1 || handlers != 2 || requestHandlers != 1 {
		t.Fatalf("initial local generation was not installed: tokens=%d handlers=%d request_handlers=%d",
			tokens, handlers, requestHandlers)
	}
	staleTicket, ok := pod.RelayLifecycle()
	if !ok {
		t.Fatal("failed to acquire subscribe lifecycle ticket")
	}

	type teardownObservation struct {
		epoch uint64
		owner relay.RelayClient
	}
	observed := make(chan teardownObservation, 1)
	local.onUnregister = func() {
		observed <- teardownObservation{
			epoch: pod.relayEpoch,
			owner: pod.GetRelayClient(),
		}
	}

	unsubscribeDone := make(chan error, 1)
	go func() {
		unsubscribeDone <- handler.OnUnsubscribePod(client.UnsubscribePodRequest{PodKey: podKey})
	}()

	select {
	case <-local.unregisterStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("unsubscribe did not enter local generation teardown")
	}
	observation := <-observed
	if observation.epoch <= uint64(staleTicket) {
		t.Fatalf("local generation removed before relay epoch invalidation: epoch=%d ticket=%d",
			observation.epoch, staleTicket)
	}
	if observation.owner != nil {
		t.Fatal("cloud relay owner was still attached during local generation teardown")
	}
	if !cloud.IsConnected() {
		t.Fatal("cloud client stopped while relay transition lock was still held")
	}

	request := client.SubscribePodRequest{
		PodKey: podKey, RelayURL: "wss://relay.example.com",
		RunnerToken: "runner-token", LocalToken: "stale-local-token",
	}
	candidate := relay.NewMockClient(request.RelayURL)
	prepareStarted := make(chan struct{})
	prepareDone := make(chan bool, 1)
	go func() {
		close(prepareStarted)
		_, prepared := pod.WithRelayHandlerGeneration(staleTicket, candidate, func(modeRelay PodRelay, guard RelayInboundGuard) {
			handler.registerLocalRelayPod(request)
			handler.setupRelayClientHandlers(candidate, pod, modeRelay, guard, request)
		})
		if !prepared {
			candidate.Stop()
		}
		prepareDone <- prepared
	}()
	<-prepareStarted

	select {
	case prepared := <-prepareDone:
		t.Fatalf("stale subscribe crossed in-progress teardown: prepared=%v", prepared)
	default:
	}

	close(local.unregisterRelease)
	select {
	case err := <-unsubscribeDone:
		if err != nil {
			t.Fatalf("OnUnsubscribePod returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("unsubscribe did not finish")
	}
	select {
	case prepared := <-prepareDone:
		if prepared {
			t.Fatal("stale subscribe ticket recreated the local relay generation")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("stale subscribe continuation did not finish")
	}

	if tokens, handlers, requestHandlers := local.registrationCounts(podKey); tokens != 0 || handlers != 0 || requestHandlers != 0 {
		t.Fatalf("stale local state survived unsubscribe: tokens=%d handlers=%d request_handlers=%d",
			tokens, handlers, requestHandlers)
	}
	if !candidate.StopCalled {
		t.Fatal("rejected stale relay candidate was not stopped")
	}
	if !cloud.StopCalled {
		t.Fatal("detached cloud relay owner was not stopped")
	}

	replacementTicket, accepting := pod.RelayLifecycle()
	if !accepting {
		t.Fatal("unsubscribe permanently blocked a later replacement subscribe")
	}
	replacementRequest := request
	replacementRequest.LocalToken = "replacement-local-token"
	replacement := relay.NewMockClient(replacementRequest.RelayURL)
	_, replacementPrepared := pod.WithRelayHandlerGeneration(replacementTicket, replacement, func(modeRelay PodRelay, guard RelayInboundGuard) {
		handler.registerLocalRelayPod(replacementRequest)
		handler.setupRelayClientHandlers(replacement, pod, modeRelay, guard, replacementRequest)
	})
	if !replacementPrepared {
		t.Fatal("fresh replacement subscribe was rejected after unsubscribe")
	}
	if tokens, handlers, requestHandlers := local.registrationCounts(podKey); tokens != 1 || handlers != 2 || requestHandlers != 1 {
		t.Fatalf("replacement local generation was not rebuilt: tokens=%d handlers=%d request_handlers=%d",
			tokens, handlers, requestHandlers)
	}
}

func TestRelayInboundGenerationRejectsCapturedHandlersAcrossTeardownAndReplacement(t *testing.T) {
	const podKey = "captured-inbound-generation"
	local := newGatedUnregisterLocalBroker()
	defer func() {
		select {
		case <-local.unregisterRelease:
		default:
			close(local.unregisterRelease)
		}
	}()

	activeEntered := make(chan struct{})
	activeRelease := make(chan struct{})
	defer func() {
		select {
		case <-activeRelease:
		default:
			close(activeRelease)
		}
	}()
	oldInputs := make(chan string, 8)
	oldIO := &stubPodIO{onSendInput: func(text string) error {
		if text == "in-flight" {
			close(activeEntered)
			<-activeRelease
		}
		oldInputs <- text
		return nil
	}}
	pod := &Pod{PodKey: podKey, Status: PodStatusRunning, IO: oldIO}
	oldRelay := NewPTYPodRelay(podKey, oldIO, nil, local)
	pod.Relay = oldRelay
	oldCloud := relay.NewMockClient("wss://old.example.com")
	oldCloud.SetConnected(true)

	oldTicket, ok := pod.RelayLifecycle()
	oldGeneration, prepared := pod.WithRelayHandlerGeneration(oldTicket, oldCloud, func(modeRelay PodRelay, guard RelayInboundGuard) {
		local.RegisterPod(podKey, "old-local-token")
		modeRelay.SetupHandlers(oldCloud, guard)
	})
	if !ok || !prepared {
		t.Fatal("failed to prepare old inbound generation")
	}
	if !pod.TryInstallRelayClient(oldCloud, oldGeneration) {
		t.Fatal("failed to install old cloud owner")
	}
	oldLocal := local.messageHandler(podKey, relay.MsgTypeInput)
	if oldLocal == nil {
		t.Fatal("old local input handler was not registered")
	}

	activeDone := make(chan struct{})
	go func() {
		oldLocal([]byte("in-flight"))
		close(activeDone)
	}()
	select {
	case <-activeEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("old local callback did not acquire its activity lease")
	}

	teardownDone := make(chan struct{})
	go func() {
		pod.TeardownRelayTransports(local)
		close(teardownDone)
	}()
	deadline := time.Now().Add(2 * time.Second)
	for {
		pod.relayMu.RLock()
		invalidated := pod.relayTransportBlock && pod.RelayClient == nil && !pod.relayLocalActive
		pod.relayMu.RUnlock()
		if invalidated {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("teardown did not invalidate relay generations")
		}
		time.Sleep(time.Millisecond)
	}

	select {
	case <-local.unregisterStarted:
		t.Fatal("teardown unregistered local handlers before the active callback drained")
	default:
	}
	if _, accepting := pod.RelayLifecycle(); accepting {
		t.Fatal("new relay lifecycle crossed the in-progress transport drain")
	}

	rejectedDone := make(chan struct{})
	go func() {
		oldCloud.SimulateMessage(relay.MsgTypeInput, []byte("stale-cloud-during-teardown"))
		oldLocal([]byte("stale-local-during-teardown"))
		close(rejectedDone)
	}()
	select {
	case <-rejectedDone:
	case <-time.After(2 * time.Second):
		t.Fatal("callbacks arriving after invalidation did not reject immediately")
	}
	select {
	case got := <-oldInputs:
		t.Fatalf("old local handler executed after epoch invalidation: %q", got)
	default:
	}

	close(activeRelease)
	select {
	case <-activeDone:
	case <-time.After(2 * time.Second):
		t.Fatal("admitted callback did not finish after release")
	}
	select {
	case got := <-oldInputs:
		if got != "in-flight" {
			t.Fatalf("admitted callback delivered %q, want in-flight", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("admitted callback did not execute against the old IO")
	}

	select {
	case <-local.unregisterStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("teardown did not continue after the activity lease drained")
	}
	close(local.unregisterRelease)
	select {
	case <-teardownDone:
	case <-time.After(2 * time.Second):
		t.Fatal("teardown did not finish")
	}

	transition := pod.BeginRelayRuntimeTransition()
	newInputs := make(chan string, 4)
	newIO := &stubPodIO{onSendInput: func(text string) error {
		newInputs <- text
		return nil
	}}
	newRelay := NewPTYPodRelay(podKey, newIO, nil, local)
	if !pod.installRuntimeDuringTransition(transition, PodRuntime{IO: newIO, Relay: newRelay}) {
		t.Fatal("failed to publish replacement runtime")
	}
	if !pod.EndRelayRuntimeTransition(transition) {
		t.Fatal("failed to commit replacement runtime")
	}

	oldCloud.SimulateMessage(relay.MsgTypeInput, []byte("old-cloud-after-replacement"))
	oldLocal([]byte("old-local-after-replacement"))
	select {
	case got := <-newInputs:
		t.Fatalf("old generation operated replacement runtime: %q", got)
	default:
	}

	newCloud := relay.NewMockClient("wss://new.example.com")
	newCloud.SetConnected(true)
	newTicket, accepting := pod.RelayLifecycle()
	newGeneration, prepared := pod.WithRelayHandlerGeneration(newTicket, newCloud, func(modeRelay PodRelay, guard RelayInboundGuard) {
		local.RegisterPod(podKey, "new-local-token")
		modeRelay.SetupHandlers(newCloud, guard)
	})
	if !accepting || !prepared {
		t.Fatal("failed to prepare replacement inbound generation")
	}
	if !pod.TryInstallRelayClient(newCloud, newGeneration) {
		t.Fatal("failed to install replacement cloud owner")
	}
	newLocal := local.messageHandler(podKey, relay.MsgTypeInput)
	if newLocal == nil {
		t.Fatal("replacement local input handler was not registered")
	}
	newCloud.SimulateMessage(relay.MsgTypeInput, []byte("new-cloud"))
	newLocal([]byte("new-local"))

	for _, want := range []string{"new-cloud", "new-local"} {
		select {
		case got := <-newInputs:
			if got != want {
				t.Fatalf("replacement handler delivered %q, want %q", got, want)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("replacement handler did not deliver %q", want)
		}
	}
}
