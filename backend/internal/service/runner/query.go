package runner

import (
	"context"
	"encoding/json"
	"log/slog"
	"sort"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

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

func (s *Service) UpdateLastSeen(ctx context.Context, runnerID int64) error {
	now := time.Now()
	return s.repo.UpdateFields(ctx, runnerID, map[string]interface{}{
		"last_heartbeat": now,
		"status":         runner.RunnerStatusOnline,
	})
}

func (s *Service) GetRunner(ctx context.Context, runnerID int64) (*runner.Runner, error) {
	if active, ok := s.activeRunners.Load(runnerID); ok {
		if ar, ok := active.(*ActiveRunner); ok && ar.Runner != nil {
			return ar.Runner, nil
		}
	}

	r, err := s.repo.GetByID(ctx, runnerID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrRunnerNotFound
	}
	return r, nil
}

func (s *Service) ListRunners(ctx context.Context, orgID int64, userID int64) ([]*runner.Runner, error) {
	return s.repo.ListByOrg(ctx, orgID, userID)
}

func (s *Service) ListAvailableRunners(ctx context.Context, orgID int64, userID int64) ([]*runner.Runner, error) {
	return s.repo.ListAvailable(ctx, orgID, userID)
}

func (s *Service) SelectAvailableRunner(ctx context.Context, orgID int64, userID int64) (*runner.Runner, error) {
	cachedRunners := s.collectEligibleRunners(ctx, orgID, userID, "")

	if len(cachedRunners) > 0 {
		sort.Slice(cachedRunners, func(i, j int) bool {
			return cachedRunners[i].PodCount < cachedRunners[j].PodCount
		})
		return cachedRunners[0].Runner, nil
	}

	runners, err := s.repo.ListAvailableOrdered(ctx, orgID, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to select available runner from DB", "org_id", orgID, "error", err)
		return nil, err
	}
	if len(runners) == 0 {
		slog.WarnContext(ctx, "no available runner found", "org_id", orgID, "user_id", userID)
		return nil, ErrRunnerOffline
	}
	return runners[0], nil
}

func (s *Service) SelectAvailableRunnerForAgent(ctx context.Context, orgID int64, userID int64, agentSlug string) (*runner.Runner, error) {
	cachedRunners := s.collectEligibleRunners(ctx, orgID, userID, agentSlug)

	if len(cachedRunners) > 0 {
		sort.Slice(cachedRunners, func(i, j int) bool {
			return cachedRunners[i].PodCount < cachedRunners[j].PodCount
		})
		return cachedRunners[0].Runner, nil
	}

	agentJSON, err := json.Marshal([]string{agentSlug})
	if err != nil {
		return nil, err
	}

	runners, err := s.repo.ListAvailableForAgent(ctx, orgID, userID, string(agentJSON))
	if err != nil {
		slog.ErrorContext(ctx, "failed to select runner for agent from DB", "org_id", orgID, "agent_slug", agentSlug, "error", err)
		return nil, err
	}
	if len(runners) == 0 {
		slog.WarnContext(ctx, "no runner available for agent", "org_id", orgID, "agent_slug", agentSlug)
		return nil, ErrNoRunnerForAgent
	}
	return runners[0], nil
}
