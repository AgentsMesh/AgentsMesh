package loop

import (
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
)

// ResolveRunStatus resolves the effective status of a run using Pod and Autopilot state.
// This implements SSOT: Pod status is the single source of truth for execution state.
//
// Parameters:
//   - run:              the LoopRun to resolve (modified in-place)
//   - podStatus:        Pod.Status (e.g., "running", "completed", "terminated")
//   - autopilotPhase:   AutopilotController.Phase (empty string if no autopilot)
//   - podFinishedAt:    Pod.FinishedAt (for computing duration)
func ResolveRunStatus(run *loopDomain.LoopRun, podStatus string, autopilotPhase string, podFinishedAt *time.Time) {
	// No Pod → keep the run's own status (pending/skipped/failed)
	if run.PodKey == nil {
		return
	}

	// Derive status from Autopilot phase (if present) or Pod status
	run.Status = DeriveRunStatus(podStatus, autopilotPhase)

	// Derive timing from Pod
	if podFinishedAt != nil {
		run.FinishedAt = podFinishedAt
		if run.StartedAt != nil {
			d := int(podFinishedAt.Sub(*run.StartedAt).Seconds())
			run.DurationSec = &d
		}
	}
}

// DeriveRunStatus maps Pod/Autopilot state to Loop Run status.
//
// Priority logic:
//   - Autopilot terminal phase (completed/failed/stopped) is authoritative
//   - If autopilot is non-terminal but Pod is done, Pod wins (ground truth)
//   - For Direct mode (no autopilot), Pod status is used directly
//
// This handles the case where a Pod is manually terminated while autopilot
// is still in an active phase — the Pod's terminal state is the ground truth.
func DeriveRunStatus(podStatus string, autopilotPhase string) string {
	// Autopilot mode
	if autopilotPhase != "" {
		// Autopilot terminal phases are authoritative
		switch autopilotPhase {
		case agentpod.AutopilotPhaseCompleted:
			return loopDomain.RunStatusCompleted
		case agentpod.AutopilotPhaseFailed:
			return loopDomain.RunStatusFailed
		case agentpod.AutopilotPhaseStopped:
			return loopDomain.RunStatusCancelled
		case agentpod.AutopilotPhaseMaxIterations:
			// max_iterations means "task not finished but iteration quota exhausted".
			// Map to completed — the autopilot did its best within the configured limit.
			return loopDomain.RunStatusCompleted
		default:
			// Autopilot is in active phase — but if Pod is done,
			// Pod's state is the ground truth (SSOT)
			if isPodDoneForLoop(podStatus) {
				return deriveFromPodStatus(podStatus)
			}
			return loopDomain.RunStatusRunning
		}
	}

	// Direct mode: Pod status is the truth
	if isPodDoneForLoop(podStatus) {
		return deriveFromPodStatus(podStatus)
	}
	return loopDomain.RunStatusRunning
}

// isPodDoneForLoop returns true if the Pod is "done" from the Loop domain's perspective.
//
// This deliberately excludes StatusOrphaned — an orphaned pod may reconnect,
// so from Loop's perspective it's still potentially active.
// This differs from Pod.IsTerminal() which includes orphaned.
func isPodDoneForLoop(podStatus string) bool {
	return podStatus == agentpod.StatusCompleted ||
		podStatus == agentpod.StatusTerminated ||
		podStatus == agentpod.StatusError
}

// deriveFromPodStatus maps a "done" Pod status to Loop Run status.
//
// StatusCompleted = Pod process exited naturally → run completed successfully.
// StatusTerminated = Pod was explicitly killed (user cancel, system cleanup) → run cancelled.
// StatusError = Pod encountered an error → run failed.
func deriveFromPodStatus(podStatus string) string {
	switch podStatus {
	case agentpod.StatusCompleted:
		return loopDomain.RunStatusCompleted
	case agentpod.StatusTerminated:
		return loopDomain.RunStatusCancelled
	case agentpod.StatusError:
		return loopDomain.RunStatusFailed
	default:
		return loopDomain.RunStatusFailed
	}
}
