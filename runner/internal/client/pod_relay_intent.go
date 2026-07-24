package client

import "sync"

type relayIntentPhase uint8

const (
	relayIntentPending relayIntentPhase = iota
	relayIntentDraining
	relayIntentOpen
	relayIntentAborted
)

type relayIntentAdmission uint8

const (
	relayIntentDirect relayIntentAdmission = iota
	relayIntentQueued
	relayIntentStart
	relayIntentInvalidate
	relayIntentRetry
)

// podRelayIntent is the subscription intent for one pod incarnation. It keeps
// at most one queued request in addition to the request currently being applied.
type podRelayIntent struct {
	podKey     string
	mu         sync.Mutex
	phase      relayIntentPhase
	revision   uint64
	pending    *SubscribePodRequest
	subscribed bool
	drainEpoch uint64
	drainOwned bool
	// A newer S/U tears down the in-flight request's Pod transport. Keep the
	// latest request behind this fence until relayTransportBlock is cleared.
	teardownEpoch   uint64
	teardownPending uint64
	teardownDone    chan struct{}
}

type podRelayIntentRegistry struct {
	mu      sync.Mutex
	intents map[string]*podRelayIntent
}

func newPodRelayIntentRegistry() *podRelayIntentRegistry {
	return &podRelayIntentRegistry{intents: make(map[string]*podRelayIntent)}
}

// Begin runs synchronously when CreatePod is received, before its worker is
// scheduled. The returned pointer is the create transaction's ownership token.
func (r *podRelayIntentRegistry) Begin(podKey string) (*podRelayIntent, bool) {
	intent := &podRelayIntent{podKey: podKey, phase: relayIntentPending}
	r.mu.Lock()
	previous := r.intents[podKey]
	r.intents[podKey] = intent
	r.mu.Unlock()
	if previous == nil {
		return intent, false
	}
	return intent, previous.abort()
}

// OfferSubscribe also creates an open intent for recovered pods whose CreatePod
// was handled before this gRPC connection started.
func (r *podRelayIntentRegistry) OfferSubscribe(
	req SubscribePodRequest,
) (*podRelayIntent, relayIntentAdmission, uint64) {
	for {
		intent := r.getOrCreateOpen(req.PodKey)
		result, teardownEpoch := intent.offerSubscribe(req)
		if result != relayIntentRetry {
			return intent, result, teardownEpoch
		}
	}
}

func (r *podRelayIntentRegistry) OfferUnsubscribe(
	podKey string,
) (*podRelayIntent, relayIntentAdmission, uint64) {
	for {
		intent := r.get(podKey)
		if intent == nil {
			return nil, relayIntentDirect, 0
		}
		result, teardownEpoch := intent.offerUnsubscribe()
		if result != relayIntentRetry {
			return intent, result, teardownEpoch
		}
	}
}

// Drain is started after CreatePod succeeds or an open incarnation receives a
// new Subscribe. A stable empty queue returns to open without deleting intent.
func (r *podRelayIntentRegistry) Drain(
	intent *podRelayIntent,
	drainEpoch uint64,
	apply func(SubscribePodRequest),
) {
	if intent == nil || apply == nil {
		return
	}
	defer intent.endDrain(drainEpoch)
	for {
		req, revision, teardownDone, ok := intent.takeLatest(drainEpoch)
		if teardownDone != nil {
			<-teardownDone
			continue
		}
		if !ok {
			return
		}
		req.IntentValid = func() bool { return intent.current(drainEpoch, revision) }
		apply(req)
		req.RunnerToken = ""
		req.LocalToken = ""
		req.IntentValid = nil
	}
}

func (r *podRelayIntentRegistry) AbortIf(expected *podRelayIntent) bool {
	if expected == nil {
		return false
	}
	r.mu.Lock()
	if r.intents[expected.podKey] != expected {
		r.mu.Unlock()
		return false
	}
	delete(r.intents, expected.podKey)
	r.mu.Unlock()
	return expected.abort()
}

func (r *podRelayIntentRegistry) Cancel(podKey string) bool {
	r.mu.Lock()
	intent := r.intents[podKey]
	if intent != nil {
		delete(r.intents, podKey)
	}
	r.mu.Unlock()
	return intent != nil && intent.abort()
}

// CancelAll closes admission before GRPCConnection waits for drain workers.
func (r *podRelayIntentRegistry) CancelAll() []string {
	r.mu.Lock()
	intents := r.intents
	r.intents = make(map[string]*podRelayIntent)
	r.mu.Unlock()

	invalidated := make([]string, 0, len(intents))
	for podKey, intent := range intents {
		if intent.abort() {
			invalidated = append(invalidated, podKey)
		}
	}
	return invalidated
}

func (r *podRelayIntentRegistry) get(podKey string) *podRelayIntent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.intents[podKey]
}

func (r *podRelayIntentRegistry) getOrCreateOpen(podKey string) *podRelayIntent {
	r.mu.Lock()
	defer r.mu.Unlock()
	if intent := r.intents[podKey]; intent != nil {
		return intent
	}
	intent := &podRelayIntent{podKey: podKey, phase: relayIntentOpen}
	r.intents[podKey] = intent
	return intent
}
