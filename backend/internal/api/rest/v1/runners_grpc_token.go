package v1

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

func (h *GRPCRunnerHandler) GenerateGRPCToken(c *gin.Context) {
	var req GenerateGRPCTokenRequest
	_ = c.ShouldBindJSON(&req)

	tenant := middleware.GetTenant(c)
	if tenant == nil {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "Unauthorized")
		return
	}

	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		apierr.ForbiddenAdmin(c)
		return
	}

	serverURL := h.config.BaseURL()

	resp, err := h.runnerService.GenerateGRPCRegistrationToken(
		c.Request.Context(),
		tenant.OrganizationID,
		tenant.UserID,
		&runner.GenerateGRPCRegistrationTokenRequest{
			Name:      req.Name,
			Labels:    req.Labels,
			SingleUse: req.SingleUse,
			MaxUses:   req.MaxUses,
			ExpiresIn: req.ExpiresIn,
		},
		serverURL,
	)
	if err != nil {
		apierr.InternalError(c, "Failed to generate token")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         resp.ID,
		"token":      resp.Token,
		"expires_at": resp.ExpiresAt,
		"command":    resp.Command,
		"message":    "Save this token securely. It will only be shown once.",
	})
}

func (h *GRPCRunnerHandler) ListGRPCTokens(c *gin.Context) {
	tenant := middleware.GetTenant(c)
	if tenant == nil {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "Unauthorized")
		return
	}

	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		apierr.ForbiddenAdmin(c)
		return
	}

	tokens, err := h.runnerService.ListGRPCRegistrationTokens(c.Request.Context(), tenant.OrganizationID)
	if err != nil {
		apierr.InternalError(c, "Failed to list tokens")
		return
	}

	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}

func (h *GRPCRunnerHandler) DeleteGRPCToken(c *gin.Context) {
	tokenID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid token ID")
		return
	}

	tenant := middleware.GetTenant(c)
	if tenant == nil {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "Unauthorized")
		return
	}

	if tenant.UserRole != "owner" && tenant.UserRole != "admin" {
		apierr.ForbiddenAdmin(c)
		return
	}

	if err := h.runnerService.DeleteGRPCRegistrationToken(c.Request.Context(), tokenID, tenant.OrganizationID); err != nil {
		if errors.Is(err, runner.ErrGRPCTokenNotFound) {
			apierr.ResourceNotFound(c, "Token not found")
			return
		}
		apierr.InternalError(c, "Failed to delete token")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Token deleted"})
}

func (h *GRPCRunnerHandler) RegisterWithToken(c *gin.Context) {
	if h.pkiService == nil {
		apierr.ServiceUnavailable(c, apierr.SERVICE_UNAVAILABLE, "PKI service not configured")
		return
	}

	var req RegisterWithTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	resp, err := h.runnerService.RegisterWithToken(
		c.Request.Context(),
		&runner.RegisterWithTokenRequest{
			Token:  req.Token,
			NodeID: req.NodeID,
		},
		h.pkiService,
	)
	if err != nil {
		switch {
		case errors.Is(err, runner.ErrInvalidToken):
			apierr.Unauthorized(c, apierr.INVALID_TOKEN, "Invalid token")
		case errors.Is(err, runner.ErrTokenExpired):
			apierr.Unauthorized(c, apierr.INVALID_TOKEN, "Token expired")
		case errors.Is(err, runner.ErrTokenExhausted):
			apierr.Unauthorized(c, apierr.INVALID_TOKEN, "Token usage exhausted")
		case errors.Is(err, runner.ErrRunnerAlreadyExists):
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "Runner with this node_id already exists")
		case errors.Is(err, runner.ErrRunnerQuotaExceeded):
			apierr.PaymentRequired(c, apierr.RUNNER_QUOTA_EXCEEDED, "Runner quota exceeded")
		default:
			apierr.InternalError(c, "Failed to register runner")
		}
		return
	}

	resp.GRPCEndpoint = h.config.GRPC.Endpoint
	c.JSON(http.StatusCreated, resp)
}
