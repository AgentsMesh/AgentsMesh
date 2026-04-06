package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	ticketDomain "github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ========== Board and View Endpoints ==========

// GetActiveTickets returns active (non-completed) tickets
// GET /api/v1/organizations/:slug/tickets/active
func (h *TicketHandler) GetActiveTickets(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	var repoID *int64
	if repoIDStr := c.Query("repository_id"); repoIDStr != "" {
		if id, err := strconv.ParseInt(repoIDStr, 10, 64); err == nil {
			repoID = &id
		}
	}

	tickets, err := h.ticketService.GetActiveTickets(c.Request.Context(), tenant.OrganizationID, repoID, limit)
	if err != nil {
		apierr.InternalError(c, "Failed to get active tickets")
		return
	}

	c.JSON(http.StatusOK, gin.H{"tickets": tickets})
}

// GetBoard returns the kanban board view
// GET /api/v1/organizations/:slug/tickets/board
func (h *TicketHandler) GetBoard(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	var repoID *int64
	if repoIDStr := c.Query("repository_id"); repoIDStr != "" {
		if id, err := strconv.ParseInt(repoIDStr, 10, 64); err == nil {
			repoID = &id
		}
	}

	board, err := h.ticketService.GetBoard(c.Request.Context(), &ticketDomain.TicketListFilter{
		OrganizationID: tenant.OrganizationID,
		RepositoryID:   repoID,
		UserRole:       tenant.UserRole,
		Limit:          50,
	})
	if err != nil {
		apierr.InternalError(c, "Failed to get board")
		return
	}

	c.JSON(http.StatusOK, gin.H{"board": board})
}

// GetSubTickets returns sub-tickets of a parent ticket
// GET /api/v1/organizations/:slug/tickets/:ticket_slug/sub-tickets
func (h *TicketHandler) GetSubTickets(c *gin.Context) {
	slug := c.Param("ticket_slug")

	tenant := middleware.GetTenant(c)

	t, err := h.ticketService.GetTicketBySlug(c.Request.Context(), tenant.OrganizationID, slug)
	if err != nil {
		apierr.ResourceNotFound(c, "Ticket not found")
		return
	}

	subTickets, err := h.ticketService.GetChildTickets(c.Request.Context(), t.ID)
	if err != nil {
		apierr.InternalError(c, "Failed to get sub-tickets")
		return
	}

	c.JSON(http.StatusOK, gin.H{"sub_tickets": subTickets})
}
