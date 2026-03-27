package detector

import (
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/safego"
)

// setState updates the current state and triggers callbacks.
// Must be called with d.mu held.
func (d *MultiSignalDetector) setState(newState AgentState) {
	d.setStateWithConfidence(newState, d.lastConfidence)
}

// setStateWithConfidence updates the current state with a specific confidence value.
// Must be called with d.mu held.
func (d *MultiSignalDetector) setStateWithConfidence(newState AgentState, confidence float64) {
	if d.currentState == newState {
		return
	}

	prevState := d.currentState
	d.currentState = newState
	now := time.Now()
	d.stateChangeTime = now

	// Create event for subscribers
	event := StateChangeEvent{
		NewState:   newState,
		PrevState:  prevState,
		Timestamp:  now,
		Confidence: confidence,
	}

	// Legacy callback (for backward compatibility)
	if d.onStateChange != nil {
		cb := d.onStateChange
		safego.Go("detector-callback", func() { cb(newState, prevState) })
	}

	// Notify subscribers (use separate lock to avoid deadlock)
	d.notifySubscribers(event)
}

// notifySubscribers sends the event to all registered subscribers.
// Each subscriber callback is invoked in a separate goroutine.
func (d *MultiSignalDetector) notifySubscribers(event StateChangeEvent) {
	d.subMu.RLock()
	// Copy subscribers to avoid holding lock during callback execution
	subs := make(map[string]func(StateChangeEvent), len(d.subscribers))
	for id, cb := range d.subscribers {
		subs[id] = cb
	}
	d.subMu.RUnlock()

	// Invoke callbacks asynchronously
	for _, cb := range subs {
		callback := cb
		safego.Go("detector-subscriber", func() { callback(event) })
	}
}
