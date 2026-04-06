// Package autopilot implements the AutopilotController for supervised Pod automation.
package autopilot

import (
	"log/slog"
	"sync"
	"time"
	"unicode/utf8"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// IterationController manages iteration counting, trigger deduplication,
// and max iteration protection.
type IterationController struct {
	mu              sync.RWMutex
	currentIter     int
	maxIterations   int
	lastIterationAt time.Time
	startedAt       time.Time

	// Trigger deduplication
	triggerMu       sync.Mutex
	lastTriggerTime time.Time
	minTriggerGap   time.Duration

	// Error tracking for retry limiting
	consecutiveErrors    int
	maxConsecutiveErrors int

	// Dependencies
	reporter     EventReporter
	autopilotKey string
	podKey       string
	log          *slog.Logger
}

// IterationControllerConfig contains configuration for creating an IterationController.
type IterationControllerConfig struct {
	MaxIterations        int
	MinTriggerGap        time.Duration
	MaxConsecutiveErrors int // Max consecutive errors before giving up (default: 3)
	Reporter             EventReporter
	AutopilotKey         string
	PodKey               string
	Logger               *slog.Logger
}

// NewIterationController creates a new IterationController instance.
func NewIterationController(cfg IterationControllerConfig) *IterationController {
	maxIterations := cfg.MaxIterations
	if maxIterations == 0 {
		maxIterations = DefaultMaxIterations
	}

	minTriggerGap := cfg.MinTriggerGap
	if minTriggerGap == 0 {
		minTriggerGap = DefaultMinTriggerGap
	}

	maxConsecutiveErrors := cfg.MaxConsecutiveErrors
	if maxConsecutiveErrors == 0 {
		maxConsecutiveErrors = DefaultMaxConsecutiveErrors
	}

	return &IterationController{
		maxIterations:        maxIterations,
		minTriggerGap:        minTriggerGap,
		maxConsecutiveErrors: maxConsecutiveErrors,
		startedAt:            time.Now(),
		reporter:             cfg.Reporter,
		autopilotKey:         cfg.AutopilotKey,
		podKey:               cfg.PodKey,
		log:                  cfg.Logger,
	}
}

// GetCurrentIteration returns the current iteration number.
func (ic *IterationController) GetCurrentIteration() int {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	return ic.currentIter
}

// GetMaxIterations returns the maximum allowed iterations.
func (ic *IterationController) GetMaxIterations() int {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	return ic.maxIterations
}

// GetStartedAt returns when the controller was started.
func (ic *IterationController) GetStartedAt() time.Time {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	return ic.startedAt
}

// GetLastIterationAt returns when the last iteration occurred.
func (ic *IterationController) GetLastIterationAt() time.Time {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	return ic.lastIterationAt
}

// AddMaxIterations adds additional iterations to the max limit.
func (ic *IterationController) AddMaxIterations(additional int) int {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	ic.maxIterations += additional
	return ic.maxIterations
}

// CheckTriggerDedup checks if enough time has passed since the last trigger.
// Returns true if the trigger should proceed, false if it should be skipped.
// Also updates the last trigger time if returning true.
func (ic *IterationController) CheckTriggerDedup() bool {
	ic.triggerMu.Lock()
	defer ic.triggerMu.Unlock()

	if time.Since(ic.lastTriggerTime) < ic.minTriggerGap {
		logger.AutopilotTrace().Trace("Skipping iteration - too soon since last trigger",
			"autopilot_key", ic.autopilotKey,
			"gap", time.Since(ic.lastTriggerTime),
			"min_gap", ic.minTriggerGap)
		return false
	}
	ic.lastTriggerTime = time.Now()
	return true
}

// UpdateTriggerTime updates the last trigger time to now.
// Used when starting prompt to prevent OnPodWaiting from double-triggering.
func (ic *IterationController) UpdateTriggerTime() {
	ic.triggerMu.Lock()
	defer ic.triggerMu.Unlock()
	ic.lastTriggerTime = time.Now()
}

// HasReachedMaxIterations checks if current iteration has reached max.
func (ic *IterationController) HasReachedMaxIterations() bool {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	return ic.currentIter >= ic.maxIterations
}

// IncrementIteration increments the iteration counter and returns the new value.
// Returns (newIteration, true) if successful, (currentIteration, false) if max reached.
func (ic *IterationController) IncrementIteration() (int, bool) {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	if ic.currentIter >= ic.maxIterations {
		return ic.currentIter, false
	}

	ic.currentIter++
	ic.lastIterationAt = time.Now()
	return ic.currentIter, true
}

// SetInitialIteration sets the iteration to 1 for the first run.
// Returns the iteration number.
func (ic *IterationController) SetInitialIteration() int {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	ic.currentIter = 1
	ic.lastIterationAt = time.Now()
	return ic.currentIter
}

// ReportIterationEvent reports an iteration event via the EventReporter.
func (ic *IterationController) ReportIterationEvent(iteration int, phase, summary string, filesChanged []string) {
	if ic.reporter == nil {
		return
	}

	// Ensure summary is valid UTF-8 and within reasonable length
	safeSummary := sanitizeUTF8String(summary, 1000)

	ic.reporter.ReportAutopilotIteration(&runnerv1.AutopilotIterationEvent{
		AutopilotKey: ic.autopilotKey,
		Iteration:    int32(iteration),
		Phase:        phase,
		Summary:      safeSummary,
		FilesChanged: filesChanged,
	})
}

// sanitizeUTF8String ensures the string is valid UTF-8 and truncates safely at rune boundaries.
func sanitizeUTF8String(s string, maxLen int) string {
	// First, ensure it's valid UTF-8 by converting invalid sequences
	if !utf8.ValidString(s) {
		// Replace invalid UTF-8 sequences with replacement character
		s = string([]rune(s))
	}

	// Truncate at rune boundary if too long
	if len(s) > maxLen {
		runes := []rune(s)
		if len(runes) > maxLen {
			runes = runes[:maxLen]
		}
		s = string(runes)
		// Check if we need to truncate further due to byte length
		for len(s) > maxLen {
			runes = runes[:len(runes)-1]
			s = string(runes)
		}
	}

	return s
}

// GetStatus returns an AutopilotStatus proto with iteration-related fields populated.
func (ic *IterationController) GetStatus() *runnerv1.AutopilotStatus {
	ic.mu.RLock()
	defer ic.mu.RUnlock()

	return &runnerv1.AutopilotStatus{
		CurrentIteration: int32(ic.currentIter),
		MaxIterations:    int32(ic.maxIterations),
		StartedAt:        ic.startedAt.Unix(),
		LastIterationAt:  ic.lastIterationAt.Unix(),
	}
}

// RecordError records a consecutive error and returns true if max errors exceeded.
func (ic *IterationController) RecordError() bool {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	ic.consecutiveErrors++
	exceeded := ic.consecutiveErrors >= ic.maxConsecutiveErrors

	if ic.log != nil {
		ic.log.Warn("Recorded consecutive error",
			"autopilot_key", ic.autopilotKey,
			"consecutive_errors", ic.consecutiveErrors,
			"max_consecutive_errors", ic.maxConsecutiveErrors,
			"exceeded", exceeded)
	}

	return exceeded
}

// ResetErrors resets the consecutive error counter (called on successful iteration).
func (ic *IterationController) ResetErrors() {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	if ic.consecutiveErrors > 0 {
		logger.AutopilotTrace().Trace("Reset consecutive errors",
			"autopilot_key", ic.autopilotKey,
			"previous_count", ic.consecutiveErrors)
		ic.consecutiveErrors = 0
	}
}

// GetConsecutiveErrors returns the current consecutive error count.
func (ic *IterationController) GetConsecutiveErrors() int {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	return ic.consecutiveErrors
}

// GetRetryDelay returns an exponential backoff delay based on consecutive errors.
// Returns 2^n seconds, capped at MaxRetryDelay.
func (ic *IterationController) GetRetryDelay() time.Duration {
	ic.mu.RLock()
	errors := ic.consecutiveErrors
	ic.mu.RUnlock()

	if errors == 0 {
		return MinRetryDelay
	}

	// Exponential backoff: 2, 4, 8, 16, 30 (capped)
	delay := time.Duration(1<<errors) * time.Second
	if delay > MaxRetryDelay {
		delay = MaxRetryDelay
	}
	return delay
}
