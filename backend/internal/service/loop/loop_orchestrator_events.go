package loop

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
)

func (o *LoopOrchestrator) HandlePodTerminated(ctx context.Context, podKey string, podStatus string, podFinishedAt *time.Time) {
	run, err := o.loopRunService.FindActiveRunByPodKey(ctx, podKey)
	if err != nil {
		return
	}

	o.logger.Info("handling pod terminated for loop run",
		"pod_key", podKey, "pod_status", podStatus, "run_id", run.ID, "loop_id", run.LoopID)

	autopilotPhase := ""
	if run.AutopilotControllerKey != nil {
		autopilotPhase = o.loopRunService.GetAutopilotPhase(ctx, *run.AutopilotControllerKey)
	}
	effectiveStatus := DeriveRunStatus(podStatus, autopilotPhase)

	if effectiveStatus == loopDomain.RunStatusRunning {
		return
	}

	o.HandleRunCompleted(ctx, run, effectiveStatus)
}

func (o *LoopOrchestrator) HandleAutopilotTerminated(ctx context.Context, autopilotKey string, phase string) {
	if !agentpod.IsAutopilotPhaseTerminal(phase) {
		return
	}

	run, err := o.loopRunService.FindActiveRunByAutopilotKey(ctx, autopilotKey)
	if err != nil {
		return
	}

	o.logger.Info("handling autopilot terminated for loop run",
		"autopilot_key", autopilotKey, "phase", phase, "run_id", run.ID, "loop_id", run.LoopID)

	effectiveStatus := DeriveRunStatus("", phase)

	o.HandleRunCompleted(ctx, run, effectiveStatus)
}
