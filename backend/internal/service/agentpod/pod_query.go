package agentpod

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

// GetPod returns a pod by key
func (s *PodService) GetPod(ctx context.Context, podKey string) (*agentpod.Pod, error) {
	var pod agentpod.Pod
	if err := s.db.WithContext(ctx).
		Preload("Runner").
		Preload("AgentType").
		Where("pod_key = ?", podKey).
		First(&pod).Error; err != nil {
		return nil, ErrPodNotFound
	}
	return &pod, nil
}

// GetPodByID returns a pod by ID
func (s *PodService) GetPodByID(ctx context.Context, podID int64) (*agentpod.Pod, error) {
	var pod agentpod.Pod
	if err := s.db.WithContext(ctx).
		Preload("Runner").
		Preload("AgentType").
		First(&pod, podID).Error; err != nil {
		return nil, ErrPodNotFound
	}
	return &pod, nil
}

// GetPodByKey returns a pod by key (implements middleware.PodService)
func (s *PodService) GetPodByKey(ctx context.Context, podKey string) (*agentpod.Pod, error) {
	return s.GetPod(ctx, podKey)
}

// GetPodInfo returns pod info for binding policy evaluation
func (s *PodService) GetPodInfo(ctx context.Context, podKey string) (map[string]interface{}, error) {
	pod, err := s.GetPod(ctx, podKey)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"user_id":         pod.CreatedByID,
		"organization_id": pod.OrganizationID,
		"ticket_id":       pod.TicketID,
		"status":          pod.Status,
	}, nil
}

// GetPodOrganizationAndCreator returns the organization ID and creator ID for a pod
func (s *PodService) GetPodOrganizationAndCreator(ctx context.Context, podKey string) (orgID, creatorID int64, err error) {
	var pod agentpod.Pod
	if err := s.db.WithContext(ctx).
		Select("organization_id", "created_by_id").
		Where("pod_key = ?", podKey).
		First(&pod).Error; err != nil {
		return 0, 0, ErrPodNotFound
	}
	return pod.OrganizationID, pod.CreatedByID, nil
}

// GetPodsByTicket returns pods for a ticket
func (s *PodService) GetPodsByTicket(ctx context.Context, ticketID int64) ([]*agentpod.Pod, error) {
	var pods []*agentpod.Pod
	if err := s.db.WithContext(ctx).
		Preload("Runner").
		Preload("AgentType").
		Where("ticket_id = ?", ticketID).
		Order("created_at DESC").
		Find(&pods).Error; err != nil {
		return nil, err
	}
	return pods, nil
}

// ListPods returns pods for an organization
func (s *PodService) ListPods(ctx context.Context, orgID int64, status string, limit, offset int) ([]*agentpod.Pod, int64, error) {
	query := s.db.WithContext(ctx).Model(&agentpod.Pod{}).Where("organization_id = ?", orgID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var pods []*agentpod.Pod
	if err := query.
		Preload("Runner").
		Preload("AgentType").
		Preload("Ticket").
		Preload("CreatedBy").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&pods).Error; err != nil {
		return nil, 0, err
	}

	return pods, total, nil
}

// ListActivePods returns active pods for a runner
func (s *PodService) ListActivePods(ctx context.Context, runnerID int64) ([]*agentpod.Pod, error) {
	var pods []*agentpod.Pod
	if err := s.db.WithContext(ctx).
		Where("runner_id = ? AND status IN ?", runnerID, []string{
			agentpod.StatusInitializing,
			agentpod.StatusRunning,
			agentpod.StatusPaused,
			agentpod.StatusDisconnected,
		}).
		Find(&pods).Error; err != nil {
		return nil, err
	}
	return pods, nil
}

// ListByRunner returns pods for a runner with optional status filter
func (s *PodService) ListByRunner(ctx context.Context, runnerID int64, status string) ([]*agentpod.Pod, error) {
	query := s.db.WithContext(ctx).Where("runner_id = ?", runnerID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var pods []*agentpod.Pod
	if err := query.
		Preload("Runner").
		Preload("AgentType").
		Order("created_at DESC").
		Find(&pods).Error; err != nil {
		return nil, err
	}
	return pods, nil
}

// ListByTicket returns pods for a ticket
func (s *PodService) ListByTicket(ctx context.Context, ticketID int64) ([]*agentpod.Pod, error) {
	var pods []*agentpod.Pod
	if err := s.db.WithContext(ctx).
		Preload("Runner").
		Preload("AgentType").
		Where("ticket_id = ?", ticketID).
		Order("created_at DESC").
		Find(&pods).Error; err != nil {
		return nil, err
	}
	return pods, nil
}
