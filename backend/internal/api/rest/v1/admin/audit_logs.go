package admin

import (
	"net/http"
	"strconv"
	"time"

	domainadmin "github.com/AgentsMesh/AgentsMesh/backend/internal/domain/admin"
	adminservice "github.com/AgentsMesh/AgentsMesh/backend/internal/service/admin"

	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// AuditLogHandler handles audit log requests
type AuditLogHandler struct {
	adminService *adminservice.Service
}

// NewAuditLogHandler creates a new audit log handler
func NewAuditLogHandler(adminSvc *adminservice.Service) *AuditLogHandler {
	return &AuditLogHandler{
		adminService: adminSvc,
	}
}

// RegisterRoutes registers audit log routes
func (h *AuditLogHandler) RegisterRoutes(rg *gin.RouterGroup) {
	auditGroup := rg.Group("/audit-logs")
	{
		auditGroup.GET("", h.ListAuditLogs)
	}
}

// ListAuditLogs returns a list of audit logs with filtering and pagination
func (h *AuditLogHandler) ListAuditLogs(c *gin.Context) {
	query := &domainadmin.AuditLogQuery{
		Page:     1,
		PageSize: 20,
	}

	// Parse pagination
	if page, err := strconv.Atoi(c.Query("page")); err == nil {
		query.Page = page
	}
	if pageSize, err := strconv.Atoi(c.Query("page_size")); err == nil {
		query.PageSize = pageSize
	}

	// Parse filters
	if adminUserIDStr := c.Query("admin_user_id"); adminUserIDStr != "" {
		if id, err := strconv.ParseInt(adminUserIDStr, 10, 64); err == nil {
			query.AdminUserID = &id
		}
	}

	if action := c.Query("action"); action != "" {
		a := domainadmin.AuditAction(action)
		query.Action = &a
	}

	if targetType := c.Query("target_type"); targetType != "" {
		t := domainadmin.TargetType(targetType)
		query.TargetType = &t
	}

	if targetIDStr := c.Query("target_id"); targetIDStr != "" {
		if id, err := strconv.ParseInt(targetIDStr, 10, 64); err == nil {
			query.TargetID = &id
		}
	}

	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			query.StartTime = &t
		}
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			query.EndTime = &t
		}
	}

	result, err := h.adminService.GetAuditLogs(c.Request.Context(), query)
	if err != nil {
		apierr.InternalError(c, "Failed to get audit logs")
		return
	}

	// Convert to response format
	logs := make([]gin.H, len(result.Data))
	for i, log := range result.Data {
		logs[i] = auditLogResponse(&log)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        logs,
		"total":       result.Total,
		"page":        result.Page,
		"page_size":   result.PageSize,
		"total_pages": result.TotalPages,
	})
}

// auditLogResponse creates a sanitized audit log response
func auditLogResponse(log *domainadmin.AuditLog) gin.H {
	response := gin.H{
		"id":            log.ID,
		"admin_user_id": log.AdminUserID,
		"action":        log.Action,
		"target_type":   log.TargetType,
		"target_id":     log.TargetID,
		"ip_address":    log.IPAddress,
		"user_agent":    log.UserAgent,
		"created_at":    log.CreatedAt,
	}

	if log.OldData != nil {
		response["old_data"] = log.OldData
	}
	if log.NewData != nil {
		response["new_data"] = log.NewData
	}

	if log.AdminUser != nil {
		response["admin_user"] = gin.H{
			"id":         log.AdminUser.ID,
			"email":      log.AdminUser.Email,
			"username":   log.AdminUser.Username,
			"name":       log.AdminUser.Name,
			"avatar_url": log.AdminUser.AvatarURL,
		}
	}

	return response
}
