package v1

import (
	"errors"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ==================== Interactive Registration (Tailscale-style) ====================

// RequestAuthURL creates a pending auth request and returns an authorization URL.
// POST /api/v1/runners/grpc/auth-url
// No authentication required - Runner initiates registration.
func (h *GRPCRunnerHandler) RequestAuthURL(c *gin.Context) {
	var req RequestAuthURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	// FrontendURL is derived from PrimaryDomain
	frontendURL := h.config.FrontendURL()

	resp, err := h.runnerService.RequestAuthURL(c.Request.Context(), &runner.RequestAuthURLRequest{
		MachineKey: req.MachineKey,
		NodeID:     req.NodeID,
		Labels:     req.Labels,
	}, frontendURL)
	if err != nil {
		apierr.InternalError(c, "Failed to create auth request")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"auth_url":   resp.AuthURL,
		"auth_key":   resp.AuthKey,
		"expires_in": resp.ExpiresIn,
	})
}

// GetAuthStatus returns the current status of a pending authorization.
// GET /api/v1/runners/grpc/auth-status?key=xxx
// No authentication required - Runner polls for completion.
func (h *GRPCRunnerHandler) GetAuthStatus(c *gin.Context) {
	authKey := c.Query("key")
	if authKey == "" {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Missing auth key")
		return
	}

	resp, err := h.runnerService.GetAuthStatus(c.Request.Context(), authKey, h.pkiService)
	if err != nil {
		if errors.Is(err, runner.ErrAuthRequestNotFound) {
			apierr.ResourceNotFound(c, "Auth request not found")
		} else {
			apierr.InternalError(c, "Failed to get auth status")
		}
		return
	}

	// Inject gRPC endpoint for authorized responses
	if resp.Status == "authorized" {
		resp.GRPCEndpoint = h.config.GRPC.Endpoint
	}

	c.JSON(http.StatusOK, resp)
}

// AuthorizeRunner authorizes a pending auth request.
// POST /api/v1/organizations/:slug/runners/grpc/authorize
// Requires JWT authentication.
func (h *GRPCRunnerHandler) AuthorizeRunner(c *gin.Context) {
	var req AuthorizeRunnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)
	if tenant == nil {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "Unauthorized")
		return
	}

	// Check admin permission
	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		apierr.ForbiddenAdmin(c)
		return
	}

	r, err := h.runnerService.AuthorizeRunner(c.Request.Context(), req.AuthKey, tenant.OrganizationID, tenant.UserID, req.NodeID)
	if err != nil {
		switch {
		case errors.Is(err, runner.ErrRunnerAlreadyExists):
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Runner with this node_id already exists")
		case errors.Is(err, runner.ErrRunnerQuotaExceeded):
			apierr.PaymentRequired(c, apierr.RUNNER_QUOTA_EXCEEDED, "Runner quota exceeded")
		case errors.Is(err, runner.ErrAuthRequestNotFound):
			apierr.ResourceNotFound(c, "Auth request not found")
		case errors.Is(err, runner.ErrAuthRequestExpired):
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Auth request expired")
		case errors.Is(err, runner.ErrAuthRequestAlreadyAuthorized):
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Auth request already authorized")
		default:
			apierr.InternalError(c, "Failed to authorize runner")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"runner_id": r.ID,
		"node_id":   r.NodeID,
		"message":   "Runner authorized successfully",
	})
}
