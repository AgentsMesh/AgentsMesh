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

// UpdateRunnerVersionAndHostInfo updates runner version and host information atomically.
// Called during gRPC initialization handshake to persist RunnerInfo from the connect request.
func (s *Service) UpdateRunnerVersionAndHostInfo(ctx context.Context, runnerID int64, version string, hostInfo map[string]interface{}) error {
	updates := map[string]interface{}{
		"runner_version": version,
		"host_info":      hostInfo,
	}
	return s.db.WithContext(ctx).Model(&runner.Runner{}).
		Where("id = ?", runnerID).
		Updates(updates).Error
}

// UpdateAvailableAgents updates the list of available agents for a runner
// Called when runner completes initialization handshake
func (s *Service) UpdateAvailableAgents(ctx context.Context, runnerID int64, agents []string) error {
	return s.db.WithContext(ctx).Model(&runner.Runner{}).
		Where("id = ?", runnerID).
		Update("available_agents", runner.StringSlice(agents)).Error
}

// UpdateAgentVersions updates the detected agent version info for a runner.
// Called when runner completes initialization handshake (Runner >= 0.4.7).
// Also refreshes the activeRunners cache to keep GetRunner consistent.
func (s *Service) UpdateAgentVersions(ctx context.Context, runnerID int64, versions []runner.AgentVersion) error {
	if err := s.db.WithContext(ctx).Model(&runner.Runner{}).
		Where("id = ?", runnerID).
		Update("agent_versions", runner.AgentVersionSlice(versions)).Error; err != nil {
		return err
	}

	// Sync in-memory cache so GetRunner returns fresh data immediately.
	// Store a new copy to avoid data races with concurrent Range readers.
	if active, ok := s.activeRunners.Load(runnerID); ok {
		if ar, ok := active.(*ActiveRunner); ok && ar.Runner != nil {
			updated := *ar.Runner
			updated.AgentVersions = runner.AgentVersionSlice(versions)
			s.activeRunners.Store(runnerID, &ActiveRunner{
				Runner:   &updated,
				LastPing: ar.LastPing,
				PodCount: ar.PodCount,
			})
		}
	}
	return nil
}

// MergeAgentVersions merges delta agent version updates into existing versions.
// Entries where both Version and Path are empty are treated as removals (agent no longer available).
// Called when runner reports version changes via heartbeat.
//
// NOTE: This method uses read-modify-write and is NOT safe for concurrent calls on the same runner.
// Correctness relies on gRPC recvLoop serializing all heartbeat messages per runner.
func (s *Service) MergeAgentVersions(ctx context.Context, runnerID int64, changes map[string]runner.AgentVersion) error {
	if len(changes) == 0 {
		return nil
	}

	r, err := s.GetRunner(ctx, runnerID)
	if err != nil {
		return err
	}

	// Build merged version map from existing data
	merged := make(map[string]runner.AgentVersion)
	for _, v := range r.AgentVersions {
		merged[v.Slug] = v
	}

	// Apply changes
	for slug, change := range changes {
		if change.Version == "" && change.Path == "" {
			// Removal: agent no longer available
			delete(merged, slug)
		} else {
			merged[slug] = change
		}
	}

	// Convert back to slice
	result := make([]runner.AgentVersion, 0, len(merged))
	for _, v := range merged {
		result = append(result, v)
	}

	return s.UpdateAgentVersions(ctx, runnerID, result)
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
