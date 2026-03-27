package admin

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	adminservice "github.com/anthropics/agentsmesh/backend/internal/service/admin"
	ssoservice "github.com/anthropics/agentsmesh/backend/internal/service/sso"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// SSOHandler handles admin SSO configuration management
type SSOHandler struct {
	ssoService   *ssoservice.Service
	adminService *adminservice.Service
}

// NewSSOHandler creates a new admin SSO handler
func NewSSOHandler(ssoSvc *ssoservice.Service, adminSvc *adminservice.Service) *SSOHandler {
	return &SSOHandler{
		ssoService:   ssoSvc,
		adminService: adminSvc,
	}
}

// RegisterRoutes registers admin SSO management routes
func (h *SSOHandler) RegisterRoutes(rg *gin.RouterGroup) {
	ssoGroup := rg.Group("/sso/configs")
	{
		ssoGroup.GET("", h.ListConfigs)
		ssoGroup.POST("", h.CreateConfig)
		ssoGroup.GET("/:id", h.GetConfig)
		ssoGroup.PUT("/:id", h.UpdateConfig)
		ssoGroup.DELETE("/:id", h.DeleteConfig)
		ssoGroup.POST("/:id/test", h.TestConnection)
		ssoGroup.POST("/:id/enable", h.EnableConfig)
		ssoGroup.POST("/:id/disable", h.DisableConfig)
	}
}

// logAction is a helper method for audit logging
func (h *SSOHandler) logAction(c *gin.Context, action admin.AuditAction, targetType admin.TargetType, targetID int64, oldData, newData interface{}) {
	LogAdminAction(c, h.adminService, action, targetType, targetID, oldData, newData)
}

// ListConfigs returns all SSO configurations with pagination
func (h *SSOHandler) ListConfigs(c *gin.Context) {
	page := 1
	pageSize := 20

	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}
	if ps, err := strconv.Atoi(c.Query("page_size")); err == nil && ps > 0 {
		pageSize = ps
	}

	var query *sso.ListQuery
	search := c.Query("search")
	protocol := c.Query("protocol")
	if search != "" || protocol != "" {
		query = &sso.ListQuery{
			Search:   search,
			Protocol: sso.Protocol(protocol),
		}
	}

	configs, total, err := h.ssoService.ListConfigs(c.Request.Context(), query, page, pageSize)
	if err != nil {
		apierr.InternalError(c, "Failed to list SSO configs")
		return
	}

	result := make([]*ssoservice.ConfigResponse, 0, len(configs))
	for _, cfg := range configs {
		result = append(result, h.ssoService.ToConfigResponse(cfg))
	}

	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(pageSize) - 1) / int64(pageSize)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        result,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
}

// CreateConfig creates a new SSO configuration
func (h *SSOHandler) CreateConfig(c *gin.Context) {
	var req ssoservice.CreateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	// Get admin user ID from context
	userID := getUserIDFromContext(c)

	cfg, err := h.ssoService.CreateConfig(c.Request.Context(), &req, userID)
	if err != nil {
		switch {
		case errors.Is(err, ssoservice.ErrDuplicateConfig):
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "SSO config already exists for this domain and protocol")
		case errors.Is(err, ssoservice.ErrInvalidProtocol):
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Invalid protocol (must be oidc, saml, or ldap)")
		default:
			apierr.InternalError(c, "Failed to create SSO config")
		}
		return
	}

	h.logAction(c, admin.AuditActionCreate, admin.TargetTypeSSOConfig, cfg.ID, nil, cfg)

	c.JSON(http.StatusCreated, h.ssoService.ToConfigResponse(cfg))
}

// GetConfig returns a single SSO configuration
func (h *SSOHandler) GetConfig(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid config ID")
		return
	}

	cfg, err := h.ssoService.GetConfig(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ssoservice.ErrConfigNotFound) {
			apierr.ResourceNotFound(c, "SSO config not found")
			return
		}
		apierr.InternalError(c, "Failed to get SSO config")
		return
	}

	c.JSON(http.StatusOK, h.ssoService.ToConfigResponse(cfg))
}

// UpdateConfig updates an SSO configuration
func (h *SSOHandler) UpdateConfig(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid config ID")
		return
	}

	var req ssoservice.UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	// Get old data for audit log
	oldCfg, auditErr := h.ssoService.GetConfig(c.Request.Context(), id)
	if auditErr != nil {
		slog.Warn("failed to retrieve old SSO config for audit", "id", id, "error", auditErr)
	}

	cfg, err := h.ssoService.UpdateConfig(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, ssoservice.ErrConfigNotFound) {
			apierr.ResourceNotFound(c, "SSO config not found")
			return
		}
		var validationErr *ssoservice.ValidationError
		if errors.As(err, &validationErr) {
			apierr.ValidationError(c, validationErr.Message)
			return
		}
		apierr.InternalError(c, "Failed to update SSO config")
		return
	}

	h.logAction(c, admin.AuditActionUpdate, admin.TargetTypeSSOConfig, id, oldCfg, cfg)

	c.JSON(http.StatusOK, h.ssoService.ToConfigResponse(cfg))
}

// DeleteConfig deletes an SSO configuration
func (h *SSOHandler) DeleteConfig(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid config ID")
		return
	}

	// Get old data for audit log
	oldCfg, auditErr := h.ssoService.GetConfig(c.Request.Context(), id)
	if auditErr != nil {
		slog.Warn("failed to retrieve old SSO config for audit", "id", id, "error", auditErr)
	}

	if err := h.ssoService.DeleteConfig(c.Request.Context(), id); err != nil {
		if errors.Is(err, ssoservice.ErrConfigNotFound) {
			apierr.ResourceNotFound(c, "SSO config not found")
			return
		}
		apierr.InternalError(c, "Failed to delete SSO config")
		return
	}

	h.logAction(c, admin.AuditActionDelete, admin.TargetTypeSSOConfig, id, oldCfg, nil)

	c.JSON(http.StatusOK, gin.H{"message": "SSO config deleted"})
}

func boolPtr(b bool) *bool { return &b }

// getUserIDFromContext extracts user ID from gin context (set by auth middleware)
func getUserIDFromContext(c *gin.Context) int64 {
	if id, exists := c.Get("user_id"); exists {
		if userID, ok := id.(int64); ok {
			return userID
		}
		// Try float64 (JSON numbers are float64 by default)
		if userID, ok := id.(float64); ok {
			return int64(userID)
		}
	}
	return 0
}
