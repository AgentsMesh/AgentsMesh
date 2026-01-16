package v1

import (
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/ticket"
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
	Type         string   `form:"type"`
	AssigneeID   *int64   `form:"assignee_id"`
	Labels       []string `form:"labels"`
	Limit        int      `form:"limit"`
	Offset       int      `form:"offset"`
}

// CreateTicketRequest represents ticket creation request
type CreateTicketRequest struct {
	RepositoryID   *int64   `json:"repository_id"`
	Type           string   `json:"type" binding:"required,oneof=task bug feature improvement epic subtask story"`
	Title          string   `json:"title" binding:"required,min=1,max=500"`
	Description    string   `json:"description"`
	Content        string   `json:"content"`
	Status         string   `json:"status"`
	Priority       string   `json:"priority"`
	AssigneeIDs    []int64  `json:"assignee_ids"`
	Labels         []string `json:"labels"`
	ParentTicketID *int64   `json:"parent_ticket_id"`
	DueDate        *string  `json:"due_date"`
}

// UpdateTicketRequest represents ticket update request
type UpdateTicketRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Content     string   `json:"content"`
	Status      string   `json:"status"`
	Priority    string   `json:"priority"`
	Type        string   `json:"type"`
	AssigneeIDs []int64  `json:"assignee_ids"`
	Labels      []string `json:"labels"`
	DueDate     *string  `json:"due_date"`
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)

	limit := req.Limit
	if limit == 0 {
		limit = 20
	}

	tickets, total, err := h.ticketService.ListTickets(c.Request.Context(), &ticket.ListTicketsFilter{
		OrganizationID: tenant.OrganizationID,
		RepositoryID:   req.RepositoryID,
		Status:         req.Status,
		Type:           req.Type,
		AssigneeID:     req.AssigneeID,
		UserRole:       tenant.UserRole,
		Limit:          limit,
		Offset:         req.Offset,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tickets"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)

	var description, content *string
	if req.Description != "" {
		description = &req.Description
	}
	if req.Content != "" {
		content = &req.Content
	}

	t, err := h.ticketService.CreateTicket(c.Request.Context(), &ticket.CreateTicketRequest{
		OrganizationID: tenant.OrganizationID,
		RepositoryID:   req.RepositoryID,
		ReporterID:     tenant.UserID,
		Type:           req.Type,
		Title:          req.Title,
		Description:    description,
		Content:        content,
		Status:         req.Status,
		Priority:       req.Priority,
		AssigneeIDs:    req.AssigneeIDs,
		Labels:         req.Labels,
		ParentTicketID: req.ParentTicketID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ticket"})
		return
	}

	// Note: Event publishing is now handled in ticket.Service.CreateTicket()

	c.JSON(http.StatusCreated, gin.H{"ticket": t})
}

// GetTicket returns ticket by identifier
// GET /api/v1/organizations/:slug/tickets/:identifier
func (h *TicketHandler) GetTicket(c *gin.Context) {
	identifier := c.Param("identifier")

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

	c.JSON(http.StatusOK, gin.H{"ticket": t})
}

// UpdateTicket updates a ticket
// PUT /api/v1/organizations/:slug/tickets/:identifier
func (h *TicketHandler) UpdateTicket(c *gin.Context) {
	identifier := c.Param("identifier")

	var req UpdateTicketRequest
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

	updates := make(map[string]interface{})
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Content != "" {
		updates["content"] = req.Content
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.Priority != "" {
		updates["priority"] = req.Priority
	}
	if req.Type != "" {
		updates["type"] = req.Type
	}

	t, err = h.ticketService.UpdateTicket(c.Request.Context(), t.ID, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update ticket"})
		return
	}

	// Update assignees if provided
	if req.AssigneeIDs != nil {
		if err := h.ticketService.UpdateAssignees(c.Request.Context(), t.ID, req.AssigneeIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update assignees"})
			return
		}
	}

	// Note: Event publishing is now handled in ticket.Service.UpdateTicket()

	c.JSON(http.StatusOK, gin.H{"ticket": t})
}

// DeleteTicket deletes a ticket
// DELETE /api/v1/organizations/:slug/tickets/:identifier
func (h *TicketHandler) DeleteTicket(c *gin.Context) {
	identifier := c.Param("identifier")

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

	if err := h.ticketService.DeleteTicket(c.Request.Context(), t.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete ticket"})
		return
	}

	// Note: Event publishing is now handled in ticket.Service.DeleteTicket()

	c.JSON(http.StatusOK, gin.H{"message": "Ticket deleted"})
}

// UpdateTicketStatus updates ticket status (convenience endpoint)
// PATCH /api/v1/organizations/:slug/tickets/:identifier/status
func (h *TicketHandler) UpdateTicketStatus(c *gin.Context) {
	identifier := c.Param("identifier")

	var req UpdateTicketStatusRequest
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

	if err := h.ticketService.UpdateStatus(c.Request.Context(), t.ID, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	// Note: Event publishing is now handled in ticket.Service.UpdateStatus()

	c.JSON(http.StatusOK, gin.H{"message": "Status updated"})
}
