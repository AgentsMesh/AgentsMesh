package loop

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
)

var ErrHasActiveRuns = errors.New("loop has active runs")

// Delete deletes a Loop (hard delete).
// Atomically checks for active runs — returns ErrHasActiveRuns if any exist.
func (s *LoopService) Delete(ctx context.Context, orgID int64, slug string) error {
	affected, err := s.repo.Delete(ctx, orgID, slug)
	if err != nil {
		if errors.Is(err, loopDomain.ErrHasActiveRuns) {
			return ErrHasActiveRuns
		}
		slog.Error("failed to delete loop", "slug", slug, "org_id", orgID, "error", err)
		return err
	}
	if affected == 0 {
		return ErrLoopNotFound
	}
	slog.Info("loop deleted", "slug", slug, "org_id", orgID)
	return nil
}

var validStatuses = map[string]bool{
	loopDomain.StatusEnabled:  true,
	loopDomain.StatusDisabled: true,
}

// SetStatus updates the status of a Loop.
// When re-enabling a cron loop, recalculates next_run_at so cron scheduling resumes immediately.
func (s *LoopService) SetStatus(ctx context.Context, orgID int64, slug string, status string) (*loopDomain.Loop, error) {
	if !validStatuses[status] {
		return nil, fmt.Errorf("%w: status must be 'enabled' or 'disabled'", ErrInvalidEnumValue)
	}

	loop, err := s.GetBySlug(ctx, orgID, slug)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"status": status,
	}

	if status == loopDomain.StatusEnabled && loop.HasCron() {
		schedule, err := cronParser.Parse(*loop.CronExpression)
		if err == nil {
			next := schedule.Next(time.Now())
			updates["next_run_at"] = next
		}
	}

	if err := s.repo.Update(ctx, loop.ID, updates); err != nil {
		slog.Error("failed to set loop status", "loop_id", loop.ID, "slug", slug, "status", status, "error", err)
		return nil, err
	}

	slog.Info("loop status changed", "loop_id", loop.ID, "slug", slug, "org_id", orgID, "status", status)
	return s.GetBySlug(ctx, orgID, slug)
}
