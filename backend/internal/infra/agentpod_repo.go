package infra

import (
	"context"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"gorm.io/gorm"
)

var _ agentpod.PodRepository = (*podRepo)(nil)

type podRepo struct{ db *gorm.DB }

// NewPodRepository creates a new PodRepository backed by GORM.
func NewPodRepository(db *gorm.DB) agentpod.PodRepository {
	return &podRepo{db: db}
}

func (r *podRepo) Create(ctx context.Context, pod *agentpod.Pod) error {
	err := r.db.WithContext(ctx).Create(pod).Error
	if err != nil && isUniqueConstraintViolation(err, "idx_pods_source_pod_key_active_unique") {
		return agentpod.ErrSandboxAlreadyResumed
	}
	return err
}

func (r *podRepo) GetByKey(ctx context.Context, podKey string) (*agentpod.Pod, error) {
	var pod agentpod.Pod
	err := r.db.WithContext(ctx).
		Preload("Runner").Preload("AgentType").Preload("Repository").
		Where("pod_key = ?", podKey).First(&pod).Error
	if err != nil {
		return nil, err
	}
	return &pod, nil
}

func (r *podRepo) GetByID(ctx context.Context, podID int64) (*agentpod.Pod, error) {
	var pod agentpod.Pod
	err := r.db.WithContext(ctx).
		Preload("Runner").Preload("AgentType").Preload("Repository").
		First(&pod, podID).Error
	if err != nil {
		return nil, err
	}
	return &pod, nil
}

func (r *podRepo) GetOrgAndCreator(ctx context.Context, podKey string) (int64, int64, error) {
	var pod agentpod.Pod
	err := r.db.WithContext(ctx).
		Select("organization_id", "created_by_id").
		Where("pod_key = ?", podKey).First(&pod).Error
	if err != nil {
		return 0, 0, err
	}
	return pod.OrganizationID, pod.CreatedByID, nil
}

func (r *podRepo) GetTicketByID(ctx context.Context, ticketID int64) (string, string, error) {
	var t ticket.Ticket
	if err := r.db.WithContext(ctx).First(&t, ticketID).Error; err != nil {
		return "", "", err
	}
	return t.Slug, t.Title, nil
}

func (r *podRepo) ListByOrg(ctx context.Context, orgID int64, statuses []string, createdByID int64, limit, offset int) ([]*agentpod.Pod, int64, error) {
	query := r.db.WithContext(ctx).Model(&agentpod.Pod{}).Where("organization_id = ?", orgID)
	switch len(statuses) {
	case 0:
	case 1:
		query = query.Where("status = ?", statuses[0])
	default:
		query = query.Where("status IN ?", statuses)
	}
	if createdByID > 0 {
		query = query.Where("created_by_id = ?", createdByID)
	}

	var total int64
	query.Count(&total)

	var pods []*agentpod.Pod
	err := query.
		Preload("Runner").Preload("AgentType").Preload("Ticket").Preload("CreatedBy").Preload("Repository").
		Order("created_at DESC").Limit(limit).Offset(offset).Find(&pods).Error
	if err != nil {
		return nil, 0, err
	}
	return pods, total, nil
}

func (r *podRepo) ListByTicket(ctx context.Context, ticketID int64) ([]*agentpod.Pod, error) {
	var pods []*agentpod.Pod
	err := r.db.WithContext(ctx).
		Preload("Runner").Preload("AgentType").Preload("Repository").
		Where("ticket_id = ?", ticketID).Order("created_at DESC").Find(&pods).Error
	return pods, err
}

func (r *podRepo) ListByRunner(ctx context.Context, runnerID int64, status string) ([]*agentpod.Pod, error) {
	query := r.db.WithContext(ctx).Where("runner_id = ?", runnerID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var pods []*agentpod.Pod
	err := query.Preload("Runner").Preload("AgentType").Preload("Repository").
		Order("created_at DESC").Find(&pods).Error
	return pods, err
}

func (r *podRepo) ListByRunnerPaginated(ctx context.Context, runnerID int64, status string, limit, offset int) ([]*agentpod.Pod, int64, error) {
	query := r.db.WithContext(ctx).Model(&agentpod.Pod{}).Where("runner_id = ?", runnerID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var pods []*agentpod.Pod
	err := query.
		Preload("AgentType").Preload("Ticket").Preload("CreatedBy").Preload("Repository").
		Order("created_at DESC").Limit(limit).Offset(offset).Find(&pods).Error
	if err != nil {
		return nil, 0, err
	}
	return pods, total, nil
}

func (r *podRepo) ListActive(ctx context.Context, runnerID int64) ([]*agentpod.Pod, error) {
	var pods []*agentpod.Pod
	err := r.db.WithContext(ctx).
		Where("runner_id = ? AND status IN ?", runnerID, []string{
			agentpod.StatusInitializing, agentpod.StatusRunning,
			agentpod.StatusPaused, agentpod.StatusDisconnected,
		}).Find(&pods).Error
	return pods, err
}

func (r *podRepo) GetActivePodBySourcePodKey(ctx context.Context, sourcePodKey string) (*agentpod.Pod, error) {
	var pod agentpod.Pod
	err := r.db.WithContext(ctx).
		Where("source_pod_key = ?", sourcePodKey).
		Where("status IN ?", []string{
			agentpod.StatusInitializing, agentpod.StatusRunning,
			agentpod.StatusPaused, agentpod.StatusDisconnected,
		}).First(&pod).Error
	if err != nil {
		return nil, err
	}
	return &pod, nil
}

func (r *podRepo) FindByBranchAndRepo(ctx context.Context, orgID, repoID int64, branchName string) (*agentpod.Pod, error) {
	var pod agentpod.Pod
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND repository_id = ? AND branch_name = ?", orgID, repoID, branchName).
		Order("created_at DESC").First(&pod).Error
	if err != nil {
		return nil, err
	}
	return &pod, nil
}

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
	result := r.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("status IN ? AND last_activity < ?", []string{agentpod.StatusDisconnected}, threshold).
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
