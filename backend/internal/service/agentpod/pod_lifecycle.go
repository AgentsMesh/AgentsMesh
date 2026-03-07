package agentpod

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

// HandlePodCreated handles the pod_created event from runner
func (s *PodService) HandlePodCreated(ctx context.Context, podKey string, ptyPID int, sandboxPath, branchName string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"pty_pid":       ptyPID,
		"status":        agentpod.StatusRunning,
		"started_at":    now,
		"last_activity": now,
	}
	if sandboxPath != "" {
		updates["sandbox_path"] = sandboxPath
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
			"status":      agentpod.StatusTerminated,
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
	if err := s.UpdatePodStatus(ctx, podKey, agentpod.StatusTerminated); err != nil {
		return err
	}

	if s.eventPublisher != nil {
		s.eventPublisher.PublishPodEvent(
			ctx,
			PodEventTerminated,
			pod.OrganizationID,
			podKey,
			agentpod.StatusTerminated,
			previousStatus,
			"",
		)
	}

	return nil
}

// MarkDisconnected marks a pod as disconnected (user closed browser)
func (s *PodService) MarkDisconnected(ctx context.Context, podKey string) error {
	return s.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ? AND status = ?", podKey, agentpod.StatusRunning).
		Update("status", agentpod.StatusDisconnected).Error
}

// MarkReconnected marks a pod as running again (user reconnected)
func (s *PodService) MarkReconnected(ctx context.Context, podKey string) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ? AND status = ?", podKey, agentpod.StatusDisconnected).
		Updates(map[string]interface{}{
			"status":        agentpod.StatusRunning,
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
			agentpod.StatusRunning,
			agentpod.StatusInitializing,
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
	var errs []error
	for _, pod := range dbPods {
		if !reportedSet[pod.PodKey] {
			if err := s.db.WithContext(ctx).Model(pod).Updates(map[string]interface{}{
				"status":      agentpod.StatusOrphaned,
				"finished_at": now,
			}).Error; err != nil {
				errs = append(errs, fmt.Errorf("mark pod %s orphaned: %w", pod.PodKey, err))
			}
		}
	}

	return errors.Join(errs...)
}

// CleanupStalePods marks stale pods as terminated
func (s *PodService) CleanupStalePods(ctx context.Context, maxIdleHours int) (int64, error) {
	threshold := time.Now().Add(-time.Duration(maxIdleHours) * time.Hour)
	now := time.Now()

	result := s.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("status IN ? AND last_activity < ?", []string{
			agentpod.StatusDisconnected,
		}, threshold).
		Updates(map[string]interface{}{
			"status":      agentpod.StatusTerminated,
			"finished_at": now,
		})

	return result.RowsAffected, result.Error
}
