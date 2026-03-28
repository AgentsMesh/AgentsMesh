package agentpod

import (
	"context"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

// GetPod returns a pod by key
func (s *PodService) GetPod(ctx context.Context, podKey string) (*agentpod.Pod, error) {
	pod, err := s.repo.GetByKey(ctx, podKey)
	if err != nil {
		return nil, ErrPodNotFound
	}
	// Best-effort: enrich with loop info for display
	_ = s.repo.EnrichWithLoopInfo(ctx, []*agentpod.Pod{pod})
	return pod, nil
}

// GetPodByID returns a pod by ID
func (s *PodService) GetPodByID(ctx context.Context, podID int64) (*agentpod.Pod, error) {
	pod, err := s.repo.GetByID(ctx, podID)
	if err != nil {
		return nil, ErrPodNotFound
	}
	return pod, nil
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
	orgID, creatorID, err = s.repo.GetOrgAndCreator(ctx, podKey)
	if err != nil {
		return 0, 0, ErrPodNotFound
	}
	return orgID, creatorID, nil
}

// GetPodsByTicket returns pods for a ticket
func (s *PodService) GetPodsByTicket(ctx context.Context, ticketID int64) ([]*agentpod.Pod, error) {
	return s.repo.ListByTicket(ctx, ticketID)
}

// ListPods returns pods for an organization
func (s *PodService) ListPods(ctx context.Context, orgID int64, statuses []string, createdByID int64, limit, offset int) ([]*agentpod.Pod, int64, error) {
	pods, total, err := s.repo.ListByOrg(ctx, orgID, statuses, createdByID, limit, offset)
	if err != nil {
		slog.Error("failed to list pods", "org_id", orgID, "error", err)
		return nil, 0, err
	}
	// Best-effort: enrich with loop info for display
	_ = s.repo.EnrichWithLoopInfo(ctx, pods)
	return pods, total, nil
}

// ListActivePods returns active pods for a runner
func (s *PodService) ListActivePods(ctx context.Context, runnerID int64) ([]*agentpod.Pod, error) {
	return s.repo.ListActive(ctx, runnerID)
}

// ListByRunner returns pods for a runner with optional status filter
func (s *PodService) ListByRunner(ctx context.Context, runnerID int64, status string) ([]*agentpod.Pod, error) {
	return s.repo.ListByRunner(ctx, runnerID, status)
}

// ListByTicket returns pods for a ticket
func (s *PodService) ListByTicket(ctx context.Context, ticketID int64) ([]*agentpod.Pod, error) {
	return s.repo.ListByTicket(ctx, ticketID)
}

// ListPodsByRunner returns pods for a runner with pagination and optional status filter
func (s *PodService) ListPodsByRunner(ctx context.Context, runnerID int64, status string, limit, offset int) ([]*agentpod.Pod, int64, error) {
	return s.repo.ListByRunnerPaginated(ctx, runnerID, status, limit, offset)
}

// GetActivePodBySourcePodKey returns an active pod that was resumed from the given source pod key
func (s *PodService) GetActivePodBySourcePodKey(ctx context.Context, sourcePodKey string) (*agentpod.Pod, error) {
	return s.repo.GetActivePodBySourcePodKey(ctx, sourcePodKey)
}

// FindByBranchAndRepo finds a Pod by branch name and repository ID
func (s *PodService) FindByBranchAndRepo(ctx context.Context, orgID, repoID int64, branchName string) (*agentpod.Pod, error) {
	return s.repo.FindByBranchAndRepo(ctx, orgID, repoID, branchName)
}
