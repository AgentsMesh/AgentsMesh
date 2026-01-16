package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// ========== Label Management Endpoints ==========

// CreateLabelRequest represents label creation request
type CreateLabelRequest struct {
	Name         string `json:"name" binding:"required,min=1,max=100"`
	Color        string `json:"color" binding:"required,hexcolor"`
	RepositoryID *int64 `json:"repository_id"`
}

// ListLabels lists labels
// GET /api/v1/organizations/:slug/labels
func (h *TicketHandler) ListLabels(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	repositoryID := c.Query("repository_id")
	var repoID *int64
	if repositoryID != "" {
		id, err := strconv.ParseInt(repositoryID, 10, 64)
		if err == nil {
			repoID = &id
		}
	}

	labels, err := h.ticketService.ListLabels(c.Request.Context(), tenant.OrganizationID, repoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list labels"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"labels": labels})
}

// CreateLabel creates a new label
// POST /api/v1/organizations/:slug/labels
func (h *TicketHandler) CreateLabel(c *gin.Context) {
	var req CreateLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)

	label, err := h.ticketService.CreateLabel(c.Request.Context(), tenant.OrganizationID, req.RepositoryID, req.Name, req.Color)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create label"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"label": label})
}

// UpdateLabelRequest represents label update request
type UpdateLabelRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// UpdateLabel updates a label
// PUT /api/v1/organizations/:slug/labels/:id
func (h *TicketHandler) UpdateLabel(c *gin.Context) {
	labelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label ID"})
		return
	}

	var req UpdateLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Color != "" {
		updates["color"] = req.Color
	}

	label, err := h.ticketService.UpdateLabel(c.Request.Context(), tenant.OrganizationID, labelID, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update label"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"label": label})
}

// DeleteLabel deletes a label
// DELETE /api/v1/organizations/:slug/labels/:id
func (h *TicketHandler) DeleteLabel(c *gin.Context) {
	labelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label ID"})
		return
	}

	tenant := middleware.GetTenant(c)

	if err := h.ticketService.DeleteLabel(c.Request.Context(), tenant.OrganizationID, labelID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete label"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Label deleted"})
}

// AddLabelRequest represents label addition request
type AddLabelRequest struct {
	LabelID int64 `json:"label_id" binding:"required"`
}

// AddLabel adds a label to a ticket
// POST /api/v1/organizations/:slug/tickets/:identifier/labels
func (h *TicketHandler) AddLabel(c *gin.Context) {
	identifier := c.Param("identifier")

	var req AddLabelRequest
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

	if err := h.ticketService.AddLabel(c.Request.Context(), t.ID, req.LabelID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add label"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Label added"})
}

// RemoveLabel removes a label from a ticket
// DELETE /api/v1/organizations/:slug/tickets/:identifier/labels/:label_id
func (h *TicketHandler) RemoveLabel(c *gin.Context) {
	identifier := c.Param("identifier")
	labelID, err := strconv.ParseInt(c.Param("label_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid label ID"})
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

	if err := h.ticketService.RemoveLabel(c.Request.Context(), t.ID, labelID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove label"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Label removed"})
}
