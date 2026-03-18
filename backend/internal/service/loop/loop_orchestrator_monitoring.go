package loop

import (
	"context"
	"fmt"
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

	// Derive effective status using SSOT logic
	autopilotPhase := ""
	if run.AutopilotControllerKey != nil {
		autopilotPhase = o.loopRunService.GetAutopilotPhase(ctx, *run.AutopilotControllerKey)
	}
	effectiveStatus := loopDomain.DeriveRunStatus(podStatus, autopilotPhase)

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

	// Delegate to DeriveRunStatus for consistent mapping (SSOT)
	// Pod status is irrelevant when autopilot phase is terminal — DeriveRunStatus handles this.
	effectiveStatus := loopDomain.DeriveRunStatus("", phase)

	o.HandleRunCompleted(ctx, run, effectiveStatus)
}

// CheckTimeoutRuns detects loop runs that have exceeded their timeout and marks them as timed out.
// orgIDs filters to specific organizations; nil means all orgs (single-instance mode).
// Called periodically by the LoopScheduler.
func (o *LoopOrchestrator) CheckTimeoutRuns(ctx context.Context, orgIDs []int64) error {
	runs, err := o.loopRunService.GetTimedOutRuns(ctx, orgIDs)
	if err != nil {
		o.logger.Error("failed to get timed out runs", "error", err)
		return err
	}

	if len(runs) == 0 {
		return nil
	}

	o.logger.Info("found timed out loop runs", "count", len(runs))

	for _, run := range runs {
		o.HandleRunCompleted(ctx, run, loopDomain.RunStatusTimeout)

		// Terminate the Pod if podTerminator is available
		if run.PodKey != nil && o.podTerminator != nil {
			if termErr := o.podTerminator.TerminatePod(ctx, *run.PodKey); termErr != nil {
				o.logger.Error("failed to terminate timed out pod",
					"pod_key", *run.PodKey,
					"run_id", run.ID,
					"error", termErr,
				)
			}
		}

		o.logger.Info("marked loop run as timed out",
			"run_id", run.ID,
			"loop_id", run.LoopID,
			"pod_key", run.PodKey,
		)
	}

	return nil
}

// CheckApprovalTimeouts detects Autopilot controllers stuck in waiting_approval
// beyond their configured approval_timeout_min and terminates their Pods.
// Without this, a forgotten approval request could hold resources indefinitely
// until the Loop-level timeout_minutes fires (which may be hours).
// orgIDs filters to specific organizations; nil means all orgs (single-instance mode).
func (o *LoopOrchestrator) CheckApprovalTimeouts(ctx context.Context, orgIDs []int64) error {
	if o.autopilotSvc == nil {
		return nil
	}

	timedOut, err := o.autopilotSvc.GetApprovalTimedOut(ctx, orgIDs)
	if err != nil {
		o.logger.Error("failed to get approval-timed-out autopilots", "error", err)
		return err
	}

	if len(timedOut) == 0 {
		return nil
	}

	o.logger.Info("found approval-timed-out autopilot controllers", "count", len(timedOut))

	for _, ac := range timedOut {
		// Mark the autopilot as stopped due to approval timeout
		now := time.Now()
		if updateErr := o.autopilotSvc.UpdateAutopilotControllerStatus(ctx, ac.AutopilotControllerKey, map[string]interface{}{
			"phase":        agentpod.AutopilotPhaseStopped,
			"completed_at": now,
			"updated_at":   now,
		}); updateErr != nil {
			o.logger.Error("failed to update approval-timed-out autopilot",
				"autopilot_key", ac.AutopilotControllerKey, "error", updateErr)
			continue
		}

		// Terminate the Pod to release resources
		if o.podTerminator != nil {
			if termErr := o.podTerminator.TerminatePod(ctx, ac.PodKey); termErr != nil {
				o.logger.Error("failed to terminate approval-timed-out pod",
					"pod_key", ac.PodKey,
					"autopilot_key", ac.AutopilotControllerKey,
					"error", termErr)
			}
		}

		o.logger.Info("stopped autopilot due to approval timeout",
			"autopilot_key", ac.AutopilotControllerKey,
			"pod_key", ac.PodKey,
			"approval_timeout_min", ac.ApprovalTimeoutMin)
	}

	return nil
}

