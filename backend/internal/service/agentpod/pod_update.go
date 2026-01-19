package agentpod

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

// UpdatePodStatus updates pod status
func (s *PodService) UpdatePodStatus(ctx context.Context, podKey, status string) error {
	updates := map[string]interface{}{"status": status}

	if status == agentpod.StatusRunning {
		updates["started_at"] = time.Now()
	} else if status == agentpod.StatusTerminated || status == agentpod.StatusOrphaned {
		updates["finished_at"] = time.Now()
	}

	result := s.db.WithContext(ctx).Model(&agentpod.Pod{}).Where("pod_key = ?", podKey).Updates(updates)
	if result.RowsAffected == 0 {
		return ErrPodNotFound
	}

	if status == agentpod.StatusTerminated || status == agentpod.StatusOrphaned {
		var pod agentpod.Pod
		s.db.WithContext(ctx).Where("pod_key = ?", podKey).First(&pod)
		s.db.WithContext(ctx).Exec("UPDATE runners SET current_pods = GREATEST(current_pods - 1, 0) WHERE id = ?", pod.RunnerID)
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
	return s.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ?", podKey).
		Updates(updates).Error
}

// UpdatePodPTY updates pod PTY PID
func (s *PodService) UpdatePodPTY(ctx context.Context, podKey string, ptyPID int) error {
	return s.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ?", podKey).
		Update("pty_pid", ptyPID).Error
}

// UpdatePodTitle updates pod title (from OSC 0/2 terminal escape sequences)
func (s *PodService) UpdatePodTitle(ctx context.Context, podKey, title string) error {
	return s.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ?", podKey).
		Update("title", title).Error
}

// UpdateWorktreePath updates pod worktree path and branch
func (s *PodService) UpdateWorktreePath(ctx context.Context, podKey, worktreePath, branchName string) error {
	updates := map[string]interface{}{"worktree_path": worktreePath}
	if branchName != "" {
		updates["branch_name"] = branchName
	}
	return s.db.WithContext(ctx).Model(&agentpod.Pod{}).
		Where("pod_key = ?", podKey).
		Updates(updates).Error
}

// PodUpdateFunc is a callback for pod updates
type PodUpdateFunc func(*agentpod.Pod)

// Subscribe subscribes to pod updates and returns an unsubscribe function
func (s *PodService) Subscribe(ctx context.Context, podKey string, callback PodUpdateFunc) (func(), error) {
	return func() {}, nil
}
