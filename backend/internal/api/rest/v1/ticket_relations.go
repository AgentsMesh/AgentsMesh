package v1

import (
	"net/http"
	"strconv"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ========== Ticket Relations Endpoints ==========

// CreateRelationRequest represents relation creation request
type CreateRelationRequest struct {
	TargetSlug       string `json:"target_slug" binding:"required"`
	RelationType     string `json:"relation_type" binding:"required,oneof=blocks blocked_by relates_to duplicates"`
}

// ListRelations lists relations for a ticket
// GET /api/v1/organizations/:slug/tickets/:ticket_slug/relations
func (h *TicketHandler) ListRelations(c *gin.Context) {
	slug := c.Param("ticket_slug")
	tenant := middleware.GetTenant(c)

	t, err := h.ticketService.GetTicketBySlug(c.Request.Context(), tenant.OrganizationID, slug)
	if err != nil {
		apierr.ResourceNotFound(c, "Ticket not found")
		return
	}

	relations, err := h.ticketService.ListRelations(c.Request.Context(), t.ID)
	if err != nil {
		apierr.InternalError(c, "Failed to list relations")
		return
	}

	c.JSON(http.StatusOK, gin.H{"relations": relations})
}

// CreateRelation creates a relation between tickets
// POST /api/v1/organizations/:slug/tickets/:ticket_slug/relations
func (h *TicketHandler) CreateRelation(c *gin.Context) {
	slug := c.Param("ticket_slug")

	var req CreateRelationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)

	// Get source ticket
	sourceTicket, err := h.ticketService.GetTicketBySlug(c.Request.Context(), tenant.OrganizationID, slug)
	if err != nil {
		apierr.ResourceNotFound(c, "Source ticket not found")
		return
	}

	// Get target ticket (same org)
	targetTicket, err := h.ticketService.GetTicketBySlug(c.Request.Context(), tenant.OrganizationID, req.TargetSlug)
	if err != nil {
		apierr.ResourceNotFound(c, "Target ticket not found")
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
		apierr.InternalError(c, "Failed to create relation: "+err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"relation": relation})
}

// DeleteRelation deletes a relation
// DELETE /api/v1/organizations/:slug/tickets/:ticket_slug/relations/:relation_id
func (h *TicketHandler) DeleteRelation(c *gin.Context) {
	slug := c.Param("ticket_slug")
	relationID, err := strconv.ParseInt(c.Param("relation_id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid relation ID")
		return
	}

	tenant := middleware.GetTenant(c)

	t, err := h.ticketService.GetTicketBySlug(c.Request.Context(), tenant.OrganizationID, slug)
	if err != nil {
		apierr.ResourceNotFound(c, "Ticket not found")
		return
	}

	_ = t // used for org-scoped lookup
	if err := h.ticketService.DeleteRelation(c.Request.Context(), relationID); err != nil {
		apierr.InternalError(c, "Failed to delete relation")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Relation deleted"})
}

// ListMergeRequests lists merge requests for a ticket
// GET /api/v1/organizations/:slug/tickets/:ticket_slug/merge-requests
func (h *TicketHandler) ListMergeRequests(c *gin.Context) {
	slug := c.Param("ticket_slug")

	tenant := middleware.GetTenant(c)

	t, err := h.ticketService.GetTicketBySlug(c.Request.Context(), tenant.OrganizationID, slug)
	if err != nil {
		apierr.ResourceNotFound(c, "Ticket not found")
		return
	}

	mergeRequests, err := h.ticketService.ListMergeRequests(c.Request.Context(), t.ID)
	if err != nil {
		apierr.InternalError(c, "Failed to list merge requests")
		return
	}

	c.JSON(http.StatusOK, gin.H{"merge_requests": mergeRequests})
}
