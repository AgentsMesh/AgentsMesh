package v1

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	loopService "github.com/anthropics/agentsmesh/backend/internal/service/loop"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// CreateLoop creates a new loop
// POST /api/v1/orgs/:slug/loops
func (h *LoopHandler) CreateLoop(c *gin.Context) {
	var req createLoopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("loop create: binding failed", "error", err)
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)

	// Marshal JSON fields
	autopilotConfig, _ := json.Marshal(req.AutopilotConfig)
	configOverrides, _ := json.Marshal(req.ConfigOverrides)
	promptVariables, _ := json.Marshal(req.PromptVariables)

	maxConcurrentRuns := 1
	if req.MaxConcurrentRuns != nil {
		maxConcurrentRuns = *req.MaxConcurrentRuns
	}
	if maxConcurrentRuns < 1 || maxConcurrentRuns > 10 {
		apierr.ValidationError(c, "max_concurrent_runs must be between 1 and 10")
		return
	}

	maxRetainedRuns := 0 // 0 = unlimited
	if req.MaxRetainedRuns != nil {
		maxRetainedRuns = *req.MaxRetainedRuns
	}
	if maxRetainedRuns < 0 || maxRetainedRuns > 10000 {
		apierr.ValidationError(c, "max_retained_runs must be between 0 and 10000 (0 = unlimited)")
		return
	}

	timeoutMinutes := 60
	if req.TimeoutMinutes != nil {
		timeoutMinutes = *req.TimeoutMinutes
	}
	if timeoutMinutes < 1 || timeoutMinutes > 1440 {
		apierr.ValidationError(c, "timeout_minutes must be between 1 and 1440 (24 hours)")
		return
	}

	sessionPersistence := true
	if req.SessionPersistence != nil {
		sessionPersistence = *req.SessionPersistence
	}

	idleTimeoutSec := 30
	if req.IdleTimeoutSec != nil {
		idleTimeoutSec = *req.IdleTimeoutSec
	}
	if idleTimeoutSec < 0 || idleTimeoutSec > 3600 {
		apierr.ValidationError(c, "idle_timeout_sec must be between 0 and 3600 (0 = disabled)")
		return
	}

	loop, err := h.loopService.Create(c.Request.Context(), &loopService.CreateLoopRequest{
		OrganizationID:      tenant.OrganizationID,
		CreatedByID:         tenant.UserID,
		Name:                req.Name,
		Slug:                req.Slug,
		Description:         req.Description,
		AgentSlug:           req.AgentSlug,
		PermissionMode:      req.PermissionMode,
		PromptTemplate:      req.PromptTemplate,
		PromptVariables:     promptVariables,
		RepositoryID:        req.RepositoryID,
		RunnerID:            req.RunnerID,
		BranchName:          req.BranchName,
		TicketID:            req.TicketID,
		CredentialProfileID: req.CredentialProfileID,
		ConfigOverrides:     configOverrides,
		ExecutionMode:       req.ExecutionMode,
		CronExpression:      req.CronExpression,
		AutopilotConfig:     autopilotConfig,
		CallbackURL:         req.CallbackURL,
		SandboxStrategy:     req.SandboxStrategy,
		SessionPersistence:  sessionPersistence,
		ConcurrencyPolicy:   req.ConcurrencyPolicy,
		MaxConcurrentRuns:   maxConcurrentRuns,
		MaxRetainedRuns:     maxRetainedRuns,
		TimeoutMinutes:      timeoutMinutes,
		IdleTimeoutSec:      idleTimeoutSec,
	})
	if err != nil {
		slog.Warn("loop create: service error", "error", err, "name", req.Name, "slug", req.Slug)
		switch {
		case errors.Is(err, loopService.ErrDuplicateSlug):
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Loop slug already exists")
		case errors.Is(err, loopService.ErrInvalidSlug):
			apierr.ValidationError(c, "Invalid slug format")
		case errors.Is(err, loopService.ErrInvalidCron):
			apierr.ValidationError(c, "Invalid cron expression")
		case errors.Is(err, loopService.ErrInvalidCallbackURL):
			apierr.ValidationError(c, err.Error())
		case errors.Is(err, loopService.ErrInvalidEnumValue):
			apierr.ValidationError(c, err.Error())
		default:
			apierr.InternalError(c, "Failed to create loop")
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"loop": loop,
	})
}

