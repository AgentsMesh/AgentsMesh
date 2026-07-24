package client

import "testing"

func TestRelayIntentRegistryDefensiveAdmissions(t *testing.T) {
	registry := newPodRelayIntentRegistry()

	if intent, admission, epoch := registry.OfferUnsubscribe("missing"); intent != nil || admission != relayIntentDirect || epoch != 0 {
		t.Fatalf("missing unsubscribe = (%v, %v, %d), want direct", intent, admission, epoch)
	}
	registry.Drain(nil, 0, func(SubscribePodRequest) { t.Fatal("nil intent was applied") })
	intent, _ := registry.Begin("pod")
	registry.Drain(intent, 0, nil)
	if registry.AbortIf(nil) {
		t.Fatal("nil AbortIf unexpectedly invalidated an intent")
	}
	replacement, _ := registry.Begin("pod")
	if registry.AbortIf(intent) {
		t.Fatal("stale incarnation aborted its replacement")
	}
	if registry.AbortIf(replacement) {
		t.Fatal("pending incarnation incorrectly reported transport invalidation")
	}
	if registry.get("pod") != nil {
		t.Fatal("current pending incarnation was not removed")
	}
	if registry.Cancel("missing") {
		t.Fatal("missing cancellation reported an invalidation")
	}
}

func TestRelayIntentStateDefensiveEpochs(t *testing.T) {
	aborted := &podRelayIntent{phase: relayIntentAborted}
	if admission, _ := aborted.offerSubscribe(SubscribePodRequest{}); admission != relayIntentRetry {
		t.Fatalf("subscribe to aborted intent = %v, want retry", admission)
	}
	if admission, _ := aborted.offerUnsubscribe(); admission != relayIntentRetry {
		t.Fatalf("unsubscribe from aborted intent = %v, want retry", admission)
	}
	if _, ok := aborted.claimDrain(); ok {
		t.Fatal("aborted intent claimed a drain")
	}

	open := &podRelayIntent{phase: relayIntentOpen}
	if admission, epoch := open.offerUnsubscribe(); admission != relayIntentDirect || epoch != 0 {
		t.Fatalf("open unsubscribe = (%v, %d), want direct", admission, epoch)
	}
	if _, ok := open.claimDrain(); ok {
		t.Fatal("open intent claimed a drain without work")
	}
	open.phase = relayIntentDraining
	epoch, ok := open.claimDrain()
	if !ok {
		t.Fatal("draining intent failed to claim its drain")
	}
	if _, ok := open.claimDrain(); ok {
		t.Fatal("intent allowed two drain owners")
	}
	if _, _, _, ok := open.takeLatest(epoch + 1); ok {
		t.Fatal("stale drain epoch took work")
	}

	firstTeardown := open.beginTeardownLocked()
	secondTeardown := open.beginTeardownLocked()
	if firstTeardown != secondTeardown || open.teardownPending != 2 {
		t.Fatalf("coalesced teardown = (%d, %d, pending %d)", firstTeardown, secondTeardown, open.teardownPending)
	}
	open.endTeardown(firstTeardown + 1)
	if open.teardownPending != 2 {
		t.Fatal("stale teardown epoch changed pending count")
	}
	open.endTeardown(firstTeardown)
	if open.teardownPending != 1 || open.teardownDone == nil {
		t.Fatal("first teardown completion released fence too early")
	}
	open.endTeardown(firstTeardown)
	if open.teardownPending != 0 || open.teardownDone != nil {
		t.Fatal("final teardown completion did not release fence")
	}
	open.endTeardown(firstTeardown)

	open.pending = &SubscribePodRequest{RunnerToken: "secret", LocalToken: "local"}
	queued := open.pending
	open.endDrain(epoch)
	if queued.RunnerToken != "" || queued.LocalToken != "" || open.phase != relayIntentOpen || open.drainOwned {
		t.Fatal("current drain cleanup did not scrub credentials and release ownership")
	}
}

