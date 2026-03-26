package v1

import (
	"errors"
	"net/http"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// CreatePodRequest represents pod creation request
type CreatePodRequest struct {
	RunnerID          int64   `json:"runner_id"`     // Required for new pods, optional when resuming (inherited from source)
	AgentTypeID       *int64  `json:"agent_type_id"` // Required unless resuming (then inherited from source pod)
	CustomAgentTypeID *int64  `json:"custom_agent_type_id"`
	RepositoryID      *int64  `json:"repository_id"`
	RepositoryURL     *string `json:"repository_url"`    // Direct repository URL (takes precedence over repository_id)
	TicketSlug        *string `json:"ticket_slug"`       // Ticket slug (e.g., "AM-123")
	InitialPrompt     string  `json:"initial_prompt"`
	Alias             *string `json:"alias"` // User-defined display name (max 100 chars)
	BranchName        *string `json:"branch_name"`
	PermissionMode    *string `json:"permission_mode"` // "plan", "default", or "bypassPermissions"

	// CredentialProfileID specifies which credential profile to use
	// - nil (field absent): use user's default profile, fallback to RunnerHost if no default
	// - 0: explicit RunnerHost mode (use Runner's local environment, no credentials injected)
	// - >0: use specified credential profile ID
	CredentialProfileID *int64 `json:"credential_profile_id"`

	// ConfigOverrides allows users to override agent type default configuration
	ConfigOverrides map[string]interface{} `json:"config_overrides"`

	// Terminal size (from browser xterm.js)
	Cols int32 `json:"cols"` // Terminal columns (width)
	Rows int32 `json:"rows"` // Terminal rows (height)

	// Resume related fields
	SourcePodKey       string `json:"source_pod_key"`       // Pod key to resume from (enables resume mode)
	ResumeAgentSession *bool  `json:"resume_agent_session"` // Whether to restore agent session (default: true when resuming)
}

// CreatePod creates a new pod
// POST /api/v1/organizations/:slug/pods
// Supports Resume mode when source_pod_key is provided
func (h *PodHandler) CreatePod(c *gin.Context) {
	var req CreatePodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)

	// Normalize alias: empty string → nil, validate length
	if req.Alias != nil {
		trimmed := strings.TrimSpace(*req.Alias)
		if trimmed == "" {
			req.Alias = nil
		} else if len(trimmed) > 100 {
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Alias must be 100 characters or less")
			return
		} else {
			req.Alias = &trimmed
		}
	}

	// Build orchestration request (protocol adaptation: HTTP → service layer)
	orchReq := &agentpod.OrchestrateCreatePodRequest{
		OrganizationID:      tenant.OrganizationID,
		UserID:              tenant.UserID,
		RunnerID:            req.RunnerID,
		AgentTypeID:         req.AgentTypeID,
		CustomAgentTypeID:   req.CustomAgentTypeID,
		RepositoryID:        req.RepositoryID,
		RepositoryURL:       req.RepositoryURL,
		TicketSlug:          req.TicketSlug,
		InitialPrompt:       req.InitialPrompt,
		Alias:               req.Alias,
		BranchName:          req.BranchName,
		PermissionMode:      req.PermissionMode,
		CredentialProfileID: req.CredentialProfileID,
		ConfigOverrides:     req.ConfigOverrides,
		Cols:                req.Cols,
		Rows:                req.Rows,
		SourcePodKey:        req.SourcePodKey,
		ResumeAgentSession:  req.ResumeAgentSession,
	}

	result, err := h.orchestrator.CreatePod(c.Request.Context(), orchReq)
	if err != nil {
		mapOrchestratorErrorToHTTP(c, err)
		return
	}

	// Return result with optional warning
	if result.Warning != "" {
		c.JSON(http.StatusCreated, gin.H{
			"pod":     result.Pod,
			"warning": result.Warning,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"pod": result.Pod})
}

// mapOrchestratorErrorToHTTP maps PodOrchestrator errors to HTTP responses.
func mapOrchestratorErrorToHTTP(c *gin.Context, err error) {
	switch {
	// Validation errors → 400
	case errors.Is(err, agentpod.ErrMissingRunnerID):
		apierr.BadRequest(c, apierr.MISSING_RUNNER_ID, err.Error())
	case errors.Is(err, agentpod.ErrMissingAgentTypeID):
		apierr.BadRequest(c, apierr.MISSING_AGENT_TYPE_ID, err.Error())
	case errors.Is(err, agentpod.ErrSourcePodNotTerminated):
		apierr.BadRequest(c, apierr.SOURCE_POD_NOT_TERMINATED, "Can only resume from terminated, completed, or orphaned pods")
	case errors.Is(err, agentpod.ErrResumeRunnerMismatch):
		apierr.BadRequest(c, apierr.RESUME_RUNNER_MISMATCH, "Resume requires same runner as source pod (Sandbox is local to runner)")

	// Billing errors → 402
	case errors.Is(err, ErrQuotaExceeded):
		apierr.PaymentRequired(c, apierr.CONCURRENT_POD_QUOTA_EXCEEDED, "Concurrent pod quota exceeded. Please upgrade your plan or terminate existing pods.")
	case errors.Is(err, ErrSubscriptionFrozen):
		apierr.PaymentRequired(c, apierr.SUBSCRIPTION_FROZEN, "Your subscription has expired. Please renew to continue.")

	// Access denied → 403
	case errors.Is(err, agentpod.ErrSourcePodAccessDenied):
		apierr.Forbidden(c, apierr.SOURCE_POD_ACCESS_DENIED, "Source pod belongs to different organization")

	// Not found → 404
	case errors.Is(err, agentpod.ErrSourcePodNotFound):
		apierr.NotFound(c, apierr.SOURCE_POD_NOT_FOUND, "Source pod not found for resume")

	// Conflict → 409
	case errors.Is(err, agentpod.ErrSourcePodAlreadyResumed):
		apierr.Conflict(c, apierr.SOURCE_POD_ALREADY_RESUMED, "Source pod has already been resumed by another active pod")
	case errors.Is(err, ErrSandboxAlreadyResumed):
		apierr.Conflict(c, apierr.SANDBOX_ALREADY_RESUMED, "Sandbox has already been resumed by another active pod")

	// No available runner → 503
	case errors.Is(err, agentpod.ErrNoAvailableRunner):
		apierr.ServiceUnavailable(c, apierr.NO_AVAILABLE_RUNNER, "No available runner supports the requested agent type")

	// Runner dispatch failure → 502
	case errors.Is(err, agentpod.ErrRunnerDispatchFailed):
		apierr.Respond(c, http.StatusBadGateway, apierr.RUNNER_DISPATCH_FAILED, "Failed to dispatch pod to runner. The runner may be offline or unreachable.")

	// Config build failure → 500
	case errors.Is(err, agentpod.ErrConfigBuildFailed):
		apierr.Respond(c, http.StatusInternalServerError, apierr.POD_CONFIG_BUILD_FAILED, "Failed to build pod configuration")

	// Fallback → 500
	default:
		apierr.InternalError(c, "Failed to create pod")
	}
}
