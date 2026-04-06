package v1

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	ticketDomain "github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/service/ticket"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// TicketHandler handles ticket-related requests
// Note: Event publishing has been moved to the Service layer (ticket.Service)
// following the Information Expert principle - the Service owns the business logic
type TicketHandler struct {
	ticketService *ticket.Service
}

// NewTicketHandler creates a new ticket handler
func NewTicketHandler(ticketService *ticket.Service) *TicketHandler {
	return &TicketHandler{
		ticketService: ticketService,
	}
}

// ========== Request Types ==========

// ListTicketsRequest represents ticket list request
type ListTicketsRequest struct {
	RepositoryID *int64   `form:"repository_id"`
	Status       string   `form:"status"`
	Priority     string   `form:"priority"`
	AssigneeID   *int64   `form:"assignee_id"`
	Labels       []string `form:"labels"`
	Query        string   `form:"query"`
	Limit        int      `form:"limit"`
	Offset       int      `form:"offset"`
}

// CreateTicketRequest represents ticket creation request
type CreateTicketRequest struct {
	RepositoryID     *int64   `json:"repository_id"`
	Title            string   `json:"title" binding:"required,min=1,max=500"`
	Content          string   `json:"content"`
	Status           string   `json:"status"`
	Priority         string   `json:"priority"`
	AssigneeIDs      []int64  `json:"assignee_ids"`
	Labels           []string `json:"labels"`
	ParentTicketSlug *string  `json:"parent_ticket_slug"`
	DueDate          *string  `json:"due_date"`
}

// UpdateTicketRequest represents ticket update request
type UpdateTicketRequest struct {
	Title        string   `json:"title"`
	Content      *string  `json:"content"`
	Status       string   `json:"status"`
	Priority     string   `json:"priority"`
	RepositoryID *int64   `json:"repository_id"`
	AssigneeIDs  []int64  `json:"assignee_ids"`
	Labels       []string `json:"labels"`
	DueDate      *string  `json:"due_date"`
}

// UpdateTicketStatusRequest represents status update request
type UpdateTicketStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// ========== Core Ticket CRUD Endpoints ==========

// ListTickets lists tickets
// GET /api/v1/organizations/:slug/tickets
func (h *TicketHandler) ListTickets(c *gin.Context) {
	var req ListTicketsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)

	limit := req.Limit
	if limit == 0 {
		limit = 20
	}

	tickets, total, err := h.ticketService.ListTickets(c.Request.Context(), &ticketDomain.TicketListFilter{
		OrganizationID: tenant.OrganizationID,
		RepositoryID:   req.RepositoryID,
		Status:         req.Status,
		Priority:       req.Priority,
		AssigneeID:     req.AssigneeID,
		Query:          req.Query,
		UserRole:       tenant.UserRole,
		Limit:          limit,
		Offset:         req.Offset,
	})
	if err != nil {
		apierr.InternalError(c, "Failed to list tickets")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tickets": tickets,
		"total":   total,
		"limit":   limit,
		"offset":  req.Offset,
	})
}

// CreateTicket creates a new ticket
// POST /api/v1/organizations/:slug/tickets
func (h *TicketHandler) CreateTicket(c *gin.Context) {
	var req CreateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)

	var content *string
	if req.Content != "" {
		content = &req.Content
	}

	// Resolve parent ticket slug to ID
	var parentTicketID *int64
	if req.ParentTicketSlug != nil && *req.ParentTicketSlug != "" {
		parent, err := h.ticketService.GetTicketByIDOrSlug(c.Request.Context(), tenant.OrganizationID, *req.ParentTicketSlug)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Parent ticket not found"})
			return
		}
		parentTicketID = &parent.ID
	}

	t, err := h.ticketService.CreateTicket(c.Request.Context(), &ticket.CreateTicketRequest{
		OrganizationID: tenant.OrganizationID,
		RepositoryID:   req.RepositoryID,
		ReporterID:     tenant.UserID,
		Title:          req.Title,
		Content:        content,
		Status:         req.Status,
		Priority:       req.Priority,
		AssigneeIDs:    req.AssigneeIDs,
		Labels:         req.Labels,
		ParentTicketID: parentTicketID,
	})
	if err != nil {
		apierr.InternalError(c, "Failed to create ticket")
		return
	}

	// Note: Event publishing is now handled in ticket.Service.CreateTicket()

	c.JSON(http.StatusCreated, gin.H{"ticket": t})
}

