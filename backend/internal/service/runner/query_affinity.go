package runner

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

func (s *Service) SelectRunnerWithAffinity(
	ctx context.Context,
	orgID int64, userID int64, agentSlug string,
	hints *runner.AffinityHints,
	repoHistory map[int64]int,
) (*runner.Runner, error) {
	if hints == nil {
		return s.SelectAvailableRunnerForAgent(ctx, orgID, userID, agentSlug)
	}

	cachedRunners := s.collectEligibleRunners(ctx, orgID, userID, agentSlug)

	if len(cachedRunners) > 0 {
		return s.selectWithScoring(cachedRunners, userID, hints, repoHistory)
	}

	agentJSON, err := json.Marshal([]string{agentSlug})
	if err != nil {
		return nil, err
	}
	dbRunners, err := s.repo.ListAvailableForAgent(ctx, orgID, userID, string(agentJSON))
	if err != nil {
		slog.Error("failed to select runner for agent from DB", "org_id", orgID, "agent_slug", agentSlug, "error", err)
		return nil, err
	}
	if len(dbRunners) == 0 {
		slog.Warn("no runner available for agent", "org_id", orgID, "agent_slug", agentSlug)
		return nil, ErrNoRunnerForAgent
	}

	dbCandidates := make([]*ActiveRunner, len(dbRunners))
	for i, r := range dbRunners {
		dbCandidates[i] = &ActiveRunner{Runner: r, PodCount: r.CurrentPods}
	}
	return s.selectWithScoring(dbCandidates, userID, hints, repoHistory)
}

func (s *Service) selectWithScoring(
	candidates []*ActiveRunner,
	userID int64,
	hints *runner.AffinityHints,
	repoHistory map[int64]int,
) (*runner.Runner, error) {
	ranked := ScoreRunners(candidates, userID, hints, repoHistory, runner.DefaultAffinityWeights())
	if len(ranked) == 0 {
		return nil, ErrNoRunnerForAgent
	}
	slog.Info("runner selected with affinity",
		"runner_id", ranked[0].Runner.ID,
		"score_count", len(ranked),
		"has_repo_hint", hints.RepositoryID != nil,
		"has_tags", len(hints.Tags) > 0,
	)
	return ranked[0].Runner, nil
}
