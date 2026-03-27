package infra

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/loop"
)

// CountActiveRuns counts runs that are actually active, using Pod status as SSOT.
func (r *loopRunRepo) CountActiveRuns(ctx context.Context, loopID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("loop_runs").
		Joins("LEFT JOIN pods ON pods.pod_key = loop_runs.pod_key").
		Where("loop_runs.loop_id = ?", loopID).
		Where(
			"(loop_runs.pod_key IS NULL AND loop_runs.status = ?) OR "+
				"(loop_runs.pod_key IS NOT NULL AND pods.status IN ?)",
			loop.RunStatusPending,
			agentpod.ActiveStatuses(),
		).
		Count(&count).Error
	return count, err
}

// GetActiveRunByPodKey finds an unfinished run by its pod key.
func (r *loopRunRepo) GetActiveRunByPodKey(ctx context.Context, podKey string) (*loop.LoopRun, error) {
	var run loop.LoopRun
	err := r.db.WithContext(ctx).
		Where("pod_key = ? AND finished_at IS NULL", podKey).
		First(&run).Error
	if err != nil {
		if isNotFound(err) {
			return nil, loop.ErrNotFound
		}
		return nil, err
	}
	return &run, nil
}

// GetTimedOutRuns returns running runs that have exceeded their timeout.
func (r *loopRunRepo) GetTimedOutRuns(ctx context.Context, orgIDs []int64) ([]*loop.LoopRun, error) {
	var runs []*loop.LoopRun
	timedOutEligible := []string{agentpod.StatusInitializing, agentpod.StatusRunning, agentpod.StatusPaused}
	query := r.db.WithContext(ctx).
		Table("loop_runs").
		Joins("JOIN loops ON loops.id = loop_runs.loop_id").
		Joins("LEFT JOIN pods ON pods.pod_key = loop_runs.pod_key").
		Where("loop_runs.pod_key IS NOT NULL").
		Where("loop_runs.finished_at IS NULL").
		Where("pods.status IN ?", timedOutEligible).
		Where("loop_runs.started_at IS NOT NULL AND loop_runs.started_at < NOW() - (loops.timeout_minutes || ' minutes')::INTERVAL")
	if len(orgIDs) > 0 {
		query = query.Where("loop_runs.organization_id IN ?", orgIDs)
	}
	err := query.Find(&runs).Error
	return runs, err
}

// GetLatestPodKey returns the pod_key from the most recent run that has one.
func (r *loopRunRepo) GetLatestPodKey(ctx context.Context, loopID int64) *string {
	type result struct {
		PodKey string `gorm:"column:pod_key"`
	}
	var res result
	err := r.db.WithContext(ctx).
		Table("loop_runs").
		Select("loop_runs.pod_key").
		Where("loop_runs.loop_id = ? AND loop_runs.pod_key IS NOT NULL", loopID).
		Order("loop_runs.id DESC").
		Limit(1).
		Scan(&res).Error

	if err != nil || res.PodKey == "" {
		return nil
	}
	return &res.PodKey
}

// GetOrphanPendingRuns returns pending runs with no pod_key that are stuck for > 5 minutes.
func (r *loopRunRepo) GetOrphanPendingRuns(ctx context.Context, orgIDs []int64) ([]*loop.LoopRun, error) {
	var runs []*loop.LoopRun
	query := r.db.WithContext(ctx).
		Where("pod_key IS NULL").
		Where("status = ?", loop.RunStatusPending).
		Where("finished_at IS NULL").
		Where("created_at < NOW() - INTERVAL '5 minutes'")
	if len(orgIDs) > 0 {
		query = query.Where("organization_id IN ?", orgIDs)
	}
	err := query.Find(&runs).Error
	return runs, err
}

// GetIdleLoopPods returns active loop runs whose Pods have been idle longer than idle_timeout_sec.
func (r *loopRunRepo) GetIdleLoopPods(ctx context.Context, orgIDs []int64) ([]*loop.LoopRun, error) {
	var runs []*loop.LoopRun
	query := r.db.WithContext(ctx).
		Table("loop_runs").
		Joins("JOIN loops ON loops.id = loop_runs.loop_id").
		Joins("JOIN pods ON pods.pod_key = loop_runs.pod_key").
		Where("loop_runs.finished_at IS NULL").
		Where("pods.status = ?", agentpod.StatusRunning).
		Where("pods.agent_status = ?", agentpod.AgentStatusWaiting).
		Where("pods.agent_waiting_since IS NOT NULL").
		Where("loops.idle_timeout_sec > 0").
		Where("pods.agent_waiting_since < NOW() - (loops.idle_timeout_sec || ' seconds')::INTERVAL")
	if len(orgIDs) > 0 {
		query = query.Where("loop_runs.organization_id IN ?", orgIDs)
	}
	err := query.Find(&runs).Error
	return runs, err
}