// GetTicket returns ticket by slug
// GET /api/v1/organizations/:slug/tickets/:ticket_slug
func (h *TicketHandler) GetTicket(c *gin.Context) {
	slug := c.Param("ticket_slug")
	tenant := middleware.GetTenant(c)

	t, err := h.ticketService.GetTicketBySlug(c.Request.Context(), tenant.OrganizationID, slug)
	if err != nil {
		apierr.ResourceNotFound(c, "Ticket not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"ticket": t})
}

// UpdateTicket updates a ticket
// PUT /api/v1/organizations/:slug/tickets/:ticket_slug
func (h *TicketHandler) UpdateTicket(c *gin.Context) {
	slug := c.Param("ticket_slug")
	tenant := middleware.GetTenant(c)

	var req UpdateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	t, err := h.ticketService.GetTicketBySlug(c.Request.Context(), tenant.OrganizationID, slug)
	if err != nil {
		apierr.ResourceNotFound(c, "Ticket not found")
		return
	}

	updates := make(map[string]interface{})
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Content != nil {
		updates["content"] = *req.Content
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.Priority != "" {
		updates["priority"] = req.Priority
	}
	if req.RepositoryID != nil {
		if *req.RepositoryID == 0 {
			// Explicitly clear the repository association
			updates["repository_id"] = nil
		} else {
			updates["repository_id"] = *req.RepositoryID
		}
	}
	if req.DueDate != nil {
		if *req.DueDate == "" {
			updates["due_date"] = nil
		} else {
			updates["due_date"] = *req.DueDate
		}
	}

	t, err = h.ticketService.UpdateTicket(c.Request.Context(), t.ID, updates)
	if err != nil {
		apierr.InternalError(c, "Failed to update ticket")
		return
	}

	// Update assignees if provided
	if req.AssigneeIDs != nil {
		if err := h.ticketService.UpdateAssignees(c.Request.Context(), t.ID, req.AssigneeIDs); err != nil {
			apierr.InternalError(c, "Failed to update assignees")
			return
		}
	}

	// Note: Event publishing is now handled in ticket.Service.UpdateTicket()

	c.JSON(http.StatusOK, gin.H{"ticket": t})
}

// DeleteTicket deletes a ticket
// DELETE /api/v1/organizations/:slug/tickets/:ticket_slug
func (h *TicketHandler) DeleteTicket(c *gin.Context) {
	slug := c.Param("ticket_slug")
	tenant := middleware.GetTenant(c)

	t, err := h.ticketService.GetTicketBySlug(c.Request.Context(), tenant.OrganizationID, slug)
	if err != nil {
		apierr.ResourceNotFound(c, "Ticket not found")
		return
	}

	if err := h.ticketService.DeleteTicket(c.Request.Context(), t.ID); err != nil {
		apierr.InternalError(c, "Failed to delete ticket")
		return
	}

	// Note: Event publishing is now handled in ticket.Service.DeleteTicket()

	c.JSON(http.StatusOK, gin.H{"message": "Ticket deleted"})
}

// UpdateTicketStatus updates ticket status (convenience endpoint)
// PATCH /api/v1/organizations/:slug/tickets/:ticket_slug/status
func (h *TicketHandler) UpdateTicketStatus(c *gin.Context) {
	slug := c.Param("ticket_slug")

	var req UpdateTicketStatusRequest
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

	if err := h.ticketService.UpdateStatus(c.Request.Context(), t.ID, req.Status); err != nil {
		apierr.InternalError(c, "Failed to update status")
		return
	}

	// Note: Event publishing is now handled in ticket.Service.UpdateStatus()

	c.JSON(http.StatusOK, gin.H{"message": "Status updated"})
}
