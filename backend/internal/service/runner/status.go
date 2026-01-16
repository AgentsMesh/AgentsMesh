package runner

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

// UpdateRunnerStatus updates runner status
func (s *Service) UpdateRunnerStatus(ctx context.Context, runnerID int64, status string) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&runner.Runner{}).Where("id = ?", runnerID).Updates(map[string]interface{}{
		"status":         status,
		"last_heartbeat": now,
	}).Error
}

// SetRunnerStatus sets the runner status (alias for UpdateRunnerStatus)
func (s *Service) SetRunnerStatus(ctx context.Context, runnerID int64, status string) error {
	return s.UpdateRunnerStatus(ctx, runnerID, status)
}

// IsConnected checks if a runner has an active connection
func (s *Service) IsConnected(runnerID int64) bool {
	_, exists := s.activeRunners.Load(runnerID)
	return exists
}

// MarkConnected marks a runner as connected
func (s *Service) MarkConnected(ctx context.Context, runnerID int64) error {
	r, err := s.GetRunner(ctx, runnerID)
	if err != nil {
		return err
	}

	// Update status in the Runner object before caching
	r.Status = runner.RunnerStatusOnline

	now := time.Now()
	s.activeRunners.Store(runnerID, &ActiveRunner{
		Runner:   r,
		LastPing: now,
		PodCount: r.CurrentPods,
	})

	return s.UpdateRunnerStatus(ctx, runnerID, runner.RunnerStatusOnline)
}

// MarkDisconnected marks a runner as disconnected
func (s *Service) MarkDisconnected(ctx context.Context, runnerID int64) error {
	s.activeRunners.Delete(runnerID)
	return s.UpdateRunnerStatus(ctx, runnerID, runner.RunnerStatusOffline)
}

// UpdateHostInfo updates runner host information
func (s *Service) UpdateHostInfo(ctx context.Context, runnerID int64, hostInfo map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&runner.Runner{}).
		Where("id = ?", runnerID).
		Update("host_info", hostInfo).Error
}

// UpdateAvailableAgents updates the list of available agents for a runner
// Called when runner completes initialization handshake
func (s *Service) UpdateAvailableAgents(ctx context.Context, runnerID int64, agents []string) error {
	return s.db.WithContext(ctx).Model(&runner.Runner{}).
		Where("id = ?", runnerID).
		Update("available_agents", runner.StringSlice(agents)).Error
}

// IncrementPods increments the pod count for a runner
func (s *Service) IncrementPods(ctx context.Context, runnerID int64) error {
	return s.db.WithContext(ctx).Exec(
		"UPDATE runners SET current_pods = current_pods + 1 WHERE id = ?",
		runnerID,
	).Error
}

// DecrementPods decrements the pod count for a runner
func (s *Service) DecrementPods(ctx context.Context, runnerID int64) error {
	return s.db.WithContext(ctx).Exec(
		"UPDATE runners SET current_pods = GREATEST(current_pods - 1, 0) WHERE id = ?",
		runnerID,
	).Error
}

// RunnerUpdateFunc is a callback for runner status updates
type RunnerUpdateFunc func(*runner.Runner)

// SubscribeStatusChanges subscribes to runner status changes and returns an unsubscribe function
func (s *Service) SubscribeStatusChanges(ctx context.Context, callback RunnerUpdateFunc) (func(), error) {
	// In a real implementation, this would use Redis pub/sub or similar
	// For now, return a simple unsubscribe function
	return func() {}, nil
}
