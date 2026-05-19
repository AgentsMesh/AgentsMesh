package runner

import (
	"context"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

type RunnerUpdateInput struct {
	Description       *string  `json:"description"`
	MaxConcurrentPods *int     `json:"max_concurrent_pods"`
	IsEnabled         *bool    `json:"is_enabled"`
	Visibility        *string  `json:"visibility"`
	Tags              []string `json:"tags"`
}

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
	if input.Tags != nil {
		updates["tags"] = runner.StringSlice(input.Tags)
	}

	if len(updates) > 0 {
		if err := s.repo.UpdateFields(ctx, runnerID, updates); err != nil {
			slog.Error("failed to update runner", "runner_id", runnerID, "error", err)
			return nil, err
		}
		slog.Info("runner updated", "runner_id", runnerID)
	}

	r, err = s.repo.GetByID(ctx, runnerID)
	if err != nil {
		return nil, err
	}
	return r, nil
}
