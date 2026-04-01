package infra

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/loop"
)

// ComputeLoopStats computes run statistics from Pod status (SSOT).
func (r *loopRunRepo) ComputeLoopStats(ctx context.Context, loopID int64) (total, successful, failed int, err error) {
	// Phase 1: Aggregate finished runs via SQL
	type finishedStats struct {
		Total      int `gorm:"column:total"`
		Successful int `gorm:"column:successful"`
		Failed     int `gorm:"column:failed"`
	}
	var fs finishedStats
	err = r.db.WithContext(ctx).
		Table("loop_runs").
		Select(`
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = ?) as successful,
			COUNT(*) FILTER (WHERE status IN (?, ?, ?)) as failed
		`, loop.RunStatusCompleted, loop.RunStatusFailed, loop.RunStatusTimeout, loop.RunStatusCancelled).
		Where("loop_id = ? AND finished_at IS NOT NULL", loopID).
		Scan(&fs).Error
	if err != nil {
		return
	}
	total = fs.Total
	successful = fs.Successful
	failed = fs.Failed

	// Phase 2: Resolve active runs via Go-side SSOT
	total, successful, failed, err = r.resolveActiveRunStats(ctx, loopID, total, successful, failed)
	return
}

// resolveActiveRunStats resolves active run stats from Pod/autopilot status.
func (r *loopRunRepo) resolveActiveRunStats(ctx context.Context, loopID int64, total, successful, failed int) (int, int, int, error) {
	type activeRunRow struct {
		Status         string  `gorm:"column:status"`
		PodKey         *string `gorm:"column:pod_key"`
		PodStatus      *string `gorm:"column:pod_status"`
		AutopilotPhase *string `gorm:"column:autopilot_phase"`
	}
	var activeRows []activeRunRow
	err := r.db.WithContext(ctx).
		Table("loop_runs lr").
		Select("lr.status, lr.pod_key, p.status as pod_status, ac.phase as autopilot_phase").
		Joins("LEFT JOIN pods p ON p.pod_key = lr.pod_key").
		Joins("LEFT JOIN autopilot_controllers ac ON ac.autopilot_controller_key = lr.autopilot_controller_key").
		Where("lr.loop_id = ? AND lr.finished_at IS NULL", loopID).
		Find(&activeRows).Error
	if err != nil {
		return total, successful, failed, err
	}

	for _, row := range activeRows {
		total++

		var effectiveStatus string
		if row.PodKey == nil {
			effectiveStatus = row.Status
		} else {
			podStatus := ""
			if row.PodStatus != nil {
				podStatus = *row.PodStatus
			}
			autopilotPhase := ""
			if row.AutopilotPhase != nil {
				autopilotPhase = *row.AutopilotPhase
			}
			effectiveStatus = deriveLoopRunStatus(podStatus, autopilotPhase)
		}

		switch effectiveStatus {
		case loop.RunStatusCompleted:
			successful++
		case loop.RunStatusFailed, loop.RunStatusTimeout, loop.RunStatusCancelled:
			failed++
		}
	}
	return total, successful, failed, nil
}

// BatchGetPodStatuses returns Pod status info for a batch of pod keys.
func (r *loopRunRepo) BatchGetPodStatuses(ctx context.Context, podKeys []string) ([]loop.PodStatusInfo, error) {
	if len(podKeys) == 0 {
		return nil, nil
	}

	var results []loop.PodStatusInfo
	err := r.db.WithContext(ctx).
		Table("pods").
		Select("pod_key, status, finished_at").
		Where("pod_key IN ?", podKeys).
		Find(&results).Error
	return results, err
}

// BatchGetAutopilotPhases returns autopilot phases for a batch of keys.
func (r *loopRunRepo) BatchGetAutopilotPhases(ctx context.Context, autopilotKeys []string) (map[string]string, error) {
	if len(autopilotKeys) == 0 {
		return nil, nil
	}

	type row struct {
		Key   string `gorm:"column:autopilot_controller_key"`
		Phase string `gorm:"column:phase"`
	}
	var rows []row
	if err := r.db.WithContext(ctx).
		Table("autopilot_controllers").
		Select("autopilot_controller_key, phase").
		Where("autopilot_controller_key IN ?", autopilotKeys).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make(map[string]string, len(rows))
	for _, r := range rows {
		result[r.Key] = r.Phase
	}
	return result, nil
}

// CountActiveRunsByLoopIDs batch-counts active runs for multiple loops using Pod status (SSOT).
func (r *loopRunRepo) CountActiveRunsByLoopIDs(ctx context.Context, loopIDs []int64) (map[int64]int64, error) {
	if len(loopIDs) == 0 {
		return nil, nil
	}

	type countRow struct {
		LoopID int64 `gorm:"column:loop_id"`
		Count  int64 `gorm:"column:count"`
	}
	var rows []countRow
	err := r.db.WithContext(ctx).
		Table("loop_runs").
		Select("loop_runs.loop_id, COUNT(*) as count").
		Joins("LEFT JOIN pods ON pods.pod_key = loop_runs.pod_key").
		Where("loop_runs.loop_id IN ?", loopIDs).
		Where(
			"(loop_runs.pod_key IS NULL AND loop_runs.status = ?) OR "+
				"(loop_runs.pod_key IS NOT NULL AND pods.status IN ?)",
			loop.RunStatusPending,
			agentpod.ActiveStatuses(),
		).
		Group("loop_runs.loop_id").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int64]int64, len(rows))
	for _, row := range rows {
		result[row.LoopID] = row.Count
	}
	return result, nil
}

// GetAvgDuration returns the average duration in seconds for completed runs of a loop.
func (r *loopRunRepo) GetAvgDuration(ctx context.Context, loopID int64) (*float64, error) {
	var avg *float64
	err := r.db.WithContext(ctx).
		Table("loop_runs").
		Where("loop_id = ? AND duration_sec IS NOT NULL AND finished_at IS NOT NULL", loopID).
		Select("AVG(duration_sec)").
		Scan(&avg).Error
	return avg, err
}

// deriveLoopRunStatus maps Pod/Autopilot state to Loop Run status.
// This is a local helper mirroring the canonical logic in service/loop.DeriveRunStatus,
// used here to avoid a circular infra → service import.
func deriveLoopRunStatus(podStatus string, autopilotPhase string) string {
	if autopilotPhase != "" {
		switch autopilotPhase {
		case agentpod.AutopilotPhaseCompleted, agentpod.AutopilotPhaseMaxIterations:
			return loop.RunStatusCompleted
		case agentpod.AutopilotPhaseFailed:
			return loop.RunStatusFailed
		case agentpod.AutopilotPhaseStopped:
			return loop.RunStatusCancelled
		default:
			if isPodDone(podStatus) {
				return podToRunStatus(podStatus)
			}
			return loop.RunStatusRunning
		}
	}
	if isPodDone(podStatus) {
		return podToRunStatus(podStatus)
	}
	return loop.RunStatusRunning
}

func isPodDone(podStatus string) bool {
	return podStatus == agentpod.StatusCompleted ||
		podStatus == agentpod.StatusTerminated ||
		podStatus == agentpod.StatusError
}

func podToRunStatus(podStatus string) string {
	switch podStatus {
	case agentpod.StatusCompleted:
		return loop.RunStatusCompleted
	case agentpod.StatusTerminated:
		return loop.RunStatusCancelled
	case agentpod.StatusError:
		return loop.RunStatusFailed
	default:
		return loop.RunStatusFailed
	}
}
