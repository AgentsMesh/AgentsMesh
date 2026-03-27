package detector

import (
	"time"
)

// MultiSignalConfig contains configuration for MultiSignalDetector.
type MultiSignalConfig struct {
	// ActivityWeight is the weight for output activity signal (default: 0.4)
	ActivityWeight float64
	// StabilityWeight is the weight for screen stability signal (default: 0.3)
	StabilityWeight float64
	// PromptWeight is the weight for prompt detection signal (default: 0.3)
	PromptWeight float64

	// MinStableTime is the minimum time screen must be stable (default: 500ms)
	MinStableTime time.Duration
	// WaitingThreshold is the confidence threshold to transition to waiting (default: 0.6)
	WaitingThreshold float64

	// IdleThreshold for activity detector (default: 500ms)
	IdleThreshold time.Duration
	// ConfirmThreshold for activity detector (default: 500ms)
	ConfirmThreshold time.Duration

	// MaxPromptLength for prompt detector (default: 100)
	MaxPromptLength int

	// OnStateChange callback
	OnStateChange StateChangeCallback
}

// applyDefaults fills zero-valued fields with their defaults.
func (c *MultiSignalConfig) applyDefaults() {
	if c.ActivityWeight == 0 {
		c.ActivityWeight = 0.4
	}
	if c.StabilityWeight == 0 {
		c.StabilityWeight = 0.3
	}
	if c.PromptWeight == 0 {
		c.PromptWeight = 0.3
	}
	if c.MinStableTime == 0 {
		c.MinStableTime = 500 * time.Millisecond
	}
	if c.WaitingThreshold == 0 {
		c.WaitingThreshold = 0.6
	}
	if c.IdleThreshold == 0 {
		c.IdleThreshold = 500 * time.Millisecond
	}
	if c.ConfirmThreshold == 0 {
		c.ConfirmThreshold = 500 * time.Millisecond
	}
	if c.MaxPromptLength == 0 {
		c.MaxPromptLength = 100
	}
}
