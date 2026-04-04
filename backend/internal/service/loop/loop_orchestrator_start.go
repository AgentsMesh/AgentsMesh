package loop

import (
	"context"
	"fmt"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	agentpodSvc "github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
)

// StartRun creates a Pod and optionally an AutopilotController for the loop run.
// Should be called asynchronously (in a goroutine) after TriggerRun returns successfully.
func (o *LoopOrchestrator) StartRun(ctx context.Context, loop *loopDomain.Loop, run *loopDomain.LoopRun, userID int64) {
	// Panic recovery — this method is always called in a goroutine, so panics would crash the process
	defer func() {
		if r := recover(); r != nil {
			o.logger.Error("panic in StartRun", "run_id", run.ID, "loop_id", loop.ID, "panic", r)
			_ = o.MarkRunFailed(ctx, run.ID, fmt.Sprintf("Internal error: %v", r))
		}
	}()

	if o.podOrchestrator == nil {
		o.logger.Error("pod orchestrator not set, cannot start run", "run_id", run.ID)
		_ = o.MarkRunFailed(ctx, run.ID, "Pod orchestrator not configured")
		return
	}

	// Check if the run was cancelled between TriggerRun and StartRun
	currentRun, err := o.loopRunService.GetByID(ctx, run.ID)
	if err != nil {
		o.logger.Error("failed to check run status before start", "run_id", run.ID, "error", err)
		return
	}
	if currentRun.FinishedAt != nil || currentRun.IsTerminal() {
		o.logger.Info("run already finished/cancelled before StartRun, skipping",
			"run_id", run.ID, "status", currentRun.Status)
		return
	}

	// Determine runner ID
	var runnerID int64
	if loop.RunnerID != nil {
		runnerID = *loop.RunnerID
	}

	// Resolve prompt
	resolvedPrompt := resolvePrompt(loop.PromptTemplate, loop.PromptVariables, run.TriggerParams)
	if err := o.loopRunService.UpdateStatus(ctx, run.ID, map[string]interface{}{
		"resolved_prompt": resolvedPrompt,
	}); err != nil {
		o.logger.Error("failed to persist resolved prompt", "run_id", run.ID, "error", err)
	}

	// Build AgentFile Layer from loop configuration (AgentFile SSOT)
	agentfileLayer := o.buildLoopAgentfileLayer(ctx, loop, resolvedPrompt)

	// Determine source pod key for resume (persistent sandbox strategy)
	var sourcePodKey string
	resumeSession := loop.SessionPersistence
	if loop.IsPersistent() && loop.LastPodKey != nil {
		sourcePodKey = *loop.LastPodKey
	}

	// Create Pod via PodOrchestrator
	podResult, err := o.podOrchestrator.CreatePod(ctx, &agentpodSvc.OrchestrateCreatePodRequest{
		OrganizationID:     loop.OrganizationID,
		UserID:             userID,
		RunnerID:           runnerID,
		AgentSlug:          loop.AgentSlug,
		TicketID:           loop.TicketID,
		AgentfileLayer:     &agentfileLayer,
		Cols:               120,
		Rows:               40,
		SourcePodKey:       sourcePodKey,
		ResumeAgentSession: &resumeSession,
	})
	if err != nil {
		// M3: If resume mode failed, retry without resume (degrade to fresh sandbox)
		if sourcePodKey != "" {
			o.logger.Warn("persistent sandbox resume failed, degrading to fresh",
				"loop_id", loop.ID, "run_id", run.ID, "source_pod_key", sourcePodKey, "error", err)

			o.publishWarningEvent(loop.OrganizationID, loop.ID, run.ID, run.RunNumber,
				"sandbox_resume_degraded",
				fmt.Sprintf("Resume from pod %s failed: %v. Degraded to fresh sandbox.", sourcePodKey, err))

			podResult, err = o.podOrchestrator.CreatePod(ctx, &agentpodSvc.OrchestrateCreatePodRequest{
				OrganizationID: loop.OrganizationID,
				UserID:         userID,
				RunnerID:       runnerID,
				AgentSlug:      loop.AgentSlug,
				TicketID:       loop.TicketID,
				AgentfileLayer:   &agentfileLayer,
				Cols:           120,
				Rows:           40,
			})
			if err != nil {
				_ = o.MarkRunFailed(ctx, run.ID, fmt.Sprintf("Pod creation failed (after resume degradation): %v", err))
				return
			}
			_ = o.loopService.ClearRuntimeState(ctx, loop.ID)
		} else {
			_ = o.MarkRunFailed(ctx, run.ID, fmt.Sprintf("Pod creation failed: %v", err))
			return
		}
	}

	pod := podResult.Pod
	autopilotKey := ""

	// If autopilot mode, create AutopilotController
	if loop.IsAutopilot() && o.autopilotSvc != nil {
		var err error
		autopilotKey, err = o.startAutopilot(ctx, loop, run, pod, resolvedPrompt)
		if err != nil {
			o.logger.Error("autopilot creation failed, terminating Pod",
				"run_id", run.ID, "pod_key", pod.PodKey, "error", err)
			if o.podTerminator != nil {
				_ = o.podTerminator.TerminatePod(ctx, pod.PodKey)
			}
			_ = o.MarkRunFailed(ctx, run.ID, fmt.Sprintf("Autopilot creation failed: %v", err))
			return
		}
	}

	// Associate Pod with run
	if err := o.SetRunPodKey(ctx, run.ID, pod.PodKey, autopilotKey); err != nil {
		o.logger.Error("failed to set run pod key", "run_id", run.ID, "error", err)
	}

	o.logger.Info("loop run started",
		"loop_id", loop.ID,
		"run_id", run.ID,
		"pod_key", pod.PodKey,
		"autopilot_key", autopilotKey,
		"execution_mode", loop.ExecutionMode,
	)
}

