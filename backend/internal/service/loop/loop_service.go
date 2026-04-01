package loop

import (
	"context"
	"errors"
	"log/slog"
	"time"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
)

// LoopService handles Loop CRUD operations.
type LoopService struct {
	repo loopDomain.LoopRepository
}

func NewLoopService(repo loopDomain.LoopRepository) *LoopService {
	return &LoopService{repo: repo}
}

// GetBySlug retrieves a Loop by organization ID and slug.
func (s *LoopService) GetBySlug(ctx context.Context, orgID int64, slug string) (*loopDomain.Loop, error) {
	loop, err := s.repo.GetBySlug(ctx, orgID, slug)
	if err != nil {
		if errors.Is(err, loopDomain.ErrNotFound) {
			return nil, ErrLoopNotFound
		}
		return nil, err
	}
	return loop, nil
}

func (s *LoopService) GetByID(ctx context.Context, id int64) (*loopDomain.Loop, error) {
	loop, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, loopDomain.ErrNotFound) {
			return nil, ErrLoopNotFound
		}
		return nil, err
	}
	return loop, nil
}

func (s *LoopService) List(ctx context.Context, filter *ListLoopsFilter) ([]*loopDomain.Loop, int64, error) {
	return s.repo.List(ctx, filter)
}

func (s *LoopService) UpdateRunStats(ctx context.Context, loopID int64, status string, lastRunAt time.Time) error {
	if err := s.repo.IncrementRunStats(ctx, loopID, status, lastRunAt); err != nil {
		slog.Error("failed to update loop run stats", "loop_id", loopID, "status", status, "error", err)
		return err
	}
	return nil
}

// UpdateStats sets the run statistics on a Loop to absolute values.
func (s *LoopService) UpdateStats(ctx context.Context, loopID int64, total, successful, failed int) error {
	return s.repo.Update(ctx, loopID, map[string]interface{}{
		"total_runs":      total,
		"successful_runs": successful,
		"failed_runs":     failed,
	})
}

func (s *LoopService) ClearRuntimeState(ctx context.Context, loopID int64) error {
	if err := s.repo.Update(ctx, loopID, map[string]interface{}{
		"sandbox_path": nil,
		"last_pod_key": nil,
	}); err != nil {
		slog.Error("failed to clear loop runtime state", "loop_id", loopID, "error", err)
		return err
	}
	slog.Info("loop runtime state cleared", "loop_id", loopID)
	return nil
}

func (s *LoopService) UpdateRuntimeState(ctx context.Context, loopID int64, sandboxPath *string, lastPodKey *string) error {
	updates := map[string]interface{}{}
	if sandboxPath != nil {
		updates["sandbox_path"] = *sandboxPath
	}
	if lastPodKey != nil {
		updates["last_pod_key"] = *lastPodKey
	}
	if len(updates) == 0 {
		return nil
	}
	return s.repo.Update(ctx, loopID, updates)
}

func (s *LoopService) UpdateNextRunAt(ctx context.Context, loopID int64, nextRunAt *time.Time) error {
	return s.repo.Update(ctx, loopID, map[string]interface{}{
		"next_run_at": nextRunAt,
	})
}

// GetDueCronLoops returns enabled loops with cron scheduling that are due for execution.
func (s *LoopService) GetDueCronLoops(ctx context.Context, orgIDs []int64) ([]*loopDomain.Loop, error) {
	return s.repo.GetDueCronLoops(ctx, orgIDs)
}

// ClaimCronLoop atomically claims a cron loop with SKIP LOCKED and advances next_run_at.
func (s *LoopService) ClaimCronLoop(ctx context.Context, loopID int64, nextRunAt *time.Time) (bool, error) {
	return s.repo.ClaimCronLoop(ctx, loopID, nextRunAt)
}

// FindLoopsNeedingNextRun returns enabled cron loops with next_run_at IS NULL.
func (s *LoopService) FindLoopsNeedingNextRun(ctx context.Context, orgIDs []int64) ([]*loopDomain.Loop, error) {
	return s.repo.FindLoopsNeedingNextRun(ctx, orgIDs)
}
