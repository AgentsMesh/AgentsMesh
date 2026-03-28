package loop

import (
	"context"
	"time"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

// SetRunPodKey associates a Pod with a run and marks it as running.
// After this, the run's effective status is derived from Pod (SSOT).
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

// MarkRunFailed marks a run as failed when Pod creation or Autopilot setup fails.
// This is only used when no Pod exists (or Pod was cleaned up) — because no SSOT is available.
func (o *LoopOrchestrator) MarkRunFailed(ctx context.Context, runID int64, errorMessage string) error {
	o.logger.Warn("marking run as failed", "run_id", runID, "error_message", errorMessage)
	return o.markRunTerminal(ctx, runID, loopDomain.RunStatusFailed, errorMessage)
}

// MarkRunCancelled marks a run as cancelled (e.g. user-initiated cancellation of a pending run).
func (o *LoopOrchestrator) MarkRunCancelled(ctx context.Context, runID int64, reason string) error {
	o.logger.Info("marking run as cancelled", "run_id", runID, "reason", reason)
	return o.markRunTerminal(ctx, runID, loopDomain.RunStatusCancelled, reason)
}

// markRunTerminal sets a run to a terminal state directly (bypassing Pod SSOT).
// Used only when no Pod exists to derive status from.
// Uses FinishRun (WHERE finished_at IS NULL) for idempotency — concurrent calls are no-ops.
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
		// Already finished by a concurrent call — skip side-effects
		return nil
	}

	// Publish event — cancelled counts as failed (consistent with stats)
	run, _ := o.loopRunService.GetByID(ctx, runID)
	if run != nil {
		o.publishRunEvent(run.OrganizationID, eventbus.EventLoopRunFailed, run)
		// Update loop stats (incremental)
		_ = o.loopService.UpdateRunStats(ctx, run.LoopID, status, now)
	}
	return nil
}

// HandleRunCompleted processes a completed loop run.
// Called when a Pod or Autopilot reaches a terminal state.
//
// Responsibilities:
//  1. Update Loop run statistics (incremental counters)
//  2. Update runtime state (last_pod_key for persistent sandbox resume)
//  3. Publish completion event
func (o *LoopOrchestrator) HandleRunCompleted(ctx context.Context, run *loopDomain.LoopRun, effectiveStatus string) {
	now := time.Now()

	// 0. Atomically mark the run as finished (optimistic locking via WHERE finished_at IS NULL).
	// If another concurrent event already finished this run, FinishRun returns false and we skip
	// all side-effects (stats, events, webhooks) to prevent double-counting.
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
		// Already finished by a concurrent event — skip all side-effects
		o.logger.Debug("run already finished, skipping duplicate completion",
			"run_id", run.ID)
		return
	}

	// Sync the in-memory struct so downstream consumers (publishRunEvent, webhook, ticket comment)
	// see the resolved status instead of the stale DB value (e.g. "pending").
	run.Status = effectiveStatus
	run.FinishedAt = &now

	// 1. Update loop run statistics (incremental)
	if err := o.loopService.UpdateRunStats(ctx, run.LoopID, effectiveStatus, now); err != nil {
		o.logger.Error("failed to update loop run stats",
			"loop_id", run.LoopID, "run_id", run.ID, "error", err)
	}

	// 2. Update runtime state for persistent sandbox resume.
	loop, _ := o.loopService.GetByID(ctx, run.LoopID)
	if run.PodKey != nil && loop != nil && loop.IsPersistent() {
		switch effectiveStatus {
		case loopDomain.RunStatusCompleted:
			// Successful completion: advance the resume chain to this run's pod.
			if err := o.loopService.UpdateRuntimeState(ctx, run.LoopID, nil, run.PodKey); err != nil {
				o.logger.Error("failed to update loop runtime state",
					"loop_id", run.LoopID, "error", err)
			}
		case loopDomain.RunStatusFailed:
			// Failed run: clear the resume chain to break the death spiral.
			// Without this, async runner errors (e.g., "mcp.json escapes sandbox root")
			// cause every subsequent run to retry the same broken resume, since the
			// degradation logic in StartRun only catches synchronous CreatePod errors.
			// Clearing last_pod_key forces the next run to start fresh.
			if err := o.loopService.ClearRuntimeState(ctx, run.LoopID); err != nil {
				o.logger.Error("failed to clear loop runtime state after failure",
					"loop_id", run.LoopID, "error", err)
			}
			o.logger.Info("cleared persistent sandbox resume chain after run failure",
				"loop_id", run.LoopID, "run_id", run.ID, "pod_key", *run.PodKey)
		}
	}

	// 3. Publish completion event — cancelled/failed/timeout all emit RunFailed
	eventType := eventbus.EventLoopRunCompleted
	if effectiveStatus == loopDomain.RunStatusFailed || effectiveStatus == loopDomain.RunStatusTimeout || effectiveStatus == loopDomain.RunStatusCancelled {
		eventType = eventbus.EventLoopRunFailed
	}
	o.publishRunEvent(run.OrganizationID, eventType, run)

	// 4. Webhook callback (async, fire-and-forget)
	if loop != nil && loop.CallbackURL != nil && *loop.CallbackURL != "" {
		go o.sendWebhookCallback(*loop.CallbackURL, loop, run, effectiveStatus)
	}

	// 5. Ticket comment (async — use background context since the parent ctx may be cancelled)
	if loop != nil && loop.TicketID != nil && o.ticketService != nil {
		go o.postTicketComment(context.Background(), *loop.TicketID, loop.CreatedByID, loop, run, effectiveStatus)
	}

	// 6. Data retention — trim old finished runs if max_retained_runs is configured
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

