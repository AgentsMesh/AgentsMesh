package runner

import (
	"context"
	"sort"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

// GetByNodeID returns a runner by its node ID.
// This is used by gRPC server for mTLS authentication.
func (s *Service) GetByNodeID(ctx context.Context, nodeID string) (*runner.Runner, error) {
	var r runner.Runner
	if err := s.db.WithContext(ctx).Where("node_id = ?", nodeID).First(&r).Error; err != nil {
		return nil, ErrRunnerNotFound
	}
	return &r, nil
}

// UpdateLastSeen updates the last heartbeat time for a runner.
// This is called when gRPC server receives messages from a runner.
func (s *Service) UpdateLastSeen(ctx context.Context, runnerID int64) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&runner.Runner{}).
		Where("id = ?", runnerID).
		Updates(map[string]interface{}{
			"last_heartbeat": now,
			"status":         runner.RunnerStatusOnline,
		}).Error
}

// GetRunner returns a runner by ID
// Tries to return from cache first, falls back to database
func (s *Service) GetRunner(ctx context.Context, runnerID int64) (*runner.Runner, error) {
	// Try cache first
	if active, ok := s.activeRunners.Load(runnerID); ok {
		if ar, ok := active.(*ActiveRunner); ok && ar.Runner != nil {
			return ar.Runner, nil
		}
	}

	// Fall back to database
	var r runner.Runner
	if err := s.db.WithContext(ctx).First(&r, runnerID).Error; err != nil {
		return nil, ErrRunnerNotFound
	}
	return &r, nil
}

// ListRunners returns runners for an organization
func (s *Service) ListRunners(ctx context.Context, orgID int64) ([]*runner.Runner, error) {
	var runners []*runner.Runner
	if err := s.db.WithContext(ctx).Where("organization_id = ?", orgID).Find(&runners).Error; err != nil {
		return nil, err
	}
	return runners, nil
}

// ListAvailableRunners returns online runners that can accept pods
func (s *Service) ListAvailableRunners(ctx context.Context, orgID int64) ([]*runner.Runner, error) {
	var runners []*runner.Runner
	if err := s.db.WithContext(ctx).
		Where("organization_id = ? AND status = ? AND is_enabled = ? AND current_pods < max_concurrent_pods", orgID, runner.RunnerStatusOnline, true).
		Find(&runners).Error; err != nil {
		return nil, err
	}
	return runners, nil
}

// SelectAvailableRunner selects an available runner using least-pods strategy
// Prioritizes runners from activeRunners cache for better performance
func (s *Service) SelectAvailableRunner(ctx context.Context, orgID int64) (*runner.Runner, error) {
	// First, try to find available runners from cache
	// This avoids DB round-trip for most cases when runners are actively connected
	var cachedRunners []*ActiveRunner
	s.activeRunners.Range(func(key, value interface{}) bool {
		if ar, ok := value.(*ActiveRunner); ok && ar.Runner != nil {
			r := ar.Runner
			// Check if runner matches criteria
			if r.OrganizationID == orgID &&
				r.Status == runner.RunnerStatusOnline &&
				r.IsEnabled &&
				ar.PodCount < r.MaxConcurrentPods &&
				time.Since(ar.LastPing) < 90*time.Second { // Still active
				cachedRunners = append(cachedRunners, ar)
			}
		}
		return true
	})

	if len(cachedRunners) > 0 {
		// Sort by pod count (least loaded first)
		sort.Slice(cachedRunners, func(i, j int) bool {
			return cachedRunners[i].PodCount < cachedRunners[j].PodCount
		})
		return cachedRunners[0].Runner, nil
	}

	// Fall back to database query if cache miss
	var runners []*runner.Runner
	if err := s.db.WithContext(ctx).
		Where("organization_id = ? AND status = ? AND is_enabled = ? AND current_pods < max_concurrent_pods", orgID, runner.RunnerStatusOnline, true).
		Order("current_pods ASC").
		Find(&runners).Error; err != nil {
		return nil, err
	}

	if len(runners) == 0 {
		return nil, ErrRunnerOffline
	}

	// Return the runner with least pods
	return runners[0], nil
}

// RunnerUpdateInput represents input for updating a runner
type RunnerUpdateInput struct {
	Description       *string `json:"description"`
	MaxConcurrentPods *int    `json:"max_concurrent_pods"`
	IsEnabled         *bool   `json:"is_enabled"`
}

// UpdateRunner updates a runner's configuration
func (s *Service) UpdateRunner(ctx context.Context, runnerID int64, input RunnerUpdateInput) (*runner.Runner, error) {
	var r runner.Runner
	if err := s.db.WithContext(ctx).First(&r, runnerID).Error; err != nil {
		return nil, ErrRunnerNotFound
	}

	updates := make(map[string]interface{})
	if input.Description != nil {
		updates["description"] = *input.Description
	}
	if input.MaxConcurrentPods != nil {
		updates["max_concurrent_pods"] = *input.MaxConcurrentPods
	}
	if input.IsEnabled != nil {
		updates["is_enabled"] = *input.IsEnabled
	}

	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(&r).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	// Reload the runner
	if err := s.db.WithContext(ctx).First(&r, runnerID).Error; err != nil {
		return nil, err
	}

	return &r, nil
}
