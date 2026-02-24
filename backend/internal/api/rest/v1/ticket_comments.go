package v1

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	ticketsvc "github.com/anthropics/agentsmesh/backend/internal/service/ticket"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ========== Comment Management Endpoints ==========

// CreateCommentRequest represents a comment creation request
type CreateCommentRequest struct {
	Content  string `json:"content" binding:"required,min=1"`
	ParentID *int64 `json:"parent_id"`
	Mentions []struct {
		UserID   int64  `json:"user_id"`
		Username string `json:"username"`
	} `json:"mentions"`
}

// UpdateCommentRequest represents a comment update request
type UpdateCommentRequest struct {
	Content  string `json:"content" binding:"required,min=1"`
	Mentions []struct {
		UserID   int64  `json:"user_id"`
		Username string `json:"username"`
	} `json:"mentions"`
}

// ListComments lists comments for a ticket
// GET /api/v1/orgs/:slug/tickets/:ticket_slug/comments
func (h *TicketHandler) ListComments(c *gin.Context) {
	slug := c.Param("ticket_slug")
	tenant := middleware.GetTenant(c)

	t, err := h.ticketService.GetTicketBySlug(c.Request.Context(), tenant.OrganizationID, slug)
	if err != nil {
		apierr.ResourceNotFound(c, "Ticket not found")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	comments, total, err := h.ticketService.ListComments(c.Request.Context(), t.ID, limit, offset)
	if err != nil {
		apierr.InternalError(c, "Failed to list comments")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"comments": comments,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// CreateComment creates a new comment on a ticket
// POST /api/v1/orgs/:slug/tickets/:ticket_slug/comments
func (h *TicketHandler) CreateComment(c *gin.Context) {
	slug := c.Param("ticket_slug")

	var req CreateCommentRequest
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

	// Convert mentions
	var mentions []ticket.CommentMention
	for _, m := range req.Mentions {
		mentions = append(mentions, ticket.CommentMention{
			UserID:   m.UserID,
			Username: m.Username,
		})
	}

	comment, err := h.ticketService.CreateComment(
		c.Request.Context(),
		t.ID,
		tenant.UserID,
		req.Content,
		req.ParentID,
		mentions,
	)
	if err != nil {
		if errors.Is(err, ticketsvc.ErrCommentNotFound) {
			apierr.ResourceNotFound(c, "Parent comment not found")
			return
		}
		apierr.InternalError(c, "Failed to create comment")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"comment": comment})
}

// UpdateComment updates a comment
// PUT /api/v1/orgs/:slug/tickets/:ticket_slug/comments/:id
func (h *TicketHandler) UpdateComment(c *gin.Context) {
	slug := c.Param("ticket_slug")
	commentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid comment ID")
		return
	}

	var req UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)

	// Verify ticket exists and belongs to org
	t, err := h.ticketService.GetTicketBySlug(c.Request.Context(), tenant.OrganizationID, slug)
	if err != nil {
		apierr.ResourceNotFound(c, "Ticket not found")
		return
	}

	// Convert mentions
	var mentions []ticket.CommentMention
	for _, m := range req.Mentions {
		mentions = append(mentions, ticket.CommentMention{
			UserID:   m.UserID,
			Username: m.Username,
		})
	}

	comment, err := h.ticketService.UpdateComment(
		c.Request.Context(),
		t.ID,
		commentID,
		tenant.UserID,
		req.Content,
		mentions,
	)
	if err != nil {
		if errors.Is(err, ticketsvc.ErrCommentNotFound) {
			apierr.ResourceNotFound(c, "Comment not found")
			return
		}
		if errors.Is(err, ticketsvc.ErrUnauthorizedComment) {
			apierr.Forbidden(c, apierr.ACCESS_DENIED, "Only the author can edit this comment")
			return
		}
		apierr.InternalError(c, "Failed to update comment")
		return
	}

	c.JSON(http.StatusOK, gin.H{"comment": comment})
}

// DeleteComment deletes a comment
// DELETE /api/v1/orgs/:slug/tickets/:ticket_slug/comments/:id
func (h *TicketHandler) DeleteComment(c *gin.Context) {
	slug := c.Param("ticket_slug")
	commentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid comment ID")
		return
	}

	tenant := middleware.GetTenant(c)

	// Verify ticket exists and belongs to org
	t, err := h.ticketService.GetTicketBySlug(c.Request.Context(), tenant.OrganizationID, slug)
	if err != nil {
		apierr.ResourceNotFound(c, "Ticket not found")
		return
	}

	if err := h.ticketService.DeleteComment(c.Request.Context(), t.ID, commentID, tenant.UserID); err != nil {
		if errors.Is(err, ticketsvc.ErrCommentNotFound) {
			apierr.ResourceNotFound(c, "Comment not found")
			return
		}
		if errors.Is(err, ticketsvc.ErrUnauthorizedComment) {
			apierr.Forbidden(c, apierr.ACCESS_DENIED, "Only the author can delete this comment")
			return
		}
		apierr.InternalError(c, "Failed to delete comment")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Comment deleted"})
}
