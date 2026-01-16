package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// ========== Ticket Relations Endpoints ==========

// CreateRelationRequest represents relation creation request
type CreateRelationRequest struct {
	TargetIdentifier string `json:"target_identifier" binding:"required"`
	RelationType     string `json:"relation_type" binding:"required,oneof=blocks blocked_by relates_to duplicates"`
}

// ListRelations lists relations for a ticket
// GET /api/v1/organizations/:slug/tickets/:identifier/relations
func (h *TicketHandler) ListRelations(c *gin.Context) {
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

	relations, err := h.ticketService.ListRelations(c.Request.Context(), t.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list relations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"relations": relations})
}

// CreateRelation creates a relation between tickets
// POST /api/v1/organizations/:slug/tickets/:identifier/relations
func (h *TicketHandler) CreateRelation(c *gin.Context) {
	identifier := c.Param("identifier")

	var req CreateRelationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := middleware.GetTenant(c)

	// Get source ticket
	sourceTicket, err := h.ticketService.GetTicketByIdentifier(c.Request.Context(), identifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Source ticket not found"})
		return
	}

	if sourceTicket.OrganizationID != tenant.OrganizationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get target ticket
	targetTicket, err := h.ticketService.GetTicketByIdentifier(c.Request.Context(), req.TargetIdentifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Target ticket not found"})
		return
	}

	relation, err := h.ticketService.CreateRelation(
		c.Request.Context(),
		tenant.OrganizationID,
		sourceTicket.ID,
		targetTicket.ID,
		req.RelationType,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create relation: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"relation": relation})
}

// DeleteRelation deletes a relation
// DELETE /api/v1/organizations/:slug/tickets/:identifier/relations/:relation_id
func (h *TicketHandler) DeleteRelation(c *gin.Context) {
	identifier := c.Param("identifier")
	relationID, err := strconv.ParseInt(c.Param("relation_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid relation ID"})
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

	if err := h.ticketService.DeleteRelation(c.Request.Context(), relationID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete relation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Relation deleted"})
}

// ListMergeRequests lists merge requests for a ticket
// GET /api/v1/organizations/:slug/tickets/:identifier/merge-requests
func (h *TicketHandler) ListMergeRequests(c *gin.Context) {
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

	mergeRequests, err := h.ticketService.ListMergeRequests(c.Request.Context(), t.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list merge requests"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"merge_requests": mergeRequests})
}
