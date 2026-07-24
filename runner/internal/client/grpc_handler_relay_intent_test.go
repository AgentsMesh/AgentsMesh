package client

import (
	"errors"
	"sync"
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

type relayIntentSubscription struct {
	req         SubscribePodRequest
	validAtCall bool
}

type relayIntentDispatchHandler struct {
	*mockHandler
	createStarted      chan struct{}
	createRelease      <-chan struct{}
	createErr          error
	createOnce         sync.Once
	subscribeCalls     chan relayIntentSubscription
	subscribeRelease   <-chan struct{}
	subscribeErr       error
	unsubscribeCalls   chan UnsubscribePodRequest
	unsubscribeRelease <-chan struct{}
}

func newRelayIntentDispatchHandler() *relayIntentDispatchHandler {
	return &relayIntentDispatchHandler{
		mockHandler:      &mockHandler{},
		createStarted:    make(chan struct{}),
		subscribeCalls:   make(chan relayIntentSubscription, 8),
		unsubscribeCalls: make(chan UnsubscribePodRequest, 8),
	}
}

func (h *relayIntentDispatchHandler) OnCreatePod(*runnerv1.CreatePodCommand) error {
	h.createOnce.Do(func() { close(h.createStarted) })
	if h.createRelease != nil {
		<-h.createRelease
	}
	return h.createErr
}

func (h *relayIntentDispatchHandler) OnSubscribePod(req SubscribePodRequest) error {
	valid := req.IntentValid == nil || req.IntentValid()
	h.subscribeCalls <- relayIntentSubscription{req: req, validAtCall: valid}
	if h.subscribeRelease != nil {
		<-h.subscribeRelease
	}
	return h.subscribeErr
}

func TestHandleSubscribeRequestDefensivePaths(t *testing.T) {
	conn := newTestConnection()
	conn.handler = nil
	conn.handleSubscribeRequest(SubscribePodRequest{PodKey: "no-handler"})

	handler := newRelayIntentDispatchHandler()
	handler.subscribeErr = errors.New("subscribe failed")
	conn.handler = handler
	conn.handleSubscribeRequest(SubscribePodRequest{PodKey: "error"})
	call := waitSubscription(t, handler.subscribeCalls)
	if call.req.PodKey != "error" {
		t.Fatalf("subscribe handler received pod %q", call.req.PodKey)
	}
}

func TestRelayIntentAdmissionDirectUnsubscribeWithoutIntent(t *testing.T) {
	handler := newRelayIntentDispatchHandler()
	conn := newTestConnection()
	conn.handler = handler

	conn.admitUnsubscribePod(&runnerv1.UnsubscribePodCommand{PodKey: "missing"})
	waitUnsubscribe(t, handler.unsubscribeCalls)
}

func (h *relayIntentDispatchHandler) OnUnsubscribePod(req UnsubscribePodRequest) error {
	h.unsubscribeCalls <- req
	if h.unsubscribeRelease != nil {
		<-h.unsubscribeRelease
	}
	return nil
}

func TestRelayIntentAdmissionCoalescesBeforeCreateWorkerReady(t *testing.T) {
	createRelease := make(chan struct{})
	handler := newRelayIntentDispatchHandler()
	handler.createRelease = createRelease
	conn := newTestConnection()
	conn.handler = handler

	conn.handleServerMessage(t.Context(), createPodServerMessage("pre-placeholder"))
	waitSignal(t, handler.createStarted, "create handler did not start")
	conn.handleServerMessage(t.Context(), subscribePodServerMessage("pre-placeholder", "wss://old", "old-token"))
	conn.handleServerMessage(t.Context(), subscribePodServerMessage("pre-placeholder", "wss://latest", "latest-token"))

	select {
	case call := <-handler.subscribeCalls:
		t.Fatalf("subscribe escaped before create completed: %+v", call.req)
	default:
	}
	close(createRelease)

	call := waitSubscription(t, handler.subscribeCalls)
	if call.req.RelayURL != "wss://latest" || call.req.RunnerToken != "latest-token" {
		t.Fatalf("got stale subscription: url=%q token=%q", call.req.RelayURL, call.req.RunnerToken)
	}
	if !call.validAtCall {
		t.Fatal("latest subscription was already invalid when dispatched")
	}
	conn.handlerWg.Wait()
	assertNoSubscription(t, handler.subscribeCalls)
	cleanupRelayIntentTest(t, conn, "pre-placeholder")
}

func TestRelayIntentAdmissionUnsubscribeWinsBeforeCreateReady(t *testing.T) {
	createRelease := make(chan struct{})
	handler := newRelayIntentDispatchHandler()
	handler.createRelease = createRelease
	conn := newTestConnection()
	conn.handler = handler

	conn.handleServerMessage(t.Context(), createPodServerMessage("pending-unsubscribe"))
	waitSignal(t, handler.createStarted, "create handler did not start")
	conn.handleServerMessage(t.Context(), subscribePodServerMessage("pending-unsubscribe", "wss://relay", "secret"))
	conn.handleServerMessage(t.Context(), unsubscribePodServerMessage("pending-unsubscribe"))
	close(createRelease)
	conn.handlerWg.Wait()

	assertNoSubscription(t, handler.subscribeCalls)
	cleanupRelayIntentTest(t, conn, "pending-unsubscribe")
}

func TestRelayIntentAdmissionTerminateCancelsPendingSubscribe(t *testing.T) {
	createRelease := make(chan struct{})
	handler := newRelayIntentDispatchHandler()
	handler.createRelease = createRelease
	conn := newTestConnection()
	conn.handler = handler

	conn.handleServerMessage(t.Context(), createPodServerMessage("pending-terminate"))
	waitSignal(t, handler.createStarted, "create handler did not start")
	conn.handleServerMessage(t.Context(), subscribePodServerMessage("pending-terminate", "wss://relay", "secret"))
	conn.handleServerMessage(t.Context(), &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_TerminatePod{
			TerminatePod: &runnerv1.TerminatePodCommand{PodKey: "pending-terminate"},
		},
	})
	close(createRelease)
	conn.handlerWg.Wait()
	conn.podQueue.Wait()

	assertNoSubscription(t, handler.subscribeCalls)
	if conn.podRelayIntents.get("pending-terminate") != nil {
		t.Fatal("terminate retained pending relay intent")
	}
}

