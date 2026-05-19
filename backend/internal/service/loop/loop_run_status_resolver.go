package loop

import (
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
)

// ResolveRunStatus derives effective run.Status from Pod (SSOT) + Autopilot phase.
func ResolveRunStatus(run *loopDomain.LoopRun, podStatus string, autopilotPhase string, podFinishedAt *time.Time) {
	if run.PodKey == nil {
		return
	}

	run.Status = DeriveRunStatus(podStatus, autopilotPhase)

	if podFinishedAt != nil {
		run.FinishedAt = podFinishedAt
		if run.StartedAt != nil {
			d := int(podFinishedAt.Sub(*run.StartedAt).Seconds())
			run.DurationSec = &d
		}
	}
}

func DeriveRunStatus(podStatus string, autopilotPhase string) string {
	if autopilotPhase != "" {
		switch autopilotPhase {
		case agentpod.AutopilotPhaseCompleted:
			return loopDomain.RunStatusCompleted
		case agentpod.AutopilotPhaseFailed:
			return loopDomain.RunStatusFailed
		case agentpod.AutopilotPhaseStopped:
			return loopDomain.RunStatusCancelled
		case agentpod.AutopilotPhaseMaxIterations:
			// MaxIterations = "best-effort within iteration quota" → still counts as completed.
			return loopDomain.RunStatusCompleted
		default:
			// Pod status is the SSOT — overrides any non-terminal autopilot phase.
			if isPodDoneForLoop(podStatus) {
				return deriveFromPodStatus(podStatus)
			}
			return loopDomain.RunStatusRunning
		}
	}

	if isPodDoneForLoop(podStatus) {
		return deriveFromPodStatus(podStatus)
	}
	return loopDomain.RunStatusRunning
}

func isPodDoneForLoop(podStatus string) bool {
	return podStatus == agentpod.StatusCompleted ||
		podStatus == agentpod.StatusTerminated ||
		podStatus == agentpod.StatusError
}

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
