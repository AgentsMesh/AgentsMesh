package admin

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	ssoservice "github.com/anthropics/agentsmesh/backend/internal/service/sso"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// TestConnection tests connectivity to the SSO provider
func (h *SSOHandler) TestConnection(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid config ID")
		return
	}

	if err := h.ssoService.TestConnection(c.Request.Context(), id); err != nil {
		// Log full error server-side; return sanitized message to client
		slog.Warn("SSO test connection failed", "id", id, "error", err)
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "Connection test failed. Check server logs for details.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Connection successful",
	})
}

// EnableConfig enables an SSO configuration
func (h *SSOHandler) EnableConfig(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid config ID")
		return
	}

	oldCfg, auditErr := h.ssoService.GetConfig(c.Request.Context(), id)
	if auditErr != nil {
		slog.Warn("failed to retrieve old SSO config for audit", "id", id, "error", auditErr)
	}

	req := ssoservice.UpdateConfigRequest{IsEnabled: boolPtr(true)}
	cfg, err := h.ssoService.UpdateConfig(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, ssoservice.ErrConfigNotFound) {
			apierr.ResourceNotFound(c, "SSO config not found")
			return
		}
		apierr.InternalError(c, "Failed to enable SSO config")
		return
	}

	h.logAction(c, admin.AuditActionActivate, admin.TargetTypeSSOConfig, id, oldCfg, cfg)

	c.JSON(http.StatusOK, h.ssoService.ToConfigResponse(cfg))
}

// DisableConfig disables an SSO configuration
func (h *SSOHandler) DisableConfig(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid config ID")
		return
	}

	oldCfg, auditErr := h.ssoService.GetConfig(c.Request.Context(), id)
	if auditErr != nil {
		slog.Warn("failed to retrieve old SSO config for audit", "id", id, "error", auditErr)
	}

	req := ssoservice.UpdateConfigRequest{IsEnabled: boolPtr(false)}
	cfg, err := h.ssoService.UpdateConfig(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, ssoservice.ErrConfigNotFound) {
			apierr.ResourceNotFound(c, "SSO config not found")
			return
		}
		apierr.InternalError(c, "Failed to disable SSO config")
		return
	}

	h.logAction(c, admin.AuditActionDeactivate, admin.TargetTypeSSOConfig, id, oldCfg, cfg)

	c.JSON(http.StatusOK, h.ssoService.ToConfigResponse(cfg))
}
