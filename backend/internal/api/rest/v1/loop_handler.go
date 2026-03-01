package v1

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	loopDomain "github.com/AgentsMesh/AgentsMesh/backend/internal/domain/loop"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	loopService "github.com/AgentsMesh/AgentsMesh/backend/internal/service/loop"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// PodTerminatorForLoop defines the minimal interface needed by LoopHandler
// to terminate Pods (used for cancel run). Follows ISP — handler only needs TerminatePod.
type PodTerminatorForLoop interface {
	TerminatePod(ctx context.Context, podKey string) error
}

// LoopHandler handles loop-related requests
type LoopHandler struct {
	loopService    *loopService.LoopService
	loopRunService *loopService.LoopRunService
	orchestrator   *loopService.LoopOrchestrator
	podTerminator  PodTerminatorForLoop
}

// NewLoopHandler creates a new loop handler
func NewLoopHandler(
	ls *loopService.LoopService,
	lrs *loopService.LoopRunService,
	orch *loopService.LoopOrchestrator,
	podTerminator PodTerminatorForLoop,
) *LoopHandler {
	return &LoopHandler{
		loopService:    ls,
		loopRunService: lrs,
		orchestrator:   orch,
		podTerminator:  podTerminator,
	}
}

// ========== Request Types ==========

type createLoopRequest struct {
	Name              string                 `json:"name" binding:"required,min=1,max=255"`
	Slug              string                 `json:"slug"`
	Description       *string                `json:"description"`
	AgentTypeID       *int64                 `json:"agent_type_id"`
	CustomAgentTypeID *int64                 `json:"custom_agent_type_id"`
	PermissionMode    string                 `json:"permission_mode"`
	PromptTemplate    string                 `json:"prompt_template" binding:"required"`
	PromptVariables   map[string]interface{} `json:"prompt_variables"`
	RepositoryID      *int64                 `json:"repository_id"`
	RunnerID          *int64                 `json:"runner_id"`
	BranchName        *string                `json:"branch_name"`
	TicketID          *int64                 `json:"ticket_id"`
	CredentialProfileID *int64               `json:"credential_profile_id"`
	ConfigOverrides   map[string]interface{} `json:"config_overrides"`
	ExecutionMode     string                 `json:"execution_mode"`
	CronExpression    *string                `json:"cron_expression"`
	AutopilotConfig   map[string]interface{} `json:"autopilot_config"`
	CallbackURL       *string                `json:"callback_url"`
	SandboxStrategy   string                 `json:"sandbox_strategy"`
	SessionPersistence *bool                 `json:"session_persistence"`
	ConcurrencyPolicy string                 `json:"concurrency_policy"`
	MaxConcurrentRuns *int                   `json:"max_concurrent_runs"`
	MaxRetainedRuns   *int                   `json:"max_retained_runs"`
	TimeoutMinutes    *int                   `json:"timeout_minutes"`
}

type updateLoopRequest struct {
	Name              *string                `json:"name"`
	Description       *string                `json:"description"`
	AgentTypeID       *int64                 `json:"agent_type_id"`
	CustomAgentTypeID *int64                 `json:"custom_agent_type_id"`
	PermissionMode    *string                `json:"permission_mode"`
	PromptTemplate    *string                `json:"prompt_template"`
	PromptVariables   map[string]interface{} `json:"prompt_variables"`
	RepositoryID      *int64                 `json:"repository_id"`
	RunnerID          *int64                 `json:"runner_id"`
	BranchName        *string                `json:"branch_name"`
	TicketID          *int64                 `json:"ticket_id"`
	CredentialProfileID *int64               `json:"credential_profile_id"`
	ConfigOverrides   map[string]interface{} `json:"config_overrides"`
	ExecutionMode     *string                `json:"execution_mode"`
	CronExpression    *string                `json:"cron_expression"`
	AutopilotConfig   map[string]interface{} `json:"autopilot_config"`
	CallbackURL       *string                `json:"callback_url"`
	SandboxStrategy   *string                `json:"sandbox_strategy"`
	SessionPersistence *bool                 `json:"session_persistence"`
	ConcurrencyPolicy *string                `json:"concurrency_policy"`
	MaxConcurrentRuns *int                   `json:"max_concurrent_runs"`
	MaxRetainedRuns   *int                   `json:"max_retained_runs"`
	TimeoutMinutes    *int                   `json:"timeout_minutes"`
}

