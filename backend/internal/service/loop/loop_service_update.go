package loop

import (
	"context"
	"errors"
	"fmt"
	"time"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
)

// Update updates a Loop.
func (s *LoopService) Update(ctx context.Context, orgID int64, slug string, req *UpdateLoopRequest) (*loopDomain.Loop, error) {
	loop, err := s.GetBySlug(ctx, orgID, slug)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.AgentSlug != "" {
		updates["agent_slug"] = req.AgentSlug
	}
	if req.PermissionMode != nil {
		updates["permission_mode"] = *req.PermissionMode
	}
	if req.PromptTemplate != nil {
		updates["prompt_template"] = *req.PromptTemplate
	}
	if req.RepositoryID != nil {
		updates["repository_id"] = *req.RepositoryID
	}
	if req.RunnerID != nil {
		updates["runner_id"] = *req.RunnerID
	}
	if req.BranchName != nil {
		updates["branch_name"] = *req.BranchName
	}
	if req.TicketID != nil {
		updates["ticket_id"] = *req.TicketID
	}
	if req.CredentialProfileID != nil {
		updates["credential_profile_id"] = *req.CredentialProfileID
	}
	if req.ConfigOverrides != nil {
		updates["config_overrides"] = req.ConfigOverrides
	}
	if req.ExecutionMode != nil {
		updates["execution_mode"] = *req.ExecutionMode
	}
	if req.CronExpression != nil {
		updates["cron_expression"] = *req.CronExpression
		if *req.CronExpression != "" {
			schedule, err := cronParser.Parse(*req.CronExpression)
			if err != nil {
				return nil, ErrInvalidCron
			}
			next := schedule.Next(time.Now())
			updates["next_run_at"] = next
		} else {
			updates["next_run_at"] = nil
		}
	}
	if req.AutopilotConfig != nil {
		updates["autopilot_config"] = req.AutopilotConfig
	}
	if req.PromptVariables != nil {
		updates["prompt_variables"] = req.PromptVariables
	}
	if req.CallbackURL != nil {
		if *req.CallbackURL == "" {
			updates["callback_url"] = nil
		} else {
			if err := validateCallbackURL(*req.CallbackURL); err != nil {
				return nil, err
			}
			updates["callback_url"] = *req.CallbackURL
		}
	}
	if req.SandboxStrategy != nil {
		updates["sandbox_strategy"] = *req.SandboxStrategy
	}
	if req.SessionPersistence != nil {
		updates["session_persistence"] = *req.SessionPersistence
	}
	if req.ConcurrencyPolicy != nil {
		updates["concurrency_policy"] = *req.ConcurrencyPolicy
	}
	if req.MaxConcurrentRuns != nil {
		updates["max_concurrent_runs"] = *req.MaxConcurrentRuns
	}
	if req.MaxRetainedRuns != nil {
		updates["max_retained_runs"] = *req.MaxRetainedRuns
	}
	if req.TimeoutMinutes != nil {
		updates["timeout_minutes"] = *req.TimeoutMinutes
	}
	if req.IdleTimeoutSec != nil {
		updates["idle_timeout_sec"] = *req.IdleTimeoutSec
	}

	// When runner changes on a persistent-sandbox loop, break the resume chain.
	if req.RunnerID != nil {
		effectiveRunnerID := *req.RunnerID
		currentRunnerID := int64(0)
		if loop.RunnerID != nil {
			currentRunnerID = *loop.RunnerID
		}
		if effectiveRunnerID != currentRunnerID && loop.IsPersistent() && loop.LastPodKey != nil {
			updates["last_pod_key"] = nil
			updates["sandbox_path"] = nil
		}
	}

	// When switching from persistent to fresh, clear runtime state
	if req.SandboxStrategy != nil && *req.SandboxStrategy == loopDomain.SandboxStrategyFresh &&
		loop.SandboxStrategy == loopDomain.SandboxStrategyPersistent {
		updates["last_pod_key"] = nil
		updates["sandbox_path"] = nil
	}

	// Validate enum fields if present
	execMode := ""
	if req.ExecutionMode != nil {
		execMode = *req.ExecutionMode
	}
	sandboxStrat := ""
	if req.SandboxStrategy != nil {
		sandboxStrat = *req.SandboxStrategy
	}
	concPolicy := ""
	if req.ConcurrencyPolicy != nil {
		concPolicy = *req.ConcurrencyPolicy
	}
	if err := validateEnumFields(execMode, sandboxStrat, concPolicy); err != nil {
		return nil, err
	}

	if len(updates) > 0 {
		if err := s.repo.Update(ctx, loop.ID, updates); err != nil {
			return nil, err
		}
	}

	return s.GetBySlug(ctx, orgID, slug)
}

var ErrHasActiveRuns = errors.New("loop has active runs")

// Delete deletes a Loop (hard delete).
// Atomically checks for active runs — returns ErrHasActiveRuns if any exist.
func (s *LoopService) Delete(ctx context.Context, orgID int64, slug string) error {
	affected, err := s.repo.Delete(ctx, orgID, slug)
	if err != nil {
		if errors.Is(err, loopDomain.ErrHasActiveRuns) {
			return ErrHasActiveRuns
		}
		return err
	}
	if affected == 0 {
		return ErrLoopNotFound
	}
	return nil
}

var validStatuses = map[string]bool{
	loopDomain.StatusEnabled:  true,
	loopDomain.StatusDisabled: true,
}

// SetStatus updates the status of a Loop.
// When re-enabling a cron loop, recalculates next_run_at so cron scheduling resumes immediately.
func (s *LoopService) SetStatus(ctx context.Context, orgID int64, slug string, status string) (*loopDomain.Loop, error) {
	if !validStatuses[status] {
		return nil, fmt.Errorf("%w: status must be 'enabled' or 'disabled'", ErrInvalidEnumValue)
	}

	loop, err := s.GetBySlug(ctx, orgID, slug)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"status": status,
	}

	if status == loopDomain.StatusEnabled && loop.HasCron() {
		schedule, err := cronParser.Parse(*loop.CronExpression)
		if err == nil {
			next := schedule.Next(time.Now())
			updates["next_run_at"] = next
		}
	}

	if err := s.repo.Update(ctx, loop.ID, updates); err != nil {
		return nil, err
	}

	return s.GetBySlug(ctx, orgID, slug)
}
