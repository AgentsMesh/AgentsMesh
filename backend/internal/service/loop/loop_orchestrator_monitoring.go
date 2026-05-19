package loop

import (
	"context"
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
)

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

// CleanupOrphanPendingRuns reaps pending runs without a Pod after >5min
// (StartRun goroutine crash or server restart between TriggerRun and StartRun).
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

func (o *LoopOrchestrator) RefreshLoopStats(ctx context.Context, loopID int64) error {
	total, successful, failed, err := o.loopRunService.ComputeLoopStats(ctx, loopID)
	if err != nil {
		o.logger.Error("failed to compute loop stats", "loop_id", loopID, "error", err)
		return fmt.Errorf("failed to compute loop stats: %w", err)
	}

	if err := o.loopService.UpdateStats(ctx, loopID, total, successful, failed); err != nil {
		o.logger.Error("failed to update loop stats", "loop_id", loopID, "error", err)
		return err
	}

	return nil
}

func (o *LoopOrchestrator) GetLastPodKey(ctx context.Context, loopID int64) *string {
	return o.loopRunService.GetLatestPodKey(ctx, loopID)
}

// CheckIdleLoopPods terminates Loop Pods idle past idle_timeout_sec (REPL agents
// like Claude Code never exit after a prompt). Marks as "completed" not "cancelled"
// so last_pod_key updates and future runs can resume from this run's sandbox state.
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
		o.HandleRunCompleted(ctx, run, loopDomain.RunStatusCompleted)

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
