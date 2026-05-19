package loop

import (
	"context"
	"time"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
)

// CheckAndTriggerCronLoops uses FOR UPDATE SKIP LOCKED in per-loop tx so multi-instance
// deployments never double-process a single loop.
func (s *LoopScheduler) CheckAndTriggerCronLoops(ctx context.Context) error {
	orgIDs := s.getOrgIDs()

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
		// Compute nextRunAt before claim so ClaimCronLoop advances it atomically with the claim.
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

		claimed, err := s.loopService.ClaimCronLoop(ctx, loop.ID, nextRunAt)
		if err != nil {
			s.logger.Error("failed to claim cron loop", "loop_id", loop.ID, "error", err)
			continue
		}
		if !claimed {
			continue
		}

		result, err := s.orchestrator.TriggerRun(ctx, &TriggerRunRequest{
			LoopID:        loop.ID,
			TriggerType:   loopDomain.RunTriggerCron,
			TriggerSource: "cron",
		})
		if err != nil {
			s.logger.Error("failed to trigger cron loop", "loop_id", loop.ID, "error", err)
			continue
		}

		if !result.Skipped && result.Run != nil && result.Loop != nil {
			go s.orchestrator.StartRun(context.Background(), result.Loop, result.Run, result.Loop.CreatedByID)
		}
	}

	return nil
}
