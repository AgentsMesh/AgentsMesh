package agentpod

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

// HandlePodCreated handles the pod_created event from runner
func (s *PodService) HandlePodCreated(ctx context.Context, podKey string, ptyPID int, worktreePath, branchName string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"pty_pid":       ptyPID,
		"status":        agentpod.PodStatusRunning,
		"started_at":    now,
		"last_activity": now,
	}
	if worktreePath != "" {
		updates["worktree_path"] = worktreePath
	}
	if branchName != "" {
		updates["branch_name"] = branchName
	}
	return s.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ?", podKey).
		Updates(updates).Error
}

// HandlePodTerminated handles the pod_terminated event from runner
func (s *PodService) HandlePodTerminated(ctx context.Context, podKey string, exitCode *int) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ?", podKey).
		Updates(map[string]interface{}{
			"status":      agentpod.PodStatusTerminated,
			"finished_at": now,
			"pty_pid":     nil,
		}).Error
}

// TerminatePod terminates a pod
func (s *PodService) TerminatePod(ctx context.Context, podKey string) error {
	pod, err := s.GetPod(ctx, podKey)
	if err != nil {
		return err
	}

	if !pod.IsActive() {
		return ErrPodTerminated
	}

	previousStatus := pod.Status
	if err := s.UpdatePodStatus(ctx, podKey, agentpod.PodStatusTerminated); err != nil {
		return err
	}

	if s.eventPublisher != nil {
		s.eventPublisher.PublishPodEvent(
			ctx,
			PodEventTerminated,
			pod.OrganizationID,
			podKey,
			agentpod.PodStatusTerminated,
			previousStatus,
			"",
		)
	}

	return nil
}

// MarkDisconnected marks a pod as disconnected (user closed browser)
func (s *PodService) MarkDisconnected(ctx context.Context, podKey string) error {
	return s.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ? AND status = ?", podKey, agentpod.PodStatusRunning).
		Update("status", agentpod.PodStatusDisconnected).Error
}

// MarkReconnected marks a pod as running again (user reconnected)
func (s *PodService) MarkReconnected(ctx context.Context, podKey string) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ? AND status = ?", podKey, agentpod.PodStatusDisconnected).
		Updates(map[string]interface{}{
			"status":        agentpod.PodStatusRunning,
			"last_activity": now,
		}).Error
}

// RecordActivity records pod activity
func (s *PodService) RecordActivity(ctx context.Context, podKey string) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ?", podKey).
		Update("last_activity", now).Error
}

// ReconcilePods marks orphaned pods that are not reported by runner
func (s *PodService) ReconcilePods(ctx context.Context, runnerID int64, reportedPodKeys []string) error {
	var dbPods []*agentpod.Pod
	err := s.db.WithContext(ctx).
		Where("runner_id = ? AND status IN ?", runnerID, []string{
			agentpod.PodStatusRunning,
			agentpod.PodStatusInitializing,
		}).
		Find(&dbPods).Error
	if err != nil {
		return err
	}

	reportedSet := make(map[string]bool)
	for _, key := range reportedPodKeys {
		reportedSet[key] = true
	}

	now := time.Now()
	for _, pod := range dbPods {
		if !reportedSet[pod.PodKey] {
			s.db.WithContext(ctx).Model(pod).Updates(map[string]interface{}{
				"status":      agentpod.PodStatusOrphaned,
				"finished_at": now,
			})
		}
	}

	return nil
}

// CleanupStalePods marks stale pods as terminated
func (s *PodService) CleanupStalePods(ctx context.Context, maxIdleHours int) (int64, error) {
	threshold := time.Now().Add(-time.Duration(maxIdleHours) * time.Hour)
	now := time.Now()

	result := s.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("status IN ? AND last_activity < ?", []string{
			agentpod.PodStatusDisconnected,
		}, threshold).
		Updates(map[string]interface{}{
			"status":      agentpod.PodStatusTerminated,
			"finished_at": now,
		})

	return result.RowsAffected, result.Error
}
