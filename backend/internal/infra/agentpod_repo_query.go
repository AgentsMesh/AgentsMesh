package infra

import (
	"context"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"gorm.io/gorm"
)

func (r *podRepo) UpdateByKey(ctx context.Context, podKey string, updates map[string]interface{}) (int64, error) {
	result := r.db.WithContext(ctx).Model(&agentpod.Pod{}).Where("pod_key = ?", podKey).Updates(updates)
	return result.RowsAffected, result.Error
}

func (r *podRepo) UpdateByKeyAndStatus(ctx context.Context, podKey, status string, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ? AND status = ?", podKey, status).
		Updates(updates).Error
}

func (r *podRepo) UpdateAgentStatus(ctx context.Context, podKey string, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ?", podKey).Updates(updates).Error
}

func (r *podRepo) UpdateField(ctx context.Context, podKey, field string, value interface{}) error {
	return r.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ?", podKey).Update(field, value).Error
}

func (r *podRepo) DecrementRunnerPods(ctx context.Context, runnerID int64) error {
	return r.db.WithContext(ctx).
		Exec("UPDATE runners SET current_pods = GREATEST(current_pods - 1, 0) WHERE id = ?", runnerID).Error
}

func (r *podRepo) ListActiveByRunner(ctx context.Context, runnerID int64) ([]*agentpod.Pod, error) {
	var pods []*agentpod.Pod
	err := r.db.WithContext(ctx).
		Where("runner_id = ? AND status IN ?", runnerID, agentpod.ActiveStatuses()).
		Find(&pods).Error
	return pods, err
}

func (r *podRepo) ListInitializingByRunner(ctx context.Context, runnerID int64) ([]*agentpod.Pod, error) {
	var pods []*agentpod.Pod
	err := r.db.WithContext(ctx).
		Where("runner_id = ? AND status = ?", runnerID, agentpod.StatusInitializing).
		Find(&pods).Error
	return pods, err
}

func (r *podRepo) MarkOrphaned(ctx context.Context, pod *agentpod.Pod, finishedAt time.Time) error {
	return r.db.WithContext(ctx).Model(pod).Updates(map[string]interface{}{
		"status":      agentpod.StatusOrphaned,
		"finished_at": finishedAt,
	}).Error
}

func (r *podRepo) MarkStaleAsDisconnected(ctx context.Context, threshold time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("status IN ? AND (last_activity < ? OR last_activity IS NULL)",
			[]string{agentpod.StatusInitializing, agentpod.StatusRunning}, threshold).
		Update("status", agentpod.StatusDisconnected)
	return result.RowsAffected, result.Error
}

func (r *podRepo) CleanupStale(ctx context.Context, threshold time.Time) (int64, error) {
	now := time.Now()
	// Clean up both disconnected and orphaned pods that have been idle too long.
	// Orphaned pods whose runner recovered but did not report them back are stuck
	// in "orphaned" forever without this cleanup.
	result := r.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("status IN ? AND last_activity < ?",
			[]string{agentpod.StatusDisconnected, agentpod.StatusOrphaned}, threshold).
		Updates(map[string]interface{}{
			"status":      agentpod.StatusTerminated,
			"finished_at": now,
		})
	return result.RowsAffected, result.Error
}

func (r *podRepo) UpdateByKeyAndStatusCounted(ctx context.Context, podKey, status string, updates map[string]interface{}) (int64, error) {
	result := r.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ? AND status = ?", podKey, status).
		Updates(updates)
	return result.RowsAffected, result.Error
}

func (r *podRepo) UpdateTerminatedWithFallbackError(ctx context.Context, podKey string, updates map[string]interface{}, fallbackErrorCode string) error {
	updates["error_code"] = gorm.Expr("COALESCE(NULLIF(error_code, ''), ?)", fallbackErrorCode)
	return r.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ?", podKey).
		Updates(updates).Error
}

// UpdateTerminatedIfActive updates a terminated pod with error info, but only if
// the pod is still in an active state (initializing/running/paused/disconnected).
// Returns rows affected so the caller can detect if the pod was already in a terminal state.
func (r *podRepo) UpdateTerminatedIfActive(ctx context.Context, podKey string, updates map[string]interface{}, fallbackErrorCode string) (int64, error) {
	updates["error_code"] = gorm.Expr("COALESCE(NULLIF(error_code, ''), ?)", fallbackErrorCode)
	result := r.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ? AND status IN ?", podKey, agentpod.ActiveStatuses()).
		Updates(updates)
	return result.RowsAffected, result.Error
}

// UpdateByKeyAndActiveStatus updates a pod only if it's in an active state.
// Returns rows affected so the caller can detect if the pod was already in a terminal state.
func (r *podRepo) UpdateByKeyAndActiveStatus(ctx context.Context, podKey string, updates map[string]interface{}) (int64, error) {
	result := r.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ? AND status IN ?", podKey, agentpod.ActiveStatuses()).
		Updates(updates)
	return result.RowsAffected, result.Error
}

func (r *podRepo) GetByKeyAndRunner(ctx context.Context, podKey string, runnerID int64) (*agentpod.Pod, error) {
	var pod agentpod.Pod
	err := r.db.WithContext(ctx).
		Where("pod_key = ? AND runner_id = ?", podKey, runnerID).
		First(&pod).Error
	if err != nil {
		return nil, err
	}
	return &pod, nil
}

func (r *podRepo) CountActiveByKeys(ctx context.Context, podKeys []string) (int, error) {
	if len(podKeys) == 0 {
		return 0, nil
	}
	var count int64
	err := r.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key IN ? AND status IN ?", podKeys,
			[]string{agentpod.StatusRunning, agentpod.StatusInitializing}).
		Count(&count).Error
	return int(count), err
}

func (r *podRepo) EnrichWithLoopInfo(ctx context.Context, pods []*agentpod.Pod) error {
	if len(pods) == 0 {
		return nil
	}

	podKeys := make([]string, 0, len(pods))
	for _, p := range pods {
		podKeys = append(podKeys, p.PodKey)
	}

	type loopRow struct {
		PodKey   string `gorm:"column:pod_key"`
		LoopID   int64  `gorm:"column:loop_id"`
		LoopName string `gorm:"column:loop_name"`
		LoopSlug string `gorm:"column:loop_slug"`
	}

	var rows []loopRow
	err := r.db.WithContext(ctx).
		Table("loop_runs").
		Select("loop_runs.pod_key, loops.id AS loop_id, loops.name AS loop_name, loops.slug AS loop_slug").
		Joins("JOIN loops ON loops.id = loop_runs.loop_id").
		Where("loop_runs.pod_key IN ?", podKeys).
		Find(&rows).Error
	if err != nil {
		return err
	}

	loopByKey := make(map[string]*agentpod.PodLoopInfo, len(rows))
	for _, row := range rows {
		loopByKey[row.PodKey] = &agentpod.PodLoopInfo{
			ID:   row.LoopID,
			Name: row.LoopName,
			Slug: row.LoopSlug,
		}
	}

	for _, p := range pods {
		if info, ok := loopByKey[p.PodKey]; ok {
			p.Loop = info
		}
	}
	return nil
}

// isUniqueConstraintViolation checks if the error is a PostgreSQL unique constraint violation.
func isUniqueConstraintViolation(err error, constraintName string) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "duplicate key value") && strings.Contains(errStr, constraintName)
}
