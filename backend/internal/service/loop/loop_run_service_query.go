package loop

import (
	"context"
	"log/slog"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
)

func (s *LoopRunService) GetTimedOutRuns(ctx context.Context, orgIDs []int64) ([]*loopDomain.LoopRun, error) {
	return s.repo.GetTimedOutRuns(ctx, orgIDs)
}

// GetOrphanPendingRuns returns pending runs with no pod_key stuck for > 5 minutes.
func (s *LoopRunService) GetOrphanPendingRuns(ctx context.Context, orgIDs []int64) ([]*loopDomain.LoopRun, error) {
	return s.repo.GetOrphanPendingRuns(ctx, orgIDs)
}

func (s *LoopRunService) GetIdleLoopPods(ctx context.Context, orgIDs []int64) ([]*loopDomain.LoopRun, error) {
	return s.repo.GetIdleLoopPods(ctx, orgIDs)
}

// ComputeLoopStats computes run statistics from Pod status (SSOT).
func (s *LoopRunService) ComputeLoopStats(ctx context.Context, loopID int64) (total int, successful int, failed int, err error) {
	return s.repo.ComputeLoopStats(ctx, loopID)
}

func (s *LoopRunService) GetLatestPodKey(ctx context.Context, loopID int64) *string {
	return s.repo.GetLatestPodKey(ctx, loopID)
}

func (s *LoopRunService) CountActiveRunsByLoopIDs(ctx context.Context, loopIDs []int64) (map[int64]int64, error) {
	return s.repo.CountActiveRunsByLoopIDs(ctx, loopIDs)
}

func (s *LoopRunService) GetAvgDuration(ctx context.Context, loopID int64) (*float64, error) {
	return s.repo.GetAvgDuration(ctx, loopID)
}

func (s *LoopRunService) DeleteOldFinishedRuns(ctx context.Context, loopID int64, keep int) (int64, error) {
	deleted, err := s.repo.DeleteOldFinishedRuns(ctx, loopID, keep)
	if err != nil {
		slog.ErrorContext(ctx, "failed to delete old finished runs", "loop_id", loopID, "keep", keep, "error", err)
		return 0, err
	}
	if deleted > 0 {
		slog.InfoContext(ctx, "old finished runs deleted", "loop_id", loopID, "deleted", deleted, "keep", keep)
	}
	return deleted, nil
}

func (s *LoopRunService) GetAutopilotPhase(ctx context.Context, autopilotKey string) string {
	phases, err := s.repo.BatchGetAutopilotPhases(ctx, []string{autopilotKey})
	if err != nil || phases == nil {
		return ""
	}
	return phases[autopilotKey]
}
