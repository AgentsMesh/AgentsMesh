package loop

import (
	"context"
	"time"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
)

// CheckAndTriggerCronLoops checks for loops with cron scheduling that are due and triggers them.
// Uses FOR UPDATE SKIP LOCKED within per-loop transactions for multi-instance safety:
// each loop is claimed atomically so only one instance processes it.
// Queries are scoped to local orgs via LocalOrgProvider.
func (s *LoopScheduler) CheckAndTriggerCronLoops(ctx context.Context) error {
	orgIDs := s.getOrgIDs()

	// Get candidate loops (lightweight query, no lock, org-scoped)
	dueLoops, err := s.loopService.GetDueCronLoops(ctx, orgIDs)
	if err != nil {
		s.logger.Error("failed to get due cron loops", "error", err)
		return err
	}

	if len(dueLoops) == 0 {
		return nil
	}

	s.logger.Info("found due cron loops", "count", len(dueLoops))

	for _, loop := range dueLoops {
		// Calculate next_run_at before claiming so we can advance it atomically
		var nextRunAt *time.Time
		if loop.CronExpression != nil {
			var calcErr error
			nextRunAt, calcErr = s.CalculateNextRun(*loop.CronExpression)
			if calcErr != nil {
				s.logger.Error("invalid cron expression, skipping loop",
					"loop_id", loop.ID, "cron", *loop.CronExpression, "error", calcErr)
				continue
			}
		}

		// Try to claim this loop with a short transaction
		claimed, err := s.loopService.ClaimCronLoop(ctx, loop.ID, nextRunAt)
		if err != nil {
			s.logger.Error("failed to claim cron loop", "loop_id", loop.ID, "error", err)
			continue
		}
		if !claimed {
			continue // Another instance claimed it
		}

		// Trigger run outside transaction (lock already released, next_run_at already advanced)
		result, err := s.orchestrator.TriggerRun(ctx, &TriggerRunRequest{
			LoopID:        loop.ID,
			TriggerType:   loopDomain.RunTriggerCron,
			TriggerSource: "cron",
		})
		if err != nil {
			s.logger.Error("failed to trigger cron loop", "loop_id", loop.ID, "error", err)
			continue
		}

		// Start the run asynchronously (Pod creation + Autopilot setup)
		// Uses the loop creator as the acting user for cron-triggered runs.
		if !result.Skipped && result.Run != nil && result.Loop != nil {
			go s.orchestrator.StartRun(context.Background(), result.Loop, result.Run, result.Loop.CreatedByID)
		}
	}

	return nil
}
