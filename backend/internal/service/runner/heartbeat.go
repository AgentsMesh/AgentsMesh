package runner

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

func (s *Service) Heartbeat(ctx context.Context, runnerID int64, currentPods int) error {
	now := time.Now()
	return s.repo.UpdateFields(ctx, runnerID, map[string]interface{}{
		"last_heartbeat": now,
		"current_pods":   currentPods,
		"status":         runner.RunnerStatusOnline,
	})
}

type HeartbeatPodInfo struct {
	PodKey      string `json:"pod_key"`
	Status      string `json:"status,omitempty"`
	AgentStatus string `json:"agent_status,omitempty"`
}

func (s *Service) UpdateHeartbeatWithPods(ctx context.Context, runnerID int64, pods []HeartbeatPodInfo, version string) error {
	r, err := s.repo.GetByID(ctx, runnerID)
	if err != nil {
		return err
	}
	if r == nil {
		return ErrRunnerNotFound
	}

	now := time.Now()
	updates := map[string]interface{}{
		"last_heartbeat": now,
		"current_pods":   len(pods),
		"status":         runner.RunnerStatusOnline,
	}
	if version != "" {
		updates["runner_version"] = version
	}

	r.CurrentPods = len(pods)
	r.Status = runner.RunnerStatusOnline
	r.LastHeartbeat = &now

	s.activeRunners.Store(runnerID, &ActiveRunner{
		Runner:   r,
		LastPing: now,
		PodCount: len(pods),
	})

	return s.repo.UpdateFields(ctx, runnerID, updates)
}

func (s *Service) MarkOfflineRunners(ctx context.Context, timeout time.Duration) error {
	threshold := time.Now().Add(-timeout)
	return s.repo.MarkOfflineRunners(ctx, threshold)
}
