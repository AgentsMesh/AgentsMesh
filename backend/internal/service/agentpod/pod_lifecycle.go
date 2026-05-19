package agentpod

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

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
		slog.ErrorContext(ctx, "failed to handle pod created", "pod_key", podKey, "error", err)
		return err
	}
	slog.InfoContext(ctx, "pod started on runner", "pod_key", podKey, "sandbox_path", sandboxPath, "pty_pid", ptyPID)
	return nil
}

func (s *PodService) HandlePodTerminated(ctx context.Context, podKey string, exitCode *int) error {
	now := time.Now()
	_, err := s.repo.UpdateByKey(ctx, podKey, map[string]interface{}{
		"status":      agentpod.StatusTerminated,
		"finished_at": now,
		"pty_pid":     nil,
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to handle pod terminated", "pod_key", podKey, "error", err)
		return err
	}
	slog.InfoContext(ctx, "pod terminated", "pod_key", podKey, "exit_code", exitCode)
	return nil
}

func (s *PodService) MarkDisconnected(ctx context.Context, podKey string) error {
	return s.repo.UpdateByKeyAndStatus(ctx, podKey, agentpod.StatusRunning, map[string]interface{}{
		"status": agentpod.StatusDisconnected,
	})
}

func (s *PodService) MarkReconnected(ctx context.Context, podKey string) error {
	now := time.Now()
	return s.repo.UpdateByKeyAndStatus(ctx, podKey, agentpod.StatusDisconnected, map[string]interface{}{
		"status":        agentpod.StatusRunning,
		"last_activity": now,
	})
}

func (s *PodService) RecordActivity(ctx context.Context, podKey string) error {
	return s.repo.UpdateField(ctx, podKey, "last_activity", time.Now())
}

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
				slog.WarnContext(ctx, "pod marked orphaned during reconciliation", "pod_key", pod.PodKey, "runner_id", runnerID)
			}
		}
	}

	return errors.Join(errs...)
}

func (s *PodService) CleanupStalePods(ctx context.Context, maxIdleHours int) (int64, error) {
	threshold := time.Now().Add(-time.Duration(maxIdleHours) * time.Hour)
	count, err := s.repo.CleanupStale(ctx, threshold)
	if err != nil {
		slog.ErrorContext(ctx, "failed to cleanup stale pods", "max_idle_hours", maxIdleHours, "error", err)
		return 0, err
	}
	if count > 0 {
		slog.InfoContext(ctx, "cleaned up stale pods", "count", count, "max_idle_hours", maxIdleHours)
	}
	return count, nil
}

func (s *PodService) MarkInitFailed(ctx context.Context, podKey, errorCode, errorMessage string) error {
	now := time.Now()
	_, err := s.repo.UpdateByKeyAndStatusCounted(ctx, podKey, agentpod.StatusInitializing, map[string]interface{}{
		"status":        agentpod.StatusError,
		"error_code":    errorCode,
		"error_message": errorMessage,
		"finished_at":   now,
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to mark pod init failed", "pod_key", podKey, "error_code", errorCode, "error", err)
		return err
	}
	slog.WarnContext(ctx, "pod init failed", "pod_key", podKey, "error_code", errorCode, "error_message", errorMessage)
	return nil
}
