package terminal

import (
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// confidenceLogData holds data for logging confidence calculation results.
// This allows logging to happen outside the lock.
type confidenceLogData struct {
	activityState    ActivityState
	activityContrib  float64
	screenStableTime time.Duration
	stabilityContrib float64
	promptDetected   bool
	promptType       PromptType
	promptContrib    float64
	confidence       float64
	threshold        float64
	screenLinesCount int
}

// DetectState analyzes all signals and returns the current agent state.
// This should be called periodically (e.g., every 200-300ms).
func (d *MultiSignalDetector) DetectState() AgentState {
	d.mu.Lock()

	d.lastCheckTime = time.Now()

	// Update activity detector state
	activityState := d.activityDetector.CheckState()

	// Calculate confidence for "waiting" state (captures values for logging)
	confidence, logData := d.calculateWaitingConfidenceLocked(activityState)

	// State transition logic
	switch d.currentState {
	case StateNotRunning:
		// Stay in NotRunning until output is received (handled in OnOutput)

	case StateExecuting:
		// Check if we should transition to Waiting
		if confidence >= d.config.WaitingThreshold {
			d.setState(StateWaiting)
		}

	case StateWaiting:
		// Check if we should transition back to Executing
		// This happens in OnOutput when new output is received
		// But we can also check if confidence dropped significantly
		if confidence < d.config.WaitingThreshold*0.5 {
			d.setState(StateExecuting)
		}
	}

	currentState := d.currentState
	d.mu.Unlock()

	// Debug logging OUTSIDE lock to avoid blocking PTY output
	logger.TerminalTrace().Trace("MultiSignalDetector confidence calculation",
		"activity_state", logData.activityState,
		"activity_contrib", logData.activityContrib,
		"screen_stable_time", logData.screenStableTime,
		"stability_contrib", logData.stabilityContrib,
		"prompt_detected", logData.promptDetected,
		"prompt_type", logData.promptType,
		"prompt_contrib", logData.promptContrib,
		"total_confidence", logData.confidence,
		"threshold", logData.threshold,
		"screen_lines_count", logData.screenLinesCount)

	return currentState
}

// calculateWaitingConfidenceLocked calculates the confidence that the agent is waiting.
// Must be called with d.mu held. Returns confidence and log data for deferred logging.
func (d *MultiSignalDetector) calculateWaitingConfidenceLocked(activityState ActivityState) (float64, confidenceLogData) {
	var confidence float64
	var activityContrib, stabilityContrib, promptContrib float64

	// Signal 1: Output Activity (weight: 0.4)
	// If output has stopped, this contributes to waiting confidence
	switch activityState {
	case ActivityStateIdle:
		activityContrib = d.config.ActivityWeight * 1.0
	case ActivityStatePotentialIdle:
		activityContrib = d.config.ActivityWeight * 0.7
	case ActivityStateActive:
		activityContrib = d.config.ActivityWeight * 0.0
	}
	confidence += activityContrib

	// Signal 2: Screen Stability (weight: 0.3)
	// If screen hasn't changed for a while, this contributes to waiting confidence
	if d.screenStableTime >= d.config.MinStableTime {
		// Scale based on how long it's been stable
		stableRatio := float64(d.screenStableTime) / float64(d.config.MinStableTime*2)
		if stableRatio > 1.0 {
			stableRatio = 1.0
		}
		stabilityContrib = d.config.StabilityWeight * stableRatio
		confidence += stabilityContrib
	}

	// Signal 3: Prompt Detection (weight: 0.3)
	// If a prompt is detected, this contributes to waiting confidence
	var promptResult PromptResult
	if len(d.screenLines) > 0 {
		promptResult = d.promptDetector.DetectPrompt(d.screenLines)
		if promptResult.IsPrompt {
			promptContrib = d.config.PromptWeight * promptResult.Confidence
			confidence += promptContrib
		}
	}

	// Optional: OSC Title Boost
	// If OSC title suggests waiting (e.g., contains waiting indicators), add a small boost
	if d.lastOSCTitle != "" && time.Since(d.lastOSCTitleTime) < 5*time.Second {
		if d.oscSuggestsWaiting(d.lastOSCTitle) {
			confidence += 0.1 // Small boost, don't depend on it
		}
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	// Capture log data for deferred logging outside lock
	logData := confidenceLogData{
		activityState:    activityState,
		activityContrib:  activityContrib,
		screenStableTime: d.screenStableTime,
		stabilityContrib: stabilityContrib,
		promptDetected:   promptResult.IsPrompt,
		promptType:       promptResult.PromptType,
		promptContrib:    promptContrib,
		confidence:       confidence,
		threshold:        d.config.WaitingThreshold,
		screenLinesCount: len(d.screenLines),
	}

	return confidence, logData
}

// oscSuggestsWaiting checks if the OSC title suggests the agent is waiting.
// This is a heuristic and not meant to be relied upon.
func (d *MultiSignalDetector) oscSuggestsWaiting(title string) bool {
	// Look for common waiting indicators
	// These are generic patterns, not specific to any agent
	waitingPatterns := []string{
		"waiting",
		"input",
		"prompt",
		"✳", // Common "idle/ready" indicator
		"⏳", // Waiting indicator
	}

	for _, pattern := range waitingPatterns {
		if containsIgnoreCase(title, pattern) {
			return true
		}
	}

	return false
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	// Simple lowercase comparison
	sLower := toLower(s)
	substrLower := toLower(substr)

	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}

// toLower converts a string to lowercase (ASCII only for performance).
func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
