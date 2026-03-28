package runner

import (
	"context"
	"encoding/json"
	"log/slog"
	"sort"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

// GetByNodeID returns a runner by its node ID.
// This is used by gRPC server for mTLS authentication.
func (s *Service) GetByNodeID(ctx context.Context, nodeID string) (*runner.Runner, error) {
	r, err := s.repo.GetByNodeID(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrRunnerNotFound
	}
	return r, nil
}

// GetByNodeIDAndOrgID returns a runner by node ID within a specific organization.
// This prevents cross-org runner mismatch when the same node_id exists in multiple orgs.
func (s *Service) GetByNodeIDAndOrgID(ctx context.Context, nodeID string, orgID int64) (*runner.Runner, error) {
	r, err := s.repo.GetByNodeIDAndOrgID(ctx, nodeID, orgID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrRunnerNotFound
	}
	return r, nil
}

// UpdateLastSeen updates the last heartbeat time for a runner.
// This is called when gRPC server receives messages from a runner.
func (s *Service) UpdateLastSeen(ctx context.Context, runnerID int64) error {
	now := time.Now()
	return s.repo.UpdateFields(ctx, runnerID, map[string]interface{}{
		"last_heartbeat": now,
		"status":         runner.RunnerStatusOnline,
	})
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
	r, err := s.repo.GetByID(ctx, runnerID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrRunnerNotFound
	}
	return r, nil
}

// ListRunners returns runners for an organization, filtered by visibility.
// Organization-visible runners are returned for all users; private runners only for the registrant.
func (s *Service) ListRunners(ctx context.Context, orgID int64, userID int64) ([]*runner.Runner, error) {
	return s.repo.ListByOrg(ctx, orgID, userID)
}

// ListAvailableRunners returns online runners that can accept pods, filtered by visibility.
func (s *Service) ListAvailableRunners(ctx context.Context, orgID int64, userID int64) ([]*runner.Runner, error) {
	return s.repo.ListAvailable(ctx, orgID, userID)
}

// SelectAvailableRunner selects an available runner using least-pods strategy, filtered by visibility.
// Prioritizes runners from activeRunners cache for better performance.
func (s *Service) SelectAvailableRunner(ctx context.Context, orgID int64, userID int64) (*runner.Runner, error) {
	// First, try to find available runners from cache
	var cachedRunners []*ActiveRunner
	s.activeRunners.Range(func(key, value interface{}) bool {
		if ar, ok := value.(*ActiveRunner); ok && ar.Runner != nil {
			r := ar.Runner
			if r.OrganizationID == orgID &&
				r.Status == runner.RunnerStatusOnline &&
				r.IsEnabled &&
				ar.PodCount < r.MaxConcurrentPods &&
				time.Since(ar.LastPing) < 90*time.Second &&
				(r.Visibility == runner.VisibilityOrganization || (r.Visibility == runner.VisibilityPrivate && r.RegisteredByUserID != nil && *r.RegisteredByUserID == userID)) {
				cachedRunners = append(cachedRunners, ar)
			}
		}
		return true
	})

	if len(cachedRunners) > 0 {
		sort.Slice(cachedRunners, func(i, j int) bool {
			return cachedRunners[i].PodCount < cachedRunners[j].PodCount
		})
		return cachedRunners[0].Runner, nil
	}

	// Fall back to database query if cache miss
	runners, err := s.repo.ListAvailableOrdered(ctx, orgID, userID)
	if err != nil {
		slog.Error("failed to select available runner from DB", "org_id", orgID, "error", err)
		return nil, err
	}
	if len(runners) == 0 {
		slog.Warn("no available runner found", "org_id", orgID, "user_id", userID)
		return nil, ErrRunnerOffline
	}
	return runners[0], nil
}

// SelectAvailableRunnerForAgent selects an available runner that supports the given agent type, filtered by visibility.
// Uses the same cache-first, DB-fallback pattern as SelectAvailableRunner with agent compatibility filtering.
func (s *Service) SelectAvailableRunnerForAgent(ctx context.Context, orgID int64, userID int64, agentSlug string) (*runner.Runner, error) {
	// First, try to find available runners from cache
	var cachedRunners []*ActiveRunner
	s.activeRunners.Range(func(key, value interface{}) bool {
		if ar, ok := value.(*ActiveRunner); ok && ar.Runner != nil {
			r := ar.Runner
			if r.OrganizationID == orgID &&
				r.Status == runner.RunnerStatusOnline &&
				r.IsEnabled &&
				ar.PodCount < r.MaxConcurrentPods &&
				time.Since(ar.LastPing) < 90*time.Second &&
				r.SupportsAgent(agentSlug) &&
				(r.Visibility == runner.VisibilityOrganization || (r.Visibility == runner.VisibilityPrivate && r.RegisteredByUserID != nil && *r.RegisteredByUserID == userID)) {
				cachedRunners = append(cachedRunners, ar)
			}
		}
		return true
	})

	if len(cachedRunners) > 0 {
		sort.Slice(cachedRunners, func(i, j int) bool {
			return cachedRunners[i].PodCount < cachedRunners[j].PodCount
		})
		return cachedRunners[0].Runner, nil
	}

	// Fall back to database query with JSONB contains filter
	agentJSON, err := json.Marshal([]string{agentSlug})
	if err != nil {
		return nil, err
	}

	runners, err := s.repo.ListAvailableForAgent(ctx, orgID, userID, string(agentJSON))
	if err != nil {
		slog.Error("failed to select runner for agent from DB", "org_id", orgID, "agent_slug", agentSlug, "error", err)
		return nil, err
	}
	if len(runners) == 0 {
		slog.Warn("no runner available for agent", "org_id", orgID, "agent_slug", agentSlug)
		return nil, ErrNoRunnerForAgent
	}
	return runners[0], nil
}

// RunnerUpdateInput represents input for updating a runner
type RunnerUpdateInput struct {
	Description       *string `json:"description"`
	MaxConcurrentPods *int    `json:"max_concurrent_pods"`
	IsEnabled         *bool   `json:"is_enabled"`
	Visibility        *string `json:"visibility"`
}

// UpdateRunner updates a runner's configuration
func (s *Service) UpdateRunner(ctx context.Context, runnerID int64, input RunnerUpdateInput) (*runner.Runner, error) {
	r, err := s.repo.GetByID(ctx, runnerID)
	if err != nil {
		return nil, err
	}
	if r == nil {
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
	if input.Visibility != nil {
		v := *input.Visibility
		if v == runner.VisibilityOrganization || v == runner.VisibilityPrivate {
			updates["visibility"] = v
		}
	}

	if len(updates) > 0 {
		if err := s.repo.UpdateFields(ctx, runnerID, updates); err != nil {
			slog.Error("failed to update runner", "runner_id", runnerID, "error", err)
			return nil, err
		}
		slog.Info("runner updated", "runner_id", runnerID)
	}

	// Reload the runner
	r, err = s.repo.GetByID(ctx, runnerID)
	if err != nil {
		return nil, err
	}
	return r, nil
}
