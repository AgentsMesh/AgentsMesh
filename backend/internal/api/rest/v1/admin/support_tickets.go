package admin

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	adminservice "github.com/anthropics/agentsmesh/backend/internal/service/admin"
	"github.com/anthropics/agentsmesh/backend/internal/service/supportticket"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// SupportTicketHandler handles admin support ticket management
type SupportTicketHandler struct {
	service      *supportticket.Service
	adminService *adminservice.Service
}

// NewSupportTicketHandler creates a new admin support ticket handler
func NewSupportTicketHandler(service *supportticket.Service, adminSvc *adminservice.Service) *SupportTicketHandler {
	return &SupportTicketHandler{
		service:      service,
		adminService: adminSvc,
	}
}

// RegisterRoutes registers admin support ticket routes
func (h *SupportTicketHandler) RegisterRoutes(rg *gin.RouterGroup) {
	group := rg.Group("/support-tickets")
	{
		group.GET("", h.List)
		group.GET("/stats", h.GetStats)
		group.GET("/:id", h.GetByID)
		group.GET("/:id/messages", h.ListMessages)
		group.POST("/:id/reply", h.Reply)
		group.PATCH("/:id/status", h.UpdateStatus)
		group.POST("/:id/assign", h.Assign)
		group.GET("/attachments/:attachmentId/url", h.GetAttachmentURL)
	}
}

// logAction is a helper method that delegates to the shared LogAdminAction function
func (h *SupportTicketHandler) logAction(c *gin.Context, action admin.AuditAction, targetType admin.TargetType, targetID int64, oldData, newData interface{}) {
	LogAdminAction(c, h.adminService, action, targetType, targetID, oldData, newData)
}

// List returns all support tickets with filtering and pagination
// GET /api/v1/admin/support-tickets
func (h *SupportTicketHandler) List(c *gin.Context) {
	query := &supportticket.AdminListQuery{
		Search:   c.Query("search"),
		Status:   c.Query("status"),
		Category: c.Query("category"),
		Priority: c.Query("priority"),
		Page:     1,
		PageSize: 20,
	}

	if page, err := strconv.Atoi(c.Query("page")); err == nil {
		query.Page = page
	}
	if pageSize, err := strconv.Atoi(c.Query("page_size")); err == nil {
		query.PageSize = pageSize
	}

	result, err := h.service.AdminList(c.Request.Context(), query)
	if err != nil {
		apierr.InternalError(c, "Failed to list support tickets")
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetStats returns support ticket statistics
// GET /api/v1/admin/support-tickets/stats
func (h *SupportTicketHandler) GetStats(c *gin.Context) {
	stats, err := h.service.AdminGetStats(c.Request.Context())
	if err != nil {
		apierr.InternalError(c, "Failed to get support ticket stats")
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetByID returns a single support ticket with messages
// GET /api/v1/admin/support-tickets/:id
func (h *SupportTicketHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid ticket ID")
		return
	}

	ticket, err := h.service.AdminGetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, supportticket.ErrTicketNotFound) {
			apierr.ResourceNotFound(c, "Support ticket not found")
			return
		}
		apierr.InternalError(c, "Failed to get support ticket")
		return
	}

	messages, err := h.service.AdminListMessages(c.Request.Context(), id)
	if err != nil {
		slog.Warn("failed to load messages for ticket", "ticket_id", id, "error", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"ticket":   ticket,
		"messages": messages,
	})
}

// ListMessages returns all messages for a support ticket
// GET /api/v1/admin/support-tickets/:id/messages
func (h *SupportTicketHandler) ListMessages(c *gin.Context) {
	ticketID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid ticket ID")
		return
	}

	messages, err := h.service.AdminListMessages(c.Request.Context(), ticketID)
	if err != nil {
		apierr.InternalError(c, "Failed to list messages")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": messages})
}