func TestRelayIntentOldDrainCannotClearNewDrain(t *testing.T) {
	registry := newPodRelayIntentRegistry()
	intent, _ := registry.Begin("pod-1")
	if _, result, _ := registry.OfferSubscribe(SubscribePodRequest{
		PodKey: "pod-1", RunnerToken: "old-token",
	}); result != relayIntentQueued {
		t.Fatalf("initial offer = %v, want queued", result)
	}
	oldEpoch, ok := intent.claimDrain()
	if !ok {
		t.Fatal("failed to claim initial drain")
	}
	if _, _, _, ok := intent.takeLatest(oldEpoch); !ok {
		t.Fatal("initial drain did not take request")
	}
	if _, _, _, ok := intent.takeLatest(oldEpoch); ok {
		t.Fatal("initial drain unexpectedly found another request")
	}

	if _, result, _ := registry.OfferSubscribe(SubscribePodRequest{
		PodKey: "pod-1", RunnerToken: "new-token",
	}); result != relayIntentStart {
		t.Fatalf("new offer = %v, want start", result)
	}
	newEpoch, ok := intent.claimDrain()
	if !ok || newEpoch == oldEpoch {
		t.Fatalf("new drain claim = (%d, %v), old epoch %d", newEpoch, ok, oldEpoch)
	}

	// This is the old goroutine's deferred cleanup after the new drain starts.
	intent.endDrain(oldEpoch)
	req, _, _, ok := intent.takeLatest(newEpoch)
	if !ok || req.RunnerToken != "new-token" {
		t.Fatalf("old drain cleared new request: ok=%v token=%q", ok, req.RunnerToken)
	}
}

func TestRelayIntentNewIncarnationInvalidatesOldDrain(t *testing.T) {
	registry := newPodRelayIntentRegistry()
	old, _ := registry.Begin("pod-1")
	if _, result, _ := registry.OfferSubscribe(SubscribePodRequest{
		PodKey: "pod-1", RunnerToken: "old-token",
	}); result != relayIntentQueued {
		t.Fatalf("old offer = %v, want queued", result)
	}
	epoch, ok := old.claimDrain()
	if !ok {
		t.Fatal("failed to claim old drain")
	}
	_, revision, _, ok := old.takeLatest(epoch)
	if !ok || !old.current(epoch, revision) {
		t.Fatal("old request was not current before replacement")
	}

	replacement, invalidate := registry.Begin("pod-1")
	if !invalidate {
		t.Fatal("new incarnation did not request old transport invalidation")
	}
	if old.current(epoch, revision) {
		t.Fatal("old drain remained current after incarnation replacement")
	}
	if got := registry.get("pod-1"); got != replacement {
		t.Fatal("replacement incarnation does not own registry key")
	}
}

func TestRelayIntentAbortReleasesTeardownWaiter(t *testing.T) {
	registry := newPodRelayIntentRegistry()
	intent, _ := registry.Begin("pod-1")
	if _, result, _ := registry.OfferSubscribe(SubscribePodRequest{
		PodKey: "pod-1", RunnerToken: "old-token",
	}); result != relayIntentQueued {
		t.Fatalf("initial offer = %v, want queued", result)
	}
	drainEpoch, ok := intent.claimDrain()
	if !ok {
		t.Fatal("failed to claim drain")
	}
	if _, _, _, ok := intent.takeLatest(drainEpoch); !ok {
		t.Fatal("initial drain did not take request")
	}
	_, result, teardownEpoch := registry.OfferSubscribe(SubscribePodRequest{
		PodKey: "pod-1", RunnerToken: "latest-token",
	})
	if result != relayIntentInvalidate || teardownEpoch == 0 {
		t.Fatalf("replacement offer = (%v, %d), want invalidate with teardown epoch", result, teardownEpoch)
	}
	_, _, teardownDone, ok := intent.takeLatest(drainEpoch)
	if ok || teardownDone == nil {
		t.Fatal("drain did not wait for invalidation teardown")
	}

	if !registry.Cancel("pod-1") {
		t.Fatal("cancel did not invalidate the draining intent")
	}
	select {
	case <-teardownDone:
	default:
		t.Fatal("abort did not release teardown waiter")
	}
	intent.endTeardown(teardownEpoch)
}

