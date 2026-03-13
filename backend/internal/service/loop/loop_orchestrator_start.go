package loop

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
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
	// (e.g., user cancelled a pending run before the goroutine started)
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

	// Determine permission mode
	permissionMode := loop.PermissionMode
	if permissionMode == "" {
		permissionMode = "bypassPermissions"
	}

	// Build config overrides
	var configOverrides map[string]interface{}
	if loop.ConfigOverrides != nil {
		_ = json.Unmarshal(loop.ConfigOverrides, &configOverrides)
	}

	// Resolve prompt: merge default variables with trigger-time overrides, then substitute {{key}} placeholders
	resolvedPrompt := resolvePrompt(loop.PromptTemplate, loop.PromptVariables, run.TriggerParams)

	// Persist resolved prompt on the run record
	if err := o.loopRunService.UpdateStatus(ctx, run.ID, map[string]interface{}{
		"resolved_prompt": resolvedPrompt,
	}); err != nil {
		o.logger.Error("failed to persist resolved prompt", "run_id", run.ID, "error", err)
	}

	// Determine source pod key for resume (persistent sandbox strategy)
	var sourcePodKey string
	resumeSession := loop.SessionPersistence
	if loop.IsPersistent() && loop.LastPodKey != nil {
		sourcePodKey = *loop.LastPodKey
	}

	// Create Pod via PodOrchestrator
	podResult, err := o.podOrchestrator.CreatePod(ctx, &agentpodSvc.OrchestrateCreatePodRequest{
		OrganizationID:      loop.OrganizationID,
		UserID:              userID,
		RunnerID:            runnerID,
		AgentTypeID:         loop.AgentTypeID,
		CustomAgentTypeID:   loop.CustomAgentTypeID,
		RepositoryID:        loop.RepositoryID,
		TicketID:            loop.TicketID,
		InitialPrompt:       resolvedPrompt,
		BranchName:          loop.BranchName,
		PermissionMode:      &permissionMode,
		CredentialProfileID: loop.CredentialProfileID,
		ConfigOverrides:     configOverrides,
		Cols:                120,
		Rows:                40,
		SourcePodKey:        sourcePodKey,
		ResumeAgentSession:  &resumeSession,
	})
	if err != nil {
		// M3: If resume mode failed, retry without resume (degrade to fresh sandbox)
		if sourcePodKey != "" {
			o.logger.Warn("persistent sandbox resume failed, degrading to fresh",
				"loop_id", loop.ID, "run_id", run.ID, "source_pod_key", sourcePodKey, "error", err)

			// Notify frontend about the degradation
			o.publishWarningEvent(loop.OrganizationID, loop.ID, run.ID, run.RunNumber,
				"sandbox_resume_degraded",
				fmt.Sprintf("Resume from pod %s failed: %v. Degraded to fresh sandbox.", sourcePodKey, err))

			podResult, err = o.podOrchestrator.CreatePod(ctx, &agentpodSvc.OrchestrateCreatePodRequest{
				OrganizationID:      loop.OrganizationID,
				UserID:              userID,
				RunnerID:            runnerID,
				AgentTypeID:         loop.AgentTypeID,
				CustomAgentTypeID:   loop.CustomAgentTypeID,
				RepositoryID:        loop.RepositoryID,
				TicketID:            loop.TicketID,
				InitialPrompt:       resolvedPrompt,
				BranchName:          loop.BranchName,
				PermissionMode:      &permissionMode,
				CredentialProfileID: loop.CredentialProfileID,
				ConfigOverrides:     configOverrides,
				Cols:                120,
				Rows:                40,
				// No SourcePodKey — fresh start
			})
			if err != nil {
				_ = o.MarkRunFailed(ctx, run.ID, fmt.Sprintf("Pod creation failed (after resume degradation): %v", err))
				return
			}

			// Clear the stale resume chain so future runs don't keep failing
			_ = o.loopService.ClearRuntimeState(ctx, loop.ID)
		} else {
			_ = o.MarkRunFailed(ctx, run.ID, fmt.Sprintf("Pod creation failed: %v", err))
			return
		}
	}

	pod := podResult.Pod
	autopilotKey := ""

	// If autopilot mode, create AutopilotController via the encapsulated service method
	if loop.IsAutopilot() && o.autopilotSvc != nil {
		var err error
		autopilotKey, err = o.startAutopilot(ctx, loop, run, pod, resolvedPrompt)
		if err != nil {
			o.logger.Error("autopilot creation failed, terminating Pod",
				"run_id", run.ID, "pod_key", pod.PodKey, "error", err)
			// Terminate the orphan Pod — nothing will drive it without Autopilot
			if o.podTerminator != nil {
				_ = o.podTerminator.TerminatePod(ctx, pod.PodKey)
			}
			_ = o.MarkRunFailed(ctx, run.ID, fmt.Sprintf("Autopilot creation failed: %v", err))
			return
		}
	}

	// Associate Pod with run — after this, run status is derived from Pod (SSOT)
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

// startAutopilot delegates Autopilot creation to AutopilotControllerService.CreateAndStart.
// Returns the autopilot controller key and any error.
func (o *LoopOrchestrator) startAutopilot(ctx context.Context, loop *loopDomain.Loop, run *loopDomain.LoopRun, pod *agentpod.Pod, resolvedPrompt string) (string, error) {
	// Extract autopilot config via typed struct (all zeros → domain defaults apply)
	apCfg := loop.ParseAutopilotConfig()

	controller, err := o.autopilotSvc.CreateAndStart(ctx, &agentpodSvc.CreateAndStartRequest{
		OrganizationID:      loop.OrganizationID,
		Pod:                 pod,
		InitialPrompt:       resolvedPrompt,
		MaxIterations:       apCfg.MaxIterations,
		IterationTimeoutSec: apCfg.IterationTimeoutSec,
		NoProgressThreshold: apCfg.NoProgressThreshold,
		SameErrorThreshold:  apCfg.SameErrorThreshold,
		ApprovalTimeoutMin:  apCfg.ApprovalTimeoutMin,
		KeyPrefix:           fmt.Sprintf("loop-%s-run%d", loop.Slug, run.RunNumber),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create autopilot controller: %w", err)
	}

	return controller.AutopilotControllerKey, nil
}
