package agentpod

import (
	"context"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

func (s *PodService) GetPod(ctx context.Context, podKey string) (*agentpod.Pod, error) {
	pod, err := s.repo.GetByKey(ctx, podKey)
	if err != nil {
		return nil, ErrPodNotFound
	}
	_ = s.repo.EnrichWithLoopInfo(ctx, []*agentpod.Pod{pod})
	return pod, nil
}

func (s *PodService) GetPodByID(ctx context.Context, podID int64) (*agentpod.Pod, error) {
	pod, err := s.repo.GetByID(ctx, podID)
	if err != nil {
		return nil, ErrPodNotFound
	}
	return pod, nil
}

func (s *PodService) GetPodByKey(ctx context.Context, podKey string) (*agentpod.Pod, error) {
	return s.GetPod(ctx, podKey)
}

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

func (s *PodService) GetPodOrganizationAndCreator(ctx context.Context, podKey string) (orgID, creatorID int64, err error) {
	orgID, creatorID, err = s.repo.GetOrgAndCreator(ctx, podKey)
	if err != nil {
		return 0, 0, ErrPodNotFound
	}
	return orgID, creatorID, nil
}

func (s *PodService) GetPodsByTicket(ctx context.Context, ticketID int64) ([]*agentpod.Pod, error) {
	return s.repo.ListByTicket(ctx, ticketID)
}

func (s *PodService) ListPods(ctx context.Context, orgID int64, q agentpod.PodListQuery) ([]*agentpod.Pod, int64, error) {
	pods, total, err := s.repo.ListByOrg(ctx, orgID, q)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list pods", "org_id", orgID, "error", err)
		return nil, 0, err
	}
	_ = s.repo.EnrichWithLoopInfo(ctx, pods)
	return pods, total, nil
}

func (s *PodService) ListActivePods(ctx context.Context, runnerID int64) ([]*agentpod.Pod, error) {
	return s.repo.ListActive(ctx, runnerID)
}

func (s *PodService) ListByRunner(ctx context.Context, runnerID int64, status string) ([]*agentpod.Pod, error) {
	return s.repo.ListByRunner(ctx, runnerID, status)
}

func (s *PodService) ListByTicket(ctx context.Context, ticketID int64) ([]*agentpod.Pod, error) {
	return s.repo.ListByTicket(ctx, ticketID)
}

func (s *PodService) ListPodsByRunner(ctx context.Context, runnerID int64, q agentpod.PodListQuery) ([]*agentpod.Pod, int64, error) {
	return s.repo.ListByRunnerPaginated(ctx, runnerID, q)
}

func (s *PodService) GetActivePodBySourcePodKey(ctx context.Context, sourcePodKey string) (*agentpod.Pod, error) {
	return s.repo.GetActivePodBySourcePodKey(ctx, sourcePodKey)
}

func (s *PodService) ListActiveResumedBy(ctx context.Context, sourcePodKeys []string) (map[string]string, error) {
	return s.repo.ListActiveResumedBy(ctx, sourcePodKeys)
}

func (s *PodService) FindByBranchAndRepo(ctx context.Context, orgID, repoID int64, branchName string) (*agentpod.Pod, error) {
	return s.repo.FindByBranchAndRepo(ctx, orgID, repoID, branchName)
}
