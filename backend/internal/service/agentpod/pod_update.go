package agentpod

import (
	"context"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

// UpdatePodStatus updates pod status
func (s *PodService) UpdatePodStatus(ctx context.Context, podKey, status string) error {
	updates := map[string]interface{}{"status": status}

	switch status {
	case agentpod.StatusRunning:
		updates["started_at"] = time.Now()
	case agentpod.StatusTerminated, agentpod.StatusOrphaned:
		updates["finished_at"] = time.Now()
	}

	rowsAffected, err := s.repo.UpdateByKey(ctx, podKey, updates)
	if err != nil {
		slog.Error("failed to update pod status", "pod_key", podKey, "status", status, "error", err)
		return err
	}
	if rowsAffected == 0 {
		return ErrPodNotFound
	}

	if status == agentpod.StatusTerminated || status == agentpod.StatusOrphaned {
		pod, err := s.repo.GetByKey(ctx, podKey)
		if err == nil {
			_ = s.repo.DecrementRunnerPods(ctx, pod.RunnerID)
		}
	}

	return nil
}

// UpdateAgentStatus updates agent status
func (s *PodService) UpdateAgentStatus(ctx context.Context, podKey, agentStatus string, agentPID *int) error {
	updates := map[string]interface{}{
		"agent_status":  agentStatus,
		"last_activity": time.Now(),
	}
	if agentPID != nil {
		updates["agent_pid"] = *agentPID
	}
	return s.repo.UpdateAgentStatus(ctx, podKey, updates)
}

// UpdatePodPTY updates pod PTY PID
func (s *PodService) UpdatePodPTY(ctx context.Context, podKey string, ptyPID int) error {
	return s.repo.UpdateField(ctx, podKey, "pty_pid", ptyPID)
}

// UpdatePodTitle updates pod title (from OSC 0/2 terminal escape sequences)
func (s *PodService) UpdatePodTitle(ctx context.Context, podKey, title string) error {
	return s.repo.UpdateField(ctx, podKey, "title", title)
}

// UpdateAlias updates the user-assigned alias for a pod.
// Pass nil to clear the alias.
func (s *PodService) UpdateAlias(ctx context.Context, podKey string, alias *string) error {
	if alias != nil {
		return s.repo.UpdateField(ctx, podKey, "alias", *alias)
	}
	return s.repo.UpdateField(ctx, podKey, "alias", nil)
}

// UpdatePerpetual updates the perpetual flag for a pod.
func (s *PodService) UpdatePerpetual(ctx context.Context, podKey string, perpetual bool) error {
	return s.repo.UpdateField(ctx, podKey, "perpetual", perpetual)
}

// UpdateSandboxPath updates pod sandbox path and branch
func (s *PodService) UpdateSandboxPath(ctx context.Context, podKey, sandboxPath, branchName string) error {
	updates := map[string]interface{}{"sandbox_path": sandboxPath}
	if branchName != "" {
		updates["branch_name"] = branchName
	}
	_, err := s.repo.UpdateByKey(ctx, podKey, updates)
	return err
}

// PodUpdateFunc is a callback for pod updates
type PodUpdateFunc func(*agentpod.Pod)

// Subscribe subscribes to pod updates and returns an unsubscribe function
func (s *PodService) Subscribe(ctx context.Context, podKey string, callback PodUpdateFunc) (func(), error) {
	return func() {}, nil
}
