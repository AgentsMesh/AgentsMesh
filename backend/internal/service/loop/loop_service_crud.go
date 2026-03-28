package loop

import (
	"context"
	"log/slog"
	"strings"
	"time"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
)

// Create creates a new Loop.
func (s *LoopService) Create(ctx context.Context, req *CreateLoopRequest) (*loopDomain.Loop, error) {
	slug := req.Slug
	if slug == "" {
		slug = generateSlug(req.Name)
	}
	if !slugRegex.MatchString(slug) {
		return nil, ErrInvalidSlug
	}

	// Set defaults
	if req.PermissionMode == "" {
		req.PermissionMode = "bypassPermissions"
	}
	if req.ExecutionMode == "" {
		req.ExecutionMode = loopDomain.ExecutionModeAutopilot
	}
	if req.SandboxStrategy == "" {
		req.SandboxStrategy = loopDomain.SandboxStrategyPersistent
	}
	if req.ConcurrencyPolicy == "" {
		req.ConcurrencyPolicy = loopDomain.ConcurrencyPolicySkip
	}
	if req.MaxConcurrentRuns == 0 {
		req.MaxConcurrentRuns = 1
	}
	if req.TimeoutMinutes == 0 {
		req.TimeoutMinutes = 60
	}
	if req.AutopilotConfig == nil {
		req.AutopilotConfig = []byte("{}")
	}
	if req.ConfigOverrides == nil {
		req.ConfigOverrides = []byte("{}")
	}
	if req.PromptVariables == nil {
		req.PromptVariables = []byte("{}")
	}

	if err := validateEnumFields(req.ExecutionMode, req.SandboxStrategy, req.ConcurrencyPolicy); err != nil {
		return nil, err
	}

	if req.CallbackURL != nil {
		if err := validateCallbackURL(*req.CallbackURL); err != nil {
			return nil, err
		}
	}

	// Calculate initial next_run_at for cron loops
	var nextRunAt *time.Time
	if req.CronExpression != nil && *req.CronExpression != "" {
		schedule, err := cronParser.Parse(*req.CronExpression)
		if err != nil {
			return nil, ErrInvalidCron
		}
		next := schedule.Next(time.Now())
		nextRunAt = &next
	}

	loop := &loopDomain.Loop{
		OrganizationID:      req.OrganizationID,
		Name:                req.Name,
		Slug:                slug,
		Description:         req.Description,
		AgentSlug:    req.AgentSlug,
		PermissionMode:      req.PermissionMode,
		PromptTemplate:      req.PromptTemplate,
		PromptVariables:     req.PromptVariables,
		RepositoryID:        req.RepositoryID,
		RunnerID:            req.RunnerID,
		BranchName:          req.BranchName,
		TicketID:            req.TicketID,
		CredentialProfileID: req.CredentialProfileID,
		ConfigOverrides:     req.ConfigOverrides,
		ExecutionMode:       req.ExecutionMode,
		CronExpression:      req.CronExpression,
		AutopilotConfig:     req.AutopilotConfig,
		CallbackURL:         req.CallbackURL,
		Status:              loopDomain.StatusEnabled,
		SandboxStrategy:     req.SandboxStrategy,
		SessionPersistence:  req.SessionPersistence,
		ConcurrencyPolicy:   req.ConcurrencyPolicy,
		MaxConcurrentRuns:   req.MaxConcurrentRuns,
		MaxRetainedRuns:     req.MaxRetainedRuns,
		TimeoutMinutes:      req.TimeoutMinutes,
		IdleTimeoutSec:      req.IdleTimeoutSec,
		CreatedByID:         req.CreatedByID,
		NextRunAt:           nextRunAt,
	}

	if err := s.repo.Create(ctx, loop); err != nil {
		if strings.Contains(err.Error(), "idx_loops_org_slug") {
			return nil, ErrDuplicateSlug
		}
		slog.Error("failed to create loop", "slug", slug, "org_id", req.OrganizationID, "error", err)
		return nil, err
	}

	slog.Info("loop created", "loop_id", loop.ID, "slug", slug, "org_id", req.OrganizationID)
	return loop, nil
}
