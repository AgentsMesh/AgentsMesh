package agentpod

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	_, err := s.repo.UpdateByKey(ctx, podKey, updates)
	if err != nil {
		slog.Error("failed to handle pod created", "pod_key", podKey, "error", err)
		return err
	}
	slog.Info("pod started on runner", "pod_key", podKey, "sandbox_path", sandboxPath, "pty_pid", ptyPID)
	return nil
}

// HandlePodTerminated handles the pod_terminated event from runner
func (s *PodService) HandlePodTerminated(ctx context.Context, podKey string, exitCode *int) error {
	now := time.Now()
	_, err := s.repo.UpdateByKey(ctx, podKey, map[string]interface{}{
		"status":      agentpod.StatusTerminated,
		"finished_at": now,
		"pty_pid":     nil,
	})
	if err != nil {
		slog.Error("failed to handle pod terminated", "pod_key", podKey, "error", err)
		return err
	}
	slog.Info("pod terminated", "pod_key", podKey, "exit_code", exitCode)
	return nil
}

// TerminatePod terminates a pod
func (s *PodService) TerminatePod(ctx context.Context, podKey string) error {
	pod, err := s.GetPod(ctx, podKey)
	if err != nil {
		slog.Error("failed to get pod for termination", "pod_key", podKey, "error", err)
		return err
	}

	if !pod.IsActive() {
		return ErrPodTerminated
	}

	slog.Info("pod terminate requested", "pod_key", podKey)
	previousStatus := pod.Status
	if err := s.UpdatePodStatus(ctx, podKey, agentpod.StatusTerminated); err != nil {
		slog.Error("failed to terminate pod", "pod_key", podKey, "error", err)
		return err
	}

	slog.Info("pod terminated by user", "pod_key", podKey, "previous_status", previousStatus)

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
	return s.repo.UpdateByKeyAndStatus(ctx, podKey, agentpod.StatusRunning, map[string]interface{}{
		"status": agentpod.StatusDisconnected,
	})
}

// MarkReconnected marks a pod as running again (user reconnected)
func (s *PodService) MarkReconnected(ctx context.Context, podKey string) error {
	now := time.Now()
	return s.repo.UpdateByKeyAndStatus(ctx, podKey, agentpod.StatusDisconnected, map[string]interface{}{
		"status":        agentpod.StatusRunning,
		"last_activity": now,
	})
}

// RecordActivity records pod activity
func (s *PodService) RecordActivity(ctx context.Context, podKey string) error {
	return s.repo.UpdateField(ctx, podKey, "last_activity", time.Now())
}

// ReconcilePods marks orphaned pods that are not reported by runner
func (s *PodService) ReconcilePods(ctx context.Context, runnerID int64, reportedPodKeys []string) error {
	dbPods, err := s.repo.ListActiveByRunner(ctx, runnerID)
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
			if err := s.repo.MarkOrphaned(ctx, pod, now); err != nil {
				errs = append(errs, fmt.Errorf("mark pod %s orphaned: %w", pod.PodKey, err))
			} else {
				slog.Warn("pod marked orphaned during reconciliation", "pod_key", pod.PodKey, "runner_id", runnerID)
			}
		}
	}

	return errors.Join(errs...)
}

// CleanupStalePods marks stale pods as terminated
func (s *PodService) CleanupStalePods(ctx context.Context, maxIdleHours int) (int64, error) {
	threshold := time.Now().Add(-time.Duration(maxIdleHours) * time.Hour)
	count, err := s.repo.CleanupStale(ctx, threshold)
	if err != nil {
		slog.Error("failed to cleanup stale pods", "max_idle_hours", maxIdleHours, "error", err)
		return 0, err
	}
	if count > 0 {
		slog.Info("cleaned up stale pods", "count", count, "max_idle_hours", maxIdleHours)
	}
	return count, nil
}

// MarkInitFailed marks an initializing pod as error with the given code and message.
// This is called when pod dispatch fails (e.g., runner unreachable). The requesting
// client receives the error via HTTP response; other clients will see the error
// status on their next pod list refresh or via subsequent status change events.
func (s *PodService) MarkInitFailed(ctx context.Context, podKey, errorCode, errorMessage string) error {
	now := time.Now()
	_, err := s.repo.UpdateByKeyAndStatusCounted(ctx, podKey, agentpod.StatusInitializing, map[string]interface{}{
		"status":        agentpod.StatusError,
		"error_code":    errorCode,
		"error_message": errorMessage,
		"finished_at":   now,
	})
	if err != nil {
		slog.Error("failed to mark pod init failed", "pod_key", podKey, "error_code", errorCode, "error", err)
		return err
	}
	slog.Warn("pod init failed", "pod_key", podKey, "error_code", errorCode, "error_message", errorMessage)
	return nil
}
