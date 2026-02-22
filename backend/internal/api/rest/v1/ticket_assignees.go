package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ========== Assignee Management Endpoints ==========

// AddAssigneeRequest represents assignee addition request
type AddAssigneeRequest struct {
	UserID int64 `json:"user_id" binding:"required"`
}

// AddAssignee adds an assignee to a ticket
// POST /api/v1/organizations/:slug/tickets/:ticket_slug/assignees
func (h *TicketHandler) AddAssignee(c *gin.Context) {
	slug := c.Param("ticket_slug")

	var req AddAssigneeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)

	t, err := h.ticketService.GetTicketBySlug(c.Request.Context(), tenant.OrganizationID, slug)
	if err != nil {
		apierr.ResourceNotFound(c, "Ticket not found")
		return
	}

	if err := h.ticketService.AddAssignee(c.Request.Context(), t.ID, req.UserID); err != nil {
		apierr.InternalError(c, "Failed to add assignee")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Assignee added"})
}

// RemoveAssignee removes an assignee from a ticket
// DELETE /api/v1/organizations/:slug/tickets/:ticket_slug/assignees/:user_id
func (h *TicketHandler) RemoveAssignee(c *gin.Context) {
	slug := c.Param("ticket_slug")
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid user ID")
		return
	}

	tenant := middleware.GetTenant(c)

	t, err := h.ticketService.GetTicketBySlug(c.Request.Context(), tenant.OrganizationID, slug)
	if err != nil {
		apierr.ResourceNotFound(c, "Ticket not found")
		return
	}

	if err := h.ticketService.RemoveAssignee(c.Request.Context(), t.ID, userID); err != nil {
		apierr.InternalError(c, "Failed to remove assignee")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Assignee removed"})
}