func TestRelayIntentAdmissionInvalidatesInFlightSubscribe(t *testing.T) {
	createRelease := make(chan struct{})
	close(createRelease)
	subscribeRelease := make(chan struct{})
	handler := newRelayIntentDispatchHandler()
	handler.createRelease = createRelease
	handler.subscribeRelease = subscribeRelease
	conn := newTestConnection()
	conn.handler = handler

	conn.handleServerMessage(t.Context(), createPodServerMessage("in-flight"))
	waitSignal(t, handler.createStarted, "create handler did not start")
	conn.handlerWg.Wait()
	conn.handleServerMessage(t.Context(), subscribePodServerMessage("in-flight", "wss://relay", "token"))
	call := waitSubscription(t, handler.subscribeCalls)
	conn.handleServerMessage(t.Context(), unsubscribePodServerMessage("in-flight"))
	waitUnsubscribe(t, handler.unsubscribeCalls)

	if call.req.IntentValid == nil || call.req.IntentValid() {
		t.Fatal("unsubscribe did not invalidate the in-flight subscription revision")
	}
	close(subscribeRelease)
	conn.handlerWg.Wait()
	assertNoSubscription(t, handler.subscribeCalls)
	cleanupRelayIntentTest(t, conn, "in-flight")
}

func TestRelayIntentAdmissionWaitsForTeardownBeforeLatestSubscribe(t *testing.T) {
	createRelease := make(chan struct{})
	close(createRelease)
	subscribeRelease := make(chan struct{})
	unsubscribeRelease := make(chan struct{})
	handler := newRelayIntentDispatchHandler()
	handler.createRelease = createRelease
	handler.subscribeRelease = subscribeRelease
	handler.unsubscribeRelease = unsubscribeRelease
	conn := newTestConnection()
	conn.handler = handler

	conn.handleServerMessage(t.Context(), createPodServerMessage("teardown-fence"))
	waitSignal(t, handler.createStarted, "create handler did not start")
	conn.handlerWg.Wait()
	conn.handleServerMessage(t.Context(), subscribePodServerMessage(
		"teardown-fence", "wss://old", "old-token",
	))
	first := waitSubscription(t, handler.subscribeCalls)

	secondAdmissionDone := make(chan struct{})
	go func() {
		conn.handleServerMessage(t.Context(), subscribePodServerMessage(
			"teardown-fence", "wss://latest", "latest-token",
		))
		close(secondAdmissionDone)
	}()
	waitUnsubscribe(t, handler.unsubscribeCalls)
	if first.req.IntentValid == nil || first.req.IntentValid() {
		t.Fatal("replacement subscribe did not invalidate the in-flight revision")
	}

	close(subscribeRelease)
	assertNoSubscription(t, handler.subscribeCalls)
	close(unsubscribeRelease)
	waitSignal(t, secondAdmissionDone, "replacement admission did not finish teardown")

	latest := waitSubscription(t, handler.subscribeCalls)
	if latest.req.RelayURL != "wss://latest" || latest.req.RunnerToken != "latest-token" {
		t.Fatalf("got stale subscription after teardown: url=%q token=%q",
			latest.req.RelayURL, latest.req.RunnerToken)
	}
	if !latest.validAtCall {
		t.Fatal("latest subscription was invalid after teardown fence released")
	}
	conn.handlerWg.Wait()
	cleanupRelayIntentTest(t, conn, "teardown-fence")
}