// UpdateLoop updates a loop
// PUT /api/v1/orgs/:slug/loops/:loop_slug
func (h *LoopHandler) UpdateLoop(c *gin.Context) {
	var req updateLoopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)
	loopSlug := c.Param("loop_slug")

	// Validate numeric bounds
	if req.MaxConcurrentRuns != nil && (*req.MaxConcurrentRuns < 1 || *req.MaxConcurrentRuns > 10) {
		apierr.ValidationError(c, "max_concurrent_runs must be between 1 and 10")
		return
	}
	if req.TimeoutMinutes != nil && (*req.TimeoutMinutes < 1 || *req.TimeoutMinutes > 1440) {
		apierr.ValidationError(c, "timeout_minutes must be between 1 and 1440 (24 hours)")
		return
	}
	if req.MaxRetainedRuns != nil && (*req.MaxRetainedRuns < 0 || *req.MaxRetainedRuns > 10000) {
		apierr.ValidationError(c, "max_retained_runs must be between 0 and 10000 (0 = unlimited)")
		return
	}
	if req.IdleTimeoutSec != nil && (*req.IdleTimeoutSec < 0 || *req.IdleTimeoutSec > 3600) {
		apierr.ValidationError(c, "idle_timeout_sec must be between 0 and 3600 (0 = disabled)")
		return
	}

	svcReq := &loopService.UpdateLoopRequest{
		Name:                req.Name,
		Description:         req.Description,
		AgentSlug:           req.AgentSlug,
		PermissionMode:      req.PermissionMode,
		PromptTemplate:      req.PromptTemplate,
		RepositoryID:        req.RepositoryID,
		RunnerID:            req.RunnerID,
		BranchName:          req.BranchName,
		TicketID:            req.TicketID,
		CredentialProfileID: req.CredentialProfileID,
		ExecutionMode:       req.ExecutionMode,
		CronExpression:      req.CronExpression,
		CallbackURL:         req.CallbackURL,
		SandboxStrategy:     req.SandboxStrategy,
		SessionPersistence:  req.SessionPersistence,
		ConcurrencyPolicy:   req.ConcurrencyPolicy,
		MaxConcurrentRuns:   req.MaxConcurrentRuns,
		MaxRetainedRuns:     req.MaxRetainedRuns,
		TimeoutMinutes:      req.TimeoutMinutes,
		IdleTimeoutSec:      req.IdleTimeoutSec,
	}

	if req.ConfigOverrides != nil {
		coBytes, _ := json.Marshal(req.ConfigOverrides)
		svcReq.ConfigOverrides = coBytes
	}
	if req.AutopilotConfig != nil {
		acBytes, _ := json.Marshal(req.AutopilotConfig)
		svcReq.AutopilotConfig = acBytes
	}
	if req.PromptVariables != nil {
		pvBytes, _ := json.Marshal(req.PromptVariables)
		svcReq.PromptVariables = pvBytes
	}

	loop, err := h.loopService.Update(c.Request.Context(), tenant.OrganizationID, loopSlug, svcReq)
	if err != nil {
		switch {
		case errors.Is(err, loopService.ErrLoopNotFound):
			apierr.ResourceNotFound(c, "Loop not found")
		case errors.Is(err, loopService.ErrInvalidCron):
			apierr.ValidationError(c, "Invalid cron expression")
		case errors.Is(err, loopService.ErrInvalidCallbackURL):
			apierr.ValidationError(c, err.Error())
		case errors.Is(err, loopService.ErrInvalidEnumValue):
			apierr.ValidationError(c, err.Error())
		default:
			apierr.InternalError(c, "Failed to update loop")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"loop": loop})
}
