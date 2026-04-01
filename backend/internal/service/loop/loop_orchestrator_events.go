package loop

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
)

// HandlePodTerminated is called when a Pod reaches a terminal state.
// It looks up the associated LoopRun and processes completion.
//
// Uses FindActiveRunByPodKey (no status resolution) because the event payload
// carries the authoritative podStatus — re-querying Pod status would be redundant.
func (o *LoopOrchestrator) HandlePodTerminated(ctx context.Context, podKey string, podStatus string, podFinishedAt *time.Time) {
	run, err := o.loopRunService.FindActiveRunByPodKey(ctx, podKey)
	if err != nil {
		// Not a loop-associated pod, ignore
		return
	}

	o.logger.Info("handling pod terminated for loop run",
		"pod_key", podKey, "pod_status", podStatus, "run_id", run.ID, "loop_id", run.LoopID)

	// Derive effective status using SSOT logic
	autopilotPhase := ""
	if run.AutopilotControllerKey != nil {
		autopilotPhase = o.loopRunService.GetAutopilotPhase(ctx, *run.AutopilotControllerKey)
	}
	effectiveStatus := DeriveRunStatus(podStatus, autopilotPhase)

	// Only process if the run reached a terminal state
	if effectiveStatus == loopDomain.RunStatusRunning {
		return
	}

	o.HandleRunCompleted(ctx, run, effectiveStatus)
}

// HandleAutopilotTerminated is called when an Autopilot reaches a terminal phase.
// It looks up the associated LoopRun and processes completion.
//
// Uses FindActiveRunByAutopilotKey (no status resolution) because the event payload
// carries the authoritative phase — re-querying would be redundant.
// Delegates to DeriveRunStatus for status mapping (SSOT — single mapping location).
func (o *LoopOrchestrator) HandleAutopilotTerminated(ctx context.Context, autopilotKey string, phase string) {
	if !agentpod.IsAutopilotPhaseTerminal(phase) {
		return // Not terminal, ignore
	}

	run, err := o.loopRunService.FindActiveRunByAutopilotKey(ctx, autopilotKey)
	if err != nil {
		// Not a loop-associated autopilot, ignore
		return
	}

	o.logger.Info("handling autopilot terminated for loop run",
		"autopilot_key", autopilotKey, "phase", phase, "run_id", run.ID, "loop_id", run.LoopID)

	// Delegate to DeriveRunStatus for consistent mapping (SSOT)
	// Pod status is irrelevant when autopilot phase is terminal — DeriveRunStatus handles this.
	effectiveStatus := DeriveRunStatus("", phase)

	o.HandleRunCompleted(ctx, run, effectiveStatus)
}
