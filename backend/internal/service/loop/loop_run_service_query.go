package loop

import (
	"context"
	"log/slog"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
)

// GetTimedOutRuns returns running runs that have exceeded their timeout.
func (s *LoopRunService) GetTimedOutRuns(ctx context.Context, orgIDs []int64) ([]*loopDomain.LoopRun, error) {
	return s.repo.GetTimedOutRuns(ctx, orgIDs)
}

// GetOrphanPendingRuns returns pending runs with no pod_key stuck for > 5 minutes.
func (s *LoopRunService) GetOrphanPendingRuns(ctx context.Context, orgIDs []int64) ([]*loopDomain.LoopRun, error) {
	return s.repo.GetOrphanPendingRuns(ctx, orgIDs)
}

// GetIdleLoopPods returns active loop runs whose Pods have been idle longer than idle_timeout_sec.
func (s *LoopRunService) GetIdleLoopPods(ctx context.Context, orgIDs []int64) ([]*loopDomain.LoopRun, error) {
	return s.repo.GetIdleLoopPods(ctx, orgIDs)
}

// ComputeLoopStats computes run statistics from Pod status (SSOT).
func (s *LoopRunService) ComputeLoopStats(ctx context.Context, loopID int64) (total int, successful int, failed int, err error) {
	return s.repo.ComputeLoopStats(ctx, loopID)
}

// GetLatestPodKey returns the pod_key from the most recent run that has one.
func (s *LoopRunService) GetLatestPodKey(ctx context.Context, loopID int64) *string {
	return s.repo.GetLatestPodKey(ctx, loopID)
}

// CountActiveRunsByLoopIDs batch-counts active runs for multiple loops.
func (s *LoopRunService) CountActiveRunsByLoopIDs(ctx context.Context, loopIDs []int64) (map[int64]int64, error) {
	return s.repo.CountActiveRunsByLoopIDs(ctx, loopIDs)
}

// GetAvgDuration returns the average duration in seconds for a loop.
func (s *LoopRunService) GetAvgDuration(ctx context.Context, loopID int64) (*float64, error) {
	return s.repo.GetAvgDuration(ctx, loopID)
}

// DeleteOldFinishedRuns deletes finished runs exceeding the retention limit.
func (s *LoopRunService) DeleteOldFinishedRuns(ctx context.Context, loopID int64, keep int) (int64, error) {
	deleted, err := s.repo.DeleteOldFinishedRuns(ctx, loopID, keep)
	if err != nil {
		slog.Error("failed to delete old finished runs", "loop_id", loopID, "keep", keep, "error", err)
		return 0, err
	}
	if deleted > 0 {
		slog.Info("old finished runs deleted", "loop_id", loopID, "deleted", deleted, "keep", keep)
	}
	return deleted, nil
}

// GetAutopilotPhase returns the autopilot phase for a given controller key.
func (s *LoopRunService) GetAutopilotPhase(ctx context.Context, autopilotKey string) string {
	phases, err := s.repo.BatchGetAutopilotPhases(ctx, []string{autopilotKey})
	if err != nil || phases == nil {
		return ""
	}
	return phases[autopilotKey]
}
