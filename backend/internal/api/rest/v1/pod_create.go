package v1

import (
	"errors"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	"github.com/gin-gonic/gin"
)

// CreatePodRequest represents pod creation request
type CreatePodRequest struct {
	RunnerID          int64   `json:"runner_id"`           // Required for new pods, optional when resuming (inherited from source)
	AgentTypeID       *int64  `json:"agent_type_id"`       // Required unless resuming (then inherited from source pod)
	CustomAgentTypeID *int64  `json:"custom_agent_type_id"`
	RepositoryID      *int64  `json:"repository_id"`
	RepositoryURL     *string `json:"repository_url"`    // Direct repository URL (takes precedence over repository_id)
	TicketID          *int64  `json:"ticket_id"`
	TicketIdentifier  *string `json:"ticket_identifier"` // Direct ticket identifier (takes precedence over ticket_id)
	InitialPrompt     string  `json:"initial_prompt"`
	BranchName        *string `json:"branch_name"`
	PermissionMode    *string `json:"permission_mode"` // "plan", "default", or "bypassPermissions"

	// CredentialProfileID specifies which credential profile to use
	// - nil or 0: RunnerHost mode (use Runner's local environment, no credentials injected)
	// - >0: Use specified credential profile ID
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)

	// Build orchestration request (protocol adaptation: HTTP → service layer)
	orchReq := &agentpod.OrchestrateCreatePodRequest{
		OrganizationID:      tenant.OrganizationID,
		UserID:              tenant.UserID,
		RunnerID:            req.RunnerID,
		AgentTypeID:         req.AgentTypeID,
		CustomAgentTypeID:   req.CustomAgentTypeID,
		RepositoryID:        req.RepositoryID,
		RepositoryURL:       req.RepositoryURL,
		TicketID:            req.TicketID,
		TicketIdentifier:    req.TicketIdentifier,
		InitialPrompt:       req.InitialPrompt,
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "MISSING_RUNNER_ID"})
	case errors.Is(err, agentpod.ErrMissingAgentTypeID):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "MISSING_AGENT_TYPE_ID"})
	case errors.Is(err, agentpod.ErrSourcePodNotTerminated):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Can only resume from terminated, completed, or orphaned pods", "code": "SOURCE_POD_NOT_TERMINATED"})
	case errors.Is(err, agentpod.ErrResumeRunnerMismatch):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Resume requires same runner as source pod (Sandbox is local to runner)", "code": "RESUME_RUNNER_MISMATCH"})

	// Billing errors → 402
	case errors.Is(err, ErrQuotaExceeded):
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Concurrent pod quota exceeded. Please upgrade your plan or terminate existing pods.", "code": "CONCURRENT_POD_QUOTA_EXCEEDED"})
	case errors.Is(err, ErrSubscriptionFrozen):
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Your subscription has expired. Please renew to continue.", "code": "SUBSCRIPTION_FROZEN"})

	// Access denied → 403
	case errors.Is(err, agentpod.ErrSourcePodAccessDenied):
		c.JSON(http.StatusForbidden, gin.H{"error": "Source pod belongs to different organization", "code": "SOURCE_POD_ACCESS_DENIED"})

	// Not found → 404
	case errors.Is(err, agentpod.ErrSourcePodNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Source pod not found for resume", "code": "SOURCE_POD_NOT_FOUND"})

	// Conflict → 409
	case errors.Is(err, agentpod.ErrSourcePodAlreadyResumed):
		c.JSON(http.StatusConflict, gin.H{"error": "Source pod has already been resumed by another active pod", "code": "SOURCE_POD_ALREADY_RESUMED"})
	case errors.Is(err, ErrSandboxAlreadyResumed):
		c.JSON(http.StatusConflict, gin.H{"error": "Sandbox has already been resumed by another active pod", "code": "SANDBOX_ALREADY_RESUMED"})

	// Config build failure → 500
	case errors.Is(err, agentpod.ErrConfigBuildFailed):
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build pod configuration", "code": "POD_CONFIG_BUILD_FAILED"})

	// Fallback → 500
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pod"})
	}
}