func TestRelayIntentAdmissionNewCreateFencesInstalledRelay(t *testing.T) {
	createRelease := make(chan struct{})
	close(createRelease)
	handler := newRelayIntentDispatchHandler()
	handler.createRelease = createRelease
	conn := newTestConnection()
	conn.handler = handler

	conn.handleServerMessage(t.Context(), createPodServerMessage("recreated"))
	waitSignal(t, handler.createStarted, "create handler did not start")
	conn.handlerWg.Wait()
	conn.handleServerMessage(t.Context(), subscribePodServerMessage("recreated", "wss://old", "old-token"))
	call := waitSubscription(t, handler.subscribeCalls)
	if !call.validAtCall {
		t.Fatal("initial subscription was invalid")
	}
	conn.handlerWg.Wait()

	conn.handleServerMessage(t.Context(), createPodServerMessage("recreated"))
	waitUnsubscribe(t, handler.unsubscribeCalls)
	conn.handlerWg.Wait()
	cleanupRelayIntentTest(t, conn, "recreated")
}

func TestRelayIntentAdmissionCreateFailureClearsCredentials(t *testing.T) {
	createRelease := make(chan struct{})
	handler := newRelayIntentDispatchHandler()
	handler.createRelease = createRelease
	handler.createErr = errors.New("build failed")
	conn := newTestConnection()
	conn.handler = handler

	conn.handleServerMessage(t.Context(), createPodServerMessage("failed-create"))
	waitSignal(t, handler.createStarted, "create handler did not start")
	conn.handleServerMessage(t.Context(), subscribePodServerMessage("failed-create", "wss://relay", "secret-token"))
	intent := conn.podRelayIntents.get("failed-create")
	intent.mu.Lock()
	queued := intent.pending
	intent.mu.Unlock()
	close(createRelease)
	conn.handlerWg.Wait()

	if conn.podRelayIntents.get("failed-create") != nil {
		t.Fatal("failed create retained relay intent")
	}
	if queued == nil || queued.RunnerToken != "" || queued.LocalToken != "" {
		t.Fatal("failed create did not clear queued credentials")
	}
	assertNoSubscription(t, handler.subscribeCalls)
}

func createPodServerMessage(podKey string) *runnerv1.ServerMessage {
	return &runnerv1.ServerMessage{Payload: &runnerv1.ServerMessage_CreatePod{
		CreatePod: &runnerv1.CreatePodCommand{PodKey: podKey, LaunchCommand: "test"},
	}}
}

func subscribePodServerMessage(podKey, relayURL, token string) *runnerv1.ServerMessage {
	return &runnerv1.ServerMessage{Payload: &runnerv1.ServerMessage_SubscribePod{
		SubscribePod: &runnerv1.SubscribePodCommand{
			PodKey: podKey, RelayUrl: relayURL, RunnerToken: token, LocalToken: "local-" + token,
		},
	}}
}

func unsubscribePodServerMessage(podKey string) *runnerv1.ServerMessage {
	return &runnerv1.ServerMessage{Payload: &runnerv1.ServerMessage_UnsubscribePod{
		UnsubscribePod: &runnerv1.UnsubscribePodCommand{PodKey: podKey},
	}}
}

func waitSignal(t *testing.T, signal <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatal(message)
	}
}

func waitSubscription(t *testing.T, calls <-chan relayIntentSubscription) relayIntentSubscription {
	t.Helper()
	select {
	case call := <-calls:
		return call
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for subscription")
		return relayIntentSubscription{}
	}
}

func waitUnsubscribe(t *testing.T, calls <-chan UnsubscribePodRequest) {
	t.Helper()
	select {
	case <-calls:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for unsubscribe")
	}
}

func assertNoSubscription(t *testing.T, calls <-chan relayIntentSubscription) {
	t.Helper()
	select {
	case call := <-calls:
		t.Fatalf("unexpected extra subscription: %+v", call.req)
	case <-time.After(50 * time.Millisecond):
	}
}

func cleanupRelayIntentTest(t *testing.T, conn *GRPCConnection, podKey string) {
	t.Helper()
	conn.handleServerMessage(t.Context(), &runnerv1.ServerMessage{
		Payload: &runnerv1.ServerMessage_TerminatePod{
			TerminatePod: &runnerv1.TerminatePodCommand{PodKey: podKey},
		},
	})
	conn.handlerWg.Wait()
	conn.podQueue.Wait()
}
