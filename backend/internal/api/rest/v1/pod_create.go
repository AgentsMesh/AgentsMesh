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

// AgentfileLayer is SSOT for pod config (MODE/CONFIG/REPO/BRANCH/CREDENTIAL/PROMPT).
type CreatePodRequest struct {
	AgentSlug    string  `json:"agent_slug"`    // determines base AgentFile
	RunnerID     int64   `json:"runner_id"`     // auto-select if omitted
	TicketSlug   *string `json:"ticket_slug"`
	Alias        *string `json:"alias"`         // max 100 chars

	AgentfileLayer *string `json:"agentfile_layer"`

	RepositoryID        *int64 `json:"repository_id,omitempty"`
	CredentialProfileID *int64 `json:"credential_profile_id,omitempty"`

	Cols int32 `json:"cols"`
	Rows int32 `json:"rows"`

	SourcePodKey       string `json:"source_pod_key"`
	ResumeAgentSession *bool  `json:"resume_agent_session"`

	Perpetual *bool `json:"perpetual"`
}

func (h *PodHandler) CreatePod(c *gin.Context) {
	var req CreatePodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)

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

	orchReq := &agentpod.OrchestrateCreatePodRequest{
		OrganizationID:      tenant.OrganizationID,
		UserID:              tenant.UserID,
		RunnerID:            req.RunnerID,
		AgentSlug:           req.AgentSlug,
		RepositoryID:        req.RepositoryID,
		TicketSlug:          req.TicketSlug,
		Alias:               req.Alias,
		CredentialProfileID: req.CredentialProfileID,
		AgentfileLayer:      req.AgentfileLayer,
		Cols:                req.Cols,
		Rows:                req.Rows,
		SourcePodKey:        req.SourcePodKey,
		ResumeAgentSession:  req.ResumeAgentSession,
		Perpetual:           req.Perpetual != nil && *req.Perpetual,
	}

	result, err := h.orchestrator.CreatePod(c.Request.Context(), orchReq)
	if err != nil {
		mapOrchestratorErrorToHTTP(c, err)
		return
	}

	if result.Warning != "" {
		c.JSON(http.StatusCreated, gin.H{
			"pod":     result.Pod,
			"warning": result.Warning,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"pod": result.Pod})
}

func mapOrchestratorErrorToHTTP(c *gin.Context, err error) {
	switch {
	case errors.Is(err, agentpod.ErrMissingRunnerID):
		apierr.BadRequest(c, apierr.MISSING_RUNNER_ID, err.Error())
	case errors.Is(err, agentpod.ErrMissingAgentSlug):
		apierr.BadRequest(c, apierr.MISSING_AGENT_SLUG, err.Error())
	case errors.Is(err, agentpod.ErrSourcePodNotTerminated):
		apierr.BadRequest(c, apierr.SOURCE_POD_NOT_TERMINATED, "Can only resume from terminated, completed, or orphaned pods")
	case errors.Is(err, agentpod.ErrResumeRunnerMismatch):
		apierr.BadRequest(c, apierr.RESUME_RUNNER_MISMATCH, "Resume requires same runner as source pod (Sandbox is local to runner)")
	case errors.Is(err, agentpod.ErrUnsupportedInteractionMode):
		apierr.BadRequest(c, apierr.UNSUPPORTED_INTERACTION_MODE, err.Error())
	case errors.Is(err, agentpod.ErrInvalidAgentfileLayer):
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, err.Error())

	case errors.Is(err, ErrQuotaExceeded):
		apierr.PaymentRequired(c, apierr.CONCURRENT_POD_QUOTA_EXCEEDED, "Concurrent pod quota exceeded. Please upgrade your plan or terminate existing pods.")
	case errors.Is(err, ErrSubscriptionFrozen):
		apierr.PaymentRequired(c, apierr.SUBSCRIPTION_FROZEN, "Your subscription has expired. Please renew to continue.")

	case errors.Is(err, agentpod.ErrSourcePodAccessDenied):
		apierr.Forbidden(c, apierr.SOURCE_POD_ACCESS_DENIED, "Source pod belongs to different organization")

	case errors.Is(err, agentpod.ErrSourcePodNotFound):
		apierr.NotFound(c, apierr.SOURCE_POD_NOT_FOUND, "Source pod not found for resume")

	case errors.Is(err, agentpod.ErrSourcePodAlreadyResumed):
		apierr.Conflict(c, apierr.SOURCE_POD_ALREADY_RESUMED, "Source pod has already been resumed by another active pod")
	case errors.Is(err, ErrSandboxAlreadyResumed):
		apierr.Conflict(c, apierr.SANDBOX_ALREADY_RESUMED, "Sandbox has already been resumed by another active pod")

	case errors.Is(err, agentpod.ErrNoAvailableRunner):
		apierr.ServiceUnavailable(c, apierr.NO_AVAILABLE_RUNNER, "No available runner supports the requested agent")

	case errors.Is(err, agentpod.ErrRunnerDispatchFailed):
		apierr.Respond(c, http.StatusBadGateway, apierr.RUNNER_DISPATCH_FAILED, "Failed to dispatch pod to runner. The runner may be offline or unreachable.")

	case errors.Is(err, agentpod.ErrConfigBuildFailed):
		apierr.Respond(c, http.StatusInternalServerError, apierr.POD_CONFIG_BUILD_FAILED, "Failed to build pod configuration")

	default:
		apierr.InternalError(c, "Failed to create pod")
	}
}
