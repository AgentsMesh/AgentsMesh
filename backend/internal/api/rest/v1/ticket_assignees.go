package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// ========== Assignee Management Endpoints ==========

// AddAssigneeRequest represents assignee addition request
type AddAssigneeRequest struct {
	UserID int64 `json:"user_id" binding:"required"`
}

// AddAssignee adds an assignee to a ticket
// POST /api/v1/organizations/:slug/tickets/:identifier/assignees
func (h *TicketHandler) AddAssignee(c *gin.Context) {
	identifier := c.Param("identifier")

	var req AddAssigneeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	t, err := h.ticketService.GetTicketByIdentifier(c.Request.Context(), identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if t.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.ticketService.AddAssignee(c.Request.Context(), t.ID, req.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add assignee"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Assignee added"})
}

// RemoveAssignee removes an assignee from a ticket
// DELETE /api/v1/organizations/:slug/tickets/:identifier/assignees/:user_id
func (h *TicketHandler) RemoveAssignee(c *gin.Context) {
	identifier := c.Param("identifier")
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	t, err := h.ticketService.GetTicketByIdentifier(c.Request.Context(), identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	tenant := middleware.GetTenant(c)
	if t.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.ticketService.RemoveAssignee(c.Request.Context(), t.ID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove assignee"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Assignee removed"})
}
