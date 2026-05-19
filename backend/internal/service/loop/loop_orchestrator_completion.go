package loop

import (
	"context"
	"time"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

func (o *LoopOrchestrator) SetRunPodKey(ctx context.Context, runID int64, podKey string, autopilotKey string) error {
	updates := map[string]interface{}{
		"pod_key": podKey,
	}
	if autopilotKey != "" {
		updates["autopilot_controller_key"] = autopilotKey
	}
	if err := o.loopRunService.UpdateStatus(ctx, runID, updates); err != nil {
		o.logger.Error("failed to set run pod key", "run_id", runID, "pod_key", podKey, "error", err)
		return err
	}
	o.logger.Info("run pod key set", "run_id", runID, "pod_key", podKey, "autopilot_key", autopilotKey)
	return nil
}

// MarkRunFailed is the no-Pod fallback path — bypasses Pod SSOT.
func (o *LoopOrchestrator) MarkRunFailed(ctx context.Context, runID int64, errorMessage string) error {
	o.logger.Warn("marking run as failed", "run_id", runID, "error_message", errorMessage)
	return o.markRunTerminal(ctx, runID, loopDomain.RunStatusFailed, errorMessage)
}

func (o *LoopOrchestrator) MarkRunCancelled(ctx context.Context, runID int64, reason string) error {
	o.logger.Info("marking run as cancelled", "run_id", runID, "reason", reason)
	return o.markRunTerminal(ctx, runID, loopDomain.RunStatusCancelled, reason)
}

// markRunTerminal uses FinishRun's WHERE finished_at IS NULL guard for idempotency under concurrent calls.
func (o *LoopOrchestrator) markRunTerminal(ctx context.Context, runID int64, status string, errorMessage string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":        status,
		"finished_at":   now,
		"error_message": errorMessage,
	}
	updated, err := o.loopRunService.FinishRun(ctx, runID, updates)
	if err != nil {
		return err
	}
	if !updated {
		return nil
	}

	run, _ := o.loopRunService.GetByID(ctx, runID)
	if run != nil {
		o.publishRunEvent(run.OrganizationID, eventbus.EventLoopRunFailed, run)
		_ = o.loopService.UpdateRunStats(ctx, run.LoopID, status, now)
	}
	return nil
}

func (o *LoopOrchestrator) HandleRunCompleted(ctx context.Context, run *loopDomain.LoopRun, effectiveStatus string) {
	now := time.Now()

	// FinishRun's WHERE finished_at IS NULL is the atomic guard against double-counting
	// when concurrent events both try to complete the same run.
	runUpdates := map[string]interface{}{
		"status":      effectiveStatus,
		"finished_at": now,
	}
	if run.StartedAt != nil {
		durationSec := int(now.Sub(*run.StartedAt).Seconds())
		runUpdates["duration_sec"] = durationSec
	}
	updated, err := o.loopRunService.FinishRun(ctx, run.ID, runUpdates)
	if err != nil {
		o.logger.Error("failed to mark run as finished",
			"run_id", run.ID, "error", err)
		return
	}
	if !updated {
		o.logger.Debug("run already finished, skipping duplicate completion",
			"run_id", run.ID)
		return
	}

	run.Status = effectiveStatus
	run.FinishedAt = &now

	if err := o.loopService.UpdateRunStats(ctx, run.LoopID, effectiveStatus, now); err != nil {
		o.logger.Error("failed to update loop run stats",
			"loop_id", run.LoopID, "run_id", run.ID, "error", err)
	}

	loop, _ := o.loopService.GetByID(ctx, run.LoopID)
	if run.PodKey != nil && loop != nil && loop.IsPersistent() {
		switch effectiveStatus {
		case loopDomain.RunStatusCompleted:
			if err := o.loopService.UpdateRuntimeState(ctx, run.LoopID, nil, run.PodKey); err != nil {
				o.logger.Error("failed to update loop runtime state",
					"loop_id", run.LoopID, "error", err)
			}
		case loopDomain.RunStatusFailed:
			if err := o.loopService.ClearRuntimeState(ctx, run.LoopID); err != nil {
				o.logger.Error("failed to clear loop runtime state after failure",
					"loop_id", run.LoopID, "error", err)
			}
			o.logger.Info("cleared persistent sandbox resume chain after run failure",
				"loop_id", run.LoopID, "run_id", run.ID, "pod_key", *run.PodKey)
		}
	}

	eventType := eventbus.EventLoopRunCompleted
	if effectiveStatus == loopDomain.RunStatusFailed || effectiveStatus == loopDomain.RunStatusTimeout || effectiveStatus == loopDomain.RunStatusCancelled {
		eventType = eventbus.EventLoopRunFailed
	}
	o.publishRunEvent(run.OrganizationID, eventType, run)

	if loop != nil && loop.CallbackURL != nil && *loop.CallbackURL != "" {
		go o.sendWebhookCallback(*loop.CallbackURL, loop, run, effectiveStatus)
	}

	if loop != nil && loop.TicketID != nil && o.ticketService != nil {
		go o.postTicketComment(context.Background(), *loop.TicketID, loop.CreatedByID, loop, run, effectiveStatus)
	}

	// Trim by loop.MaxRetainedRuns (data retention).
	if loop != nil && loop.MaxRetainedRuns > 0 {
		if deleted, err := o.loopRunService.DeleteOldFinishedRuns(ctx, loop.ID, loop.MaxRetainedRuns); err != nil {
			o.logger.Error("failed to trim old loop runs",
				"loop_id", loop.ID, "max_retained", loop.MaxRetainedRuns, "error", err)
		} else if deleted > 0 {
			o.logger.Info("trimmed old loop runs",
				"loop_id", loop.ID, "deleted", deleted, "max_retained", loop.MaxRetainedRuns)
		}
	}

	o.logger.Info("loop run completed",
		"loop_id", run.LoopID,
		"run_id", run.ID,
		"effective_status", effectiveStatus,
		"pod_key", run.PodKey,
	)
}
