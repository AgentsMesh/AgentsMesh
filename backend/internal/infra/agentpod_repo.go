package infra

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/grant"
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
		Preload("Runner").Preload("Agent").Preload("Repository").
		Where("pod_key = ?", podKey).First(&pod).Error
	if err != nil {
		return nil, err
	}
	return &pod, nil
}

func (r *podRepo) GetByID(ctx context.Context, podID int64) (*agentpod.Pod, error) {
	var pod agentpod.Pod
	err := r.db.WithContext(ctx).
		Preload("Runner").Preload("Agent").Preload("Repository").
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

func (r *podRepo) ListByOrg(ctx context.Context, orgID int64, q agentpod.PodListQuery) ([]*agentpod.Pod, int64, error) {
	query := r.db.WithContext(ctx).Model(&agentpod.Pod{}).Where("organization_id = ?", orgID)
	switch len(q.Statuses) {
	case 0:
	case 1:
		query = query.Where("status = ?", q.Statuses[0])
	default:
		query = query.Where("status IN ?", q.Statuses)
	}
	if q.CreatedByID > 0 && q.GrantedUserID > 0 {
		query = query.Where("(created_by_id = ? OR pod_key IN (SELECT resource_id FROM resource_grants WHERE resource_type = ? AND user_id = ? AND organization_id = ?))",
			q.CreatedByID, grant.TypePod, q.GrantedUserID, orgID)
	} else if q.CreatedByID > 0 {
		query = query.Where("created_by_id = ?", q.CreatedByID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var pods []*agentpod.Pod
	err := query.
		Preload("Runner").Preload("Agent").Preload("Ticket").Preload("CreatedBy").Preload("Repository").
		Order("created_at DESC").Limit(q.Limit).Offset(q.Offset).Find(&pods).Error
	if err != nil {
		return nil, 0, err
	}
	return pods, total, nil
}

func (r *podRepo) ListByTicket(ctx context.Context, ticketID int64) ([]*agentpod.Pod, error) {
	var pods []*agentpod.Pod
	err := r.db.WithContext(ctx).
		Preload("Runner").Preload("Agent").Preload("Repository").
		Where("ticket_id = ?", ticketID).Order("created_at DESC").Find(&pods).Error
	return pods, err
}

func (r *podRepo) ListByRunner(ctx context.Context, runnerID int64, status string) ([]*agentpod.Pod, error) {
	query := r.db.WithContext(ctx).Where("runner_id = ?", runnerID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var pods []*agentpod.Pod
	err := query.Preload("Runner").Preload("Agent").Preload("Repository").
		Order("created_at DESC").Find(&pods).Error
	return pods, err
}

func (r *podRepo) ListByRunnerPaginated(ctx context.Context, runnerID int64, q agentpod.PodListQuery) ([]*agentpod.Pod, int64, error) {
	query := r.db.WithContext(ctx).Model(&agentpod.Pod{}).Where("runner_id = ?", runnerID)
	switch len(q.Statuses) {
	case 0:
	case 1:
		query = query.Where("status = ?", q.Statuses[0])
	default:
		query = query.Where("status IN ?", q.Statuses)
	}
	if q.CreatedByID > 0 && q.GrantedUserID > 0 {
		query = query.Where("(created_by_id = ? OR pod_key IN (SELECT resource_id FROM resource_grants WHERE resource_type = ? AND user_id = ?))",
			q.CreatedByID, grant.TypePod, q.GrantedUserID)
	} else if q.CreatedByID > 0 {
		query = query.Where("created_by_id = ?", q.CreatedByID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var pods []*agentpod.Pod
	err := query.
		Preload("Agent").Preload("Ticket").Preload("CreatedBy").Preload("Repository").
		Order("created_at DESC").Limit(q.Limit).Offset(q.Offset).Find(&pods).Error
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
