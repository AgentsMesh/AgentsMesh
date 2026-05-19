package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	ticketDomain "github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

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

func (h *TicketHandler) GetBoard(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	filter := &ticketDomain.TicketListFilter{
		OrganizationID: tenant.OrganizationID,
		UserRole:       tenant.UserRole,
		Limit:          50,
	}

	if repoIDStr := c.Query("repository_id"); repoIDStr != "" {
		if id, err := strconv.ParseInt(repoIDStr, 10, 64); err == nil {
			filter.RepositoryID = &id
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			filter.Limit = l
			if filter.Limit > 200 {
				filter.Limit = 200
			}
		}
	}
	if priority := c.Query("priority"); priority != "" {
		filter.Priority = priority
	}
	if assigneeStr := c.Query("assignee_id"); assigneeStr != "" {
		if id, err := strconv.ParseInt(assigneeStr, 10, 64); err == nil {
			filter.AssigneeID = &id
		}
	}
	if query := c.Query("query"); query != "" {
		filter.Query = query
	}

	board, err := h.ticketService.GetBoard(c.Request.Context(), filter)
	if err != nil {
		apierr.InternalError(c, "Failed to get board")
		return
	}

	c.JSON(http.StatusOK, gin.H{"board": board})
}

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
