// Package autopilot implements the AutopilotController for supervised Pod automation.
package autopilot

import (
	"sync"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// PhaseManager manages AutopilotController lifecycle phase transitions.
// It encapsulates the state machine logic and ensures thread-safe phase changes.
type PhaseManager struct {
	mu       sync.RWMutex
	phase    Phase
	reporter EventReporter

	autopilotKey string
	podKey       string
	statusGetter func() *runnerv1.AutopilotStatus // Callback to get full status for reporting
}

// PhaseManagerConfig contains configuration for creating a PhaseManager.
type PhaseManagerConfig struct {
	AutopilotKey string
	PodKey       string
	Reporter     EventReporter
	StatusGetter func() *runnerv1.AutopilotStatus
}

// NewPhaseManager creates a new PhaseManager instance.
func NewPhaseManager(cfg PhaseManagerConfig) *PhaseManager {
	return &PhaseManager{
		phase:        PhaseInitializing,
		reporter:     cfg.Reporter,
		autopilotKey: cfg.AutopilotKey,
		podKey:       cfg.PodKey,
		statusGetter: cfg.StatusGetter,
	}
}

// GetPhase returns the current phase (thread-safe).
func (pm *PhaseManager) GetPhase() Phase {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.phase
}

// SetPhase sets the current phase and reports status update.
// Returns true if phase was changed, false if it was the same.
func (pm *PhaseManager) SetPhase(phase Phase) bool {
	pm.mu.Lock()
	if pm.phase == phase {
		pm.mu.Unlock()
		return false
	}
	oldPhase := pm.phase
	pm.phase = phase
	pm.mu.Unlock()

	logger.Autopilot().Debug("Phase transition",
		"autopilot_key", pm.autopilotKey,
		"from", oldPhase,
		"to", phase)

	// Pass phase explicitly to avoid TOCTOU between SetPhase and statusGetter
	pm.reportStatusForPhase(phase)
	return true
}

// SetPhaseWithoutReport sets the phase without triggering a status report.
// Useful when the caller needs to do additional work before reporting.
func (pm *PhaseManager) SetPhaseWithoutReport(phase Phase) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.phase == phase {
		return false
	}
	pm.phase = phase
	return true
}

// IsTerminalPhase checks if the current phase is terminal (completed/failed/stopped/max_iterations).
func (pm *PhaseManager) IsTerminalPhase() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return pm.isTerminalPhaseLocked()
}

// isTerminalPhaseLocked checks terminal phase while already holding the lock.
func (pm *PhaseManager) isTerminalPhaseLocked() bool {
	switch pm.phase {
	case PhaseCompleted, PhaseFailed, PhaseStopped, PhaseMaxIterations:
		return true
	default:
		return false
	}
}

// CanProcessIteration checks if the current phase allows processing a new iteration.
func (pm *PhaseManager) CanProcessIteration() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	switch pm.phase {
	case PhasePaused, PhaseStopped, PhaseCompleted, PhaseFailed, PhaseWaitingApproval, PhaseMaxIterations:
		return false
	default:
		return true
	}
}

// TransitionToRunning transitions to running phase if currently initializing.
func (pm *PhaseManager) TransitionToRunning() bool {
	pm.mu.Lock()
	if pm.phase == PhaseInitializing || pm.phase == PhasePaused {
		pm.phase = PhaseRunning
		pm.mu.Unlock()
		pm.reportStatusForPhase(PhaseRunning)
		return true
	}
	pm.mu.Unlock()
	return false
}

// reportStatusForPhase reports status with an explicit phase to avoid TOCTOU.
// The phase is captured at the call site under the lock, ensuring consistency
// between the phase and the status snapshot.
func (pm *PhaseManager) reportStatusForPhase(phase Phase) {
	if pm.reporter == nil || pm.statusGetter == nil {
		return
	}

	status := pm.statusGetter()
	if status == nil {
		return
	}

	status.Phase = string(phase)

	pm.reporter.ReportAutopilotStatus(&runnerv1.AutopilotStatusEvent{
		AutopilotKey: pm.autopilotKey,
		PodKey:       pm.podKey,
		Status:       status,
	})
}

// reportStatus reports status with the current phase (for ReportStatus public API).
func (pm *PhaseManager) reportStatus() {
	pm.reportStatusForPhase(pm.GetPhase())
}

// ReportStatus triggers a status report with current phase.
func (pm *PhaseManager) ReportStatus() {
	pm.reportStatus()
}
