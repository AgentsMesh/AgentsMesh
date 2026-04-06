// Package detector provides terminal state detection for AI agents.
package detector

import "time"

// StateDetector defines the interface for terminal state detection.
// This is a foundational service that can be used by any component,
// independent of Autopilot or other higher-level modules.
type StateDetector interface {
	// DetectState analyzes and returns the current agent state.
	DetectState() AgentState
	// GetState returns the current state without performing detection.
	GetState() AgentState
	// Reset resets the detector state.
	Reset()
	// OnOutput should be called when terminal output is received.
	OnOutput(bytes int)
	// OnScreenUpdate should be called with current screen lines after each Feed.
	// This enables single-direction data flow without reverse lock acquisition.
	OnScreenUpdate(lines []string)
	// Subscribe adds a subscriber for state change events.
	// The subscriber ID must be unique; duplicate IDs will replace existing subscriptions.
	Subscribe(id string, cb func(StateChangeEvent))
	// Unsubscribe removes a subscriber by ID.
	Unsubscribe(id string)
}

// StateChangeEvent represents a state transition event.
// This is used for event-driven notification to subscribers.
type StateChangeEvent struct {
	// NewState is the state after the transition.
	NewState AgentState
	// PrevState is the state before the transition.
	PrevState AgentState
	// Timestamp is when the transition occurred.
	Timestamp time.Time
	// Confidence is the confidence score (0.0-1.0) of the detection.
	// Only applicable for transitions to StateWaiting.
	Confidence float64
}
