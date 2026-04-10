package agentpod

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

func (s *PodService) GetByKey(ctx context.Context, podKey string) (*agentpod.Pod, error) {
	return s.repo.GetByKey(ctx, podKey)
}

func (s *PodService) GetByKeyAndRunner(ctx context.Context, podKey string, runnerID int64) (*agentpod.Pod, error) {
	return s.repo.GetByKeyAndRunner(ctx, podKey, runnerID)
}

func (s *PodService) ListActiveByRunner(ctx context.Context, runnerID int64) ([]*agentpod.Pod, error) {
	return s.repo.ListActiveByRunner(ctx, runnerID)
}

func (s *PodService) ListInitializingByRunner(ctx context.Context, runnerID int64) ([]*agentpod.Pod, error) {
	return s.repo.ListInitializingByRunner(ctx, runnerID)
}

func (s *PodService) CountActiveByKeys(ctx context.Context, podKeys []string) (int, error) {
	return s.repo.CountActiveByKeys(ctx, podKeys)
}

func (s *PodService) UpdateByKey(ctx context.Context, podKey string, updates map[string]interface{}) (int64, error) {
	return s.repo.UpdateByKey(ctx, podKey, updates)
}

func (s *PodService) UpdateByKeyAndStatus(ctx context.Context, podKey, status string, updates map[string]interface{}) error {
	return s.repo.UpdateByKeyAndStatus(ctx, podKey, status, updates)
}

func (s *PodService) UpdateByKeyAndActiveStatus(ctx context.Context, podKey string, updates map[string]interface{}) (int64, error) {
	return s.repo.UpdateByKeyAndActiveStatus(ctx, podKey, updates)
}

func (s *PodService) UpdateByKeyAndStatusCounted(ctx context.Context, podKey, status string, updates map[string]interface{}) (int64, error) {
	return s.repo.UpdateByKeyAndStatusCounted(ctx, podKey, status, updates)
}

func (s *PodService) UpdateTerminatedIfActive(ctx context.Context, podKey string, updates map[string]interface{}, fallbackErrorCode string) (int64, error) {
	return s.repo.UpdateTerminatedIfActive(ctx, podKey, updates, fallbackErrorCode)
}

func (s *PodService) MarkOrphaned(ctx context.Context, pod *agentpod.Pod, finishedAt time.Time) error {
	return s.repo.MarkOrphaned(ctx, pod, finishedAt)
}

func (s *PodService) UpdateField(ctx context.Context, podKey, field string, value interface{}) error {
	return s.repo.UpdateField(ctx, podKey, field, value)
}

func (s *PodService) UpdateAgentStatus(ctx context.Context, podKey string, updates map[string]interface{}) error {
	return s.repo.UpdateAgentStatus(ctx, podKey, updates)
}