// CleanupOrphanPendingRuns marks pending runs with no Pod that are stuck for > 5 minutes as failed.
// These can occur when StartRun goroutine crashes or the server restarts between TriggerRun and StartRun.
// orgIDs filters to specific organizations; nil means all orgs (single-instance mode).
func (o *LoopOrchestrator) CleanupOrphanPendingRuns(ctx context.Context, orgIDs []int64) error {
	runs, err := o.loopRunService.GetOrphanPendingRuns(ctx, orgIDs)
	if err != nil {
		return err
	}
	if len(runs) == 0 {
		return nil
	}

	o.logger.Info("cleaning up orphan pending runs", "count", len(runs))
	for _, run := range runs {
		_ = o.MarkRunFailed(ctx, run.ID, "Orphan pending run: Pod was never created (server restart or StartRun failure)")
		o.logger.Warn("marked orphan pending run as failed", "run_id", run.ID, "loop_id", run.LoopID)
	}
	return nil
}

// RefreshLoopStats recomputes loop statistics from Pod status (SSOT).
// Call this periodically or after significant events.
func (o *LoopOrchestrator) RefreshLoopStats(ctx context.Context, loopID int64) error {
	total, successful, failed, err := o.loopRunService.ComputeLoopStats(ctx, loopID)
	if err != nil {
		return fmt.Errorf("failed to compute loop stats: %w", err)
	}

	return o.loopService.UpdateStats(ctx, loopID, total, successful, failed)
}

// GetLastPodKey returns the pod_key from the most recent run that has one.
// Used for persistent sandbox resume.
func (o *LoopOrchestrator) GetLastPodKey(ctx context.Context, loopID int64) *string {
	return o.loopRunService.GetLatestPodKey(ctx, loopID)
}

// CheckIdleLoopPods detects Loop Pods that have been idle (agent waiting) longer than
// the loop's idle_timeout_sec and terminates them.
// This handles REPL-style agents (e.g., Claude Code) that don't exit after completing a prompt.
// orgIDs filters to specific organizations; nil means all orgs (single-instance mode).
//
// The run is marked as "completed" (not "cancelled") because the agent has actually finished
// its work — the idle state means it's waiting for the next prompt, not that it was interrupted.
// This is important for persistent sandbox resume: only completed runs update last_pod_key,
// so future runs can resume from this run's sandbox state.
func (o *LoopOrchestrator) CheckIdleLoopPods(ctx context.Context, orgIDs []int64) error {
	runs, err := o.loopRunService.GetIdleLoopPods(ctx, orgIDs)
	if err != nil {
		o.logger.Error("failed to get idle loop pods", "error", err)
		return err
	}

	if len(runs) == 0 {
		return nil
	}

	o.logger.Info("found idle loop pods to terminate", "count", len(runs))

	for _, run := range runs {
		// Mark the run as completed BEFORE terminating the Pod.
		// HandleRunCompleted uses FinishRun with optimistic locking (WHERE finished_at IS NULL),
		// so the subsequent pod_terminated event will be a no-op (already finished).
		o.HandleRunCompleted(ctx, run, loopDomain.RunStatusCompleted)

		// Terminate the Pod to release resources
		if run.PodKey != nil && o.podTerminator != nil {
			if termErr := o.podTerminator.TerminatePod(ctx, *run.PodKey); termErr != nil {
				o.logger.Error("failed to terminate idle loop pod",
					"pod_key", *run.PodKey,
					"run_id", run.ID,
					"loop_id", run.LoopID,
					"error", termErr,
				)
			}
		}

		o.logger.Info("terminated idle loop pod",
			"run_id", run.ID,
			"loop_id", run.LoopID,
			"pod_key", run.PodKey,
		)
	}

	return nil
}
