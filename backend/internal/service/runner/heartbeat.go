package runner

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

// Heartbeat updates runner heartbeat
func (s *Service) Heartbeat(ctx context.Context, runnerID int64, currentPods int) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&runner.Runner{}).Where("id = ?", runnerID).Updates(map[string]interface{}{
		"last_heartbeat": now,
		"current_pods":   currentPods,
		"status":         runner.RunnerStatusOnline,
	}).Error
}

// HeartbeatPodInfo represents a pod reported in heartbeat
type HeartbeatPodInfo struct {
	PodKey      string `json:"pod_key"`
	Status      string `json:"status,omitempty"`
	AgentStatus string `json:"agent_status,omitempty"`
}

// UpdateHeartbeatWithPods updates runner heartbeat with pod reconciliation
func (s *Service) UpdateHeartbeatWithPods(ctx context.Context, runnerID int64, pods []HeartbeatPodInfo, version string) error {
	var r runner.Runner
	if err := s.db.WithContext(ctx).First(&r, runnerID).Error; err != nil {
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

	// Update active runner in memory
	s.activeRunners.Store(runnerID, &ActiveRunner{
		Runner:   &r,
		LastPing: now,
		PodCount: len(pods),
	})

	return s.db.WithContext(ctx).Model(&r).Updates(updates).Error
}

// MarkOfflineRunners marks runners as offline if no heartbeat received
func (s *Service) MarkOfflineRunners(ctx context.Context, timeout time.Duration) error {
	threshold := time.Now().Add(-timeout)
	return s.db.WithContext(ctx).Model(&runner.Runner{}).
		Where("status = ? AND last_heartbeat < ?", runner.RunnerStatusOnline, threshold).
		Update("status", runner.RunnerStatusOffline).Error
}