func TestRelayIntentCancelAllReleasesTeardownWaiter(t *testing.T) {
	registry := newPodRelayIntentRegistry()
	intent, _ := registry.Begin("pod-1")
	if _, result, _ := registry.OfferSubscribe(SubscribePodRequest{PodKey: "pod-1"}); result != relayIntentQueued {
		t.Fatalf("initial offer = %v, want queued", result)
	}
	drainEpoch, ok := intent.claimDrain()
	if !ok {
		t.Fatal("failed to claim drain")
	}
	if _, _, _, ok := intent.takeLatest(drainEpoch); !ok {
		t.Fatal("initial drain did not take request")
	}
	if _, result, _ := registry.OfferSubscribe(SubscribePodRequest{PodKey: "pod-1"}); result != relayIntentInvalidate {
		t.Fatalf("replacement offer = %v, want invalidate", result)
	}
	_, _, teardownDone, ok := intent.takeLatest(drainEpoch)
	if ok || teardownDone == nil {
		t.Fatal("drain did not expose teardown fence")
	}

	invalidated := registry.CancelAll()
	if len(invalidated) != 1 || invalidated[0] != "pod-1" {
		t.Fatalf("CancelAll invalidated %v, want [pod-1]", invalidated)
	}
	select {
	case <-teardownDone:
	default:
		t.Fatal("CancelAll did not release teardown waiter")
	}
}

func TestRelayIntentNewIncarnationDoesNotInheritOldTeardownFence(t *testing.T) {
	registry := newPodRelayIntentRegistry()
	old, _ := registry.Begin("pod-1")
	if _, result, _ := registry.OfferSubscribe(SubscribePodRequest{PodKey: "pod-1"}); result != relayIntentQueued {
		t.Fatalf("initial offer = %v, want queued", result)
	}
	oldDrain, ok := old.claimDrain()
	if !ok {
		t.Fatal("failed to claim old drain")
	}
	if _, _, _, ok := old.takeLatest(oldDrain); !ok {
		t.Fatal("old drain did not take request")
	}
	if _, result, _ := registry.OfferSubscribe(SubscribePodRequest{PodKey: "pod-1"}); result != relayIntentInvalidate {
		t.Fatalf("replacement offer = %v, want invalidate", result)
	}
	_, _, oldTeardownDone, ok := old.takeLatest(oldDrain)
	if ok || oldTeardownDone == nil {
		t.Fatal("old drain did not expose teardown fence")
	}

	replacement, invalidate := registry.Begin("pod-1")
	if !invalidate {
		t.Fatal("new incarnation did not invalidate old drain")
	}
	select {
	case <-oldTeardownDone:
	default:
		t.Fatal("new incarnation did not release old teardown fence")
	}
	if _, result, _ := registry.OfferSubscribe(SubscribePodRequest{
		PodKey: "pod-1", RunnerToken: "new-incarnation",
	}); result != relayIntentQueued {
		t.Fatalf("new incarnation offer = %v, want queued", result)
	}
	newDrain, ok := replacement.claimDrain()
	if !ok {
		t.Fatal("failed to claim replacement drain")
	}
	req, _, teardownDone, ok := replacement.takeLatest(newDrain)
	if !ok || teardownDone != nil || req.RunnerToken != "new-incarnation" {
		t.Fatalf("replacement inherited old fence: ok=%v wait=%v token=%q",
			ok, teardownDone != nil, req.RunnerToken)
	}
}