type listLoopsQuery struct {
	Status        string `form:"status"`
	ExecutionMode string `form:"execution_mode"`
	CronEnabled   *bool  `form:"cron_enabled"`
	Query         string `form:"query"`
	Limit         int    `form:"limit"`
	Offset        int    `form:"offset"`
}

type listRunsQuery struct {
	Status string `form:"status"`
	Limit  int    `form:"limit"`
	Offset int    `form:"offset"`
}

// ========== Loop CRUD ==========

// ListLoops lists loops for an organization
// GET /api/v1/orgs/:slug/loops
func (h *LoopHandler) ListLoops(c *gin.Context) {
	var req listLoopsQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)
	limit := req.Limit
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	loops, total, err := h.loopService.List(c.Request.Context(), &loopService.ListLoopsFilter{
		OrganizationID: tenant.OrganizationID,
		Status:         req.Status,
		ExecutionMode:  req.ExecutionMode,
		CronEnabled:    req.CronEnabled,
		Query:          req.Query,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		apierr.InternalError(c, "Failed to list loops")
		return
	}

	// Enrich with active run counts (H2)
	if len(loops) > 0 {
		loopIDs := make([]int64, len(loops))
		for i, l := range loops {
			loopIDs[i] = l.ID
		}
		if counts, err := h.loopRunService.CountActiveRunsByLoopIDs(c.Request.Context(), loopIDs); err == nil {
			for _, l := range loops {
				if count, ok := counts[l.ID]; ok {
					l.ActiveRunCount = int(count)
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"loops":  loops,
		"total":  total,
		"limit":  limit,
		"offset": req.Offset,
	})
}

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

	loop, err := h.loopService.Create(c.Request.Context(), &loopService.CreateLoopRequest{
		OrganizationID:      tenant.OrganizationID,
		CreatedByID:         tenant.UserID,
		Name:                req.Name,
		Slug:                req.Slug,
		Description:         req.Description,
		AgentTypeID:         req.AgentTypeID,
		CustomAgentTypeID:   req.CustomAgentTypeID,
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

// GetLoop gets a loop by slug
// GET /api/v1/orgs/:slug/loops/:loop_slug
func (h *LoopHandler) GetLoop(c *gin.Context) {
	tenant := middleware.GetTenant(c)
	loopSlug := c.Param("loop_slug")

	loop, err := h.loopService.GetBySlug(c.Request.Context(), tenant.OrganizationID, loopSlug)
	if err != nil {
		if errors.Is(err, loopService.ErrLoopNotFound) {
			apierr.ResourceNotFound(c, "Loop not found")
		} else {
			apierr.InternalError(c, "Failed to get loop")
		}
		return
	}

	// Enrich with active run count (H2)
	if counts, err := h.loopRunService.CountActiveRunsByLoopIDs(c.Request.Context(), []int64{loop.ID}); err == nil {
		if count, ok := counts[loop.ID]; ok {
			loop.ActiveRunCount = int(count)
		}
	}

	// Enrich with average duration (M5)
	if avg, err := h.loopRunService.GetAvgDuration(c.Request.Context(), loop.ID); err == nil && avg != nil {
		loop.AvgDurationSec = avg
	}

	c.JSON(http.StatusOK, gin.H{"loop": loop})
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

	svcReq := &loopService.UpdateLoopRequest{
		Name:                req.Name,
		Description:         req.Description,
		AgentTypeID:         req.AgentTypeID,
		CustomAgentTypeID:   req.CustomAgentTypeID,
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

// DeleteLoop deletes a loop.
// Atomically rejects deletion if there are active (pending/running) runs to prevent orphaned Pods.
// DELETE /api/v1/orgs/:slug/loops/:loop_slug
func (h *LoopHandler) DeleteLoop(c *gin.Context) {
	tenant := middleware.GetTenant(c)
	loopSlug := c.Param("loop_slug")

	if err := h.loopService.Delete(c.Request.Context(), tenant.OrganizationID, loopSlug); err != nil {
		if errors.Is(err, loopService.ErrLoopNotFound) {
			apierr.ResourceNotFound(c, "Loop not found")
		} else if errors.Is(err, loopService.ErrHasActiveRuns) {
			apierr.BadRequest(c, apierr.VALIDATION_FAILED,
				"Cannot delete loop with active runs. Cancel or wait for runs to complete first.")
		} else {
			apierr.InternalError(c, "Failed to delete loop")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Loop deleted"})
}

// EnableLoop enables a loop
// POST /api/v1/orgs/:slug/loops/:loop_slug/enable
func (h *LoopHandler) EnableLoop(c *gin.Context) {
	tenant := middleware.GetTenant(c)
	loopSlug := c.Param("loop_slug")

	loop, err := h.loopService.SetStatus(c.Request.Context(), tenant.OrganizationID, loopSlug, loopDomain.StatusEnabled)
	if err != nil {
		if errors.Is(err, loopService.ErrLoopNotFound) {
			apierr.ResourceNotFound(c, "Loop not found")
		} else {
			apierr.InternalError(c, "Failed to enable loop")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"loop": loop})
}

// DisableLoop disables a loop
// POST /api/v1/orgs/:slug/loops/:loop_slug/disable
func (h *LoopHandler) DisableLoop(c *gin.Context) {
	tenant := middleware.GetTenant(c)
	loopSlug := c.Param("loop_slug")

	loop, err := h.loopService.SetStatus(c.Request.Context(), tenant.OrganizationID, loopSlug, loopDomain.StatusDisabled)
	if err != nil {
		if errors.Is(err, loopService.ErrLoopNotFound) {
			apierr.ResourceNotFound(c, "Loop not found")
		} else {
			apierr.InternalError(c, "Failed to disable loop")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"loop": loop})
}

// TriggerLoop manually triggers a loop run
// POST /api/v1/orgs/:slug/loops/:loop_slug/trigger
func (h *LoopHandler) TriggerLoop(c *gin.Context) {
	tenant := middleware.GetTenant(c)
	loopSlug := c.Param("loop_slug")

	// Optional request body with trigger variables
	var body struct {
		Variables json.RawMessage `json:"variables"`
	}
	// Ignore binding errors — body is optional
	_ = c.ShouldBindJSON(&body)

	loop, err := h.loopService.GetBySlug(c.Request.Context(), tenant.OrganizationID, loopSlug)
	if err != nil {
		if errors.Is(err, loopService.ErrLoopNotFound) {
			apierr.ResourceNotFound(c, "Loop not found")
		} else {
			apierr.InternalError(c, "Failed to get loop")
		}
		return
	}

	result, err := h.orchestrator.TriggerRun(c.Request.Context(), &loopService.TriggerRunRequest{
		LoopID:        loop.ID,
		TriggerType:   loopDomain.RunTriggerManual,
		TriggerSource: "user:" + strconv.FormatInt(tenant.UserID, 10),
		TriggerParams: body.Variables,
	})
	if err != nil {
		if errors.Is(err, loopService.ErrLoopDisabled) {
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Loop is disabled")
		} else {
			apierr.InternalError(c, "Failed to trigger loop")
		}
		return
	}

	if result.Skipped {
		c.JSON(http.StatusOK, gin.H{
			"run":     result.Run,
			"skipped": true,
			"reason":  result.Reason,
		})
		return
	}

	// Start run asynchronously — orchestrator handles Pod creation + Autopilot setup.
	// Use result.Loop (refreshed within transaction) instead of stale `loop` variable.
	// Timeout prevents goroutine leak if Pod creation hangs indefinitely.
	startCtx, startCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	go func() {
		defer startCancel()
		h.orchestrator.StartRun(startCtx, result.Loop, result.Run, tenant.UserID)
	}()

	c.JSON(http.StatusCreated, gin.H{"run": result.Run})
}

// ========== Run Endpoints ==========

// ListRuns lists runs for a loop
// GET /api/v1/orgs/:slug/loops/:loop_slug/runs
func (h *LoopHandler) ListRuns(c *gin.Context) {
	var req listRunsQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)
	loopSlug := c.Param("loop_slug")

	loop, err := h.loopService.GetBySlug(c.Request.Context(), tenant.OrganizationID, loopSlug)
	if err != nil {
		if errors.Is(err, loopService.ErrLoopNotFound) {
			apierr.ResourceNotFound(c, "Loop not found")
		} else {
			apierr.InternalError(c, "Failed to get loop")
		}
		return
	}

	limit := req.Limit
	if limit == 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	runsOffset := req.Offset
	if runsOffset < 0 {
		runsOffset = 0
	}

	runs, total, err := h.loopRunService.ListRuns(c.Request.Context(), &loopService.ListRunsFilter{
		LoopID: loop.ID,
		Status: req.Status,
		Limit:  limit,
		Offset: runsOffset,
	})
	if err != nil {
		apierr.InternalError(c, "Failed to list runs")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"runs":   runs,
		"total":  total,
		"limit":  limit,
		"offset": runsOffset,
	})
}

// GetRun gets a run by ID
// GET /api/v1/orgs/:slug/loops/:loop_slug/runs/:run_id
func (h *LoopHandler) GetRun(c *gin.Context) {
	tenant := middleware.GetTenant(c)
	loopSlug := c.Param("loop_slug")
	runIDStr := c.Param("run_id")

	// Validate loop exists in this org
	loop, err := h.loopService.GetBySlug(c.Request.Context(), tenant.OrganizationID, loopSlug)
	if err != nil {
		if errors.Is(err, loopService.ErrLoopNotFound) {
			apierr.ResourceNotFound(c, "Loop not found")
		} else {
			apierr.InternalError(c, "Failed to get loop")
		}
		return
	}

	runID, err := strconv.ParseInt(runIDStr, 10, 64)
	if err != nil {
		apierr.ValidationError(c, "Invalid run ID")
		return
	}

	run, err := h.loopRunService.GetByID(c.Request.Context(), runID)
	if err != nil {
		if errors.Is(err, loopService.ErrRunNotFound) {
			apierr.ResourceNotFound(c, "Run not found")
		} else {
			apierr.InternalError(c, "Failed to get run")
		}
		return
	}

	// Verify the run belongs to this loop (prevents cross-loop access)
	if run.LoopID != loop.ID {
		apierr.ResourceNotFound(c, "Run not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"run": run})
}

// CancelRun cancels a running loop run
// POST /api/v1/orgs/:slug/loops/:loop_slug/runs/:run_id/cancel
func (h *LoopHandler) CancelRun(c *gin.Context) {
	tenant := middleware.GetTenant(c)
	loopSlug := c.Param("loop_slug")
	runIDStr := c.Param("run_id")

	// Validate loop exists in this org
	loop, err := h.loopService.GetBySlug(c.Request.Context(), tenant.OrganizationID, loopSlug)
	if err != nil {
		if errors.Is(err, loopService.ErrLoopNotFound) {
			apierr.ResourceNotFound(c, "Loop not found")
		} else {
			apierr.InternalError(c, "Failed to get loop")
		}
		return
	}

	runID, err := strconv.ParseInt(runIDStr, 10, 64)
	if err != nil {
		apierr.ValidationError(c, "Invalid run ID")
		return
	}

	run, err := h.loopRunService.GetByID(c.Request.Context(), runID)
	if err != nil {
		if errors.Is(err, loopService.ErrRunNotFound) {
			apierr.ResourceNotFound(c, "Run not found")
		} else {
			apierr.InternalError(c, "Failed to get run")
		}
		return
	}

	// Verify the run belongs to this loop (prevents cross-loop access)
	if run.LoopID != loop.ID {
		apierr.ResourceNotFound(c, "Run not found")
		return
	}

	if run.IsTerminal() {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Run is already in terminal state")
		return
	}

	// SSOT: cancel by terminating the Pod — run status will be derived from Pod state
	if run.PodKey != nil && h.podTerminator != nil {
		if err := h.podTerminator.TerminatePod(c.Request.Context(), *run.PodKey); err != nil {
			apierr.InternalError(c, "Failed to terminate pod")
			return
		}
	} else {
		// No Pod yet (still pending) — mark run as cancelled directly
		if err := h.orchestrator.MarkRunCancelled(c.Request.Context(), runID, "Cancelled by user"); err != nil {
			apierr.InternalError(c, "Failed to cancel run")
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Run cancelled"})
}
