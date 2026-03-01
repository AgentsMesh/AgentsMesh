package v1

import (
	"net/http"
	"strconv"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ========== Commit Management Endpoints ==========

// LinkCommitRequest represents commit link request
type LinkCommitRequest struct {
	CommitSHA     string `json:"commit_sha" binding:"required"`
	CommitMessage string `json:"commit_message"`
	CommitURL     string `json:"commit_url"`
	AuthorName    string `json:"author_name"`
	AuthorEmail   string `json:"author_email"`
}

// ListCommits lists commits for a ticket
// GET /api/v1/organizations/:slug/tickets/:ticket_slug/commits
func (h *TicketHandler) ListCommits(c *gin.Context) {
	slug := c.Param("ticket_slug")
	tenant := middleware.GetTenant(c)

	t, err := h.ticketService.GetTicketBySlug(c.Request.Context(), tenant.OrganizationID, slug)
	if err != nil {
		apierr.ResourceNotFound(c, "Ticket not found")
		return
	}

	commits, err := h.ticketService.ListCommits(c.Request.Context(), t.ID)
	if err != nil {
		apierr.InternalError(c, "Failed to list commits")
		return
	}

	c.JSON(http.StatusOK, gin.H{"commits": commits})
}

// LinkCommit links a commit to a ticket
// POST /api/v1/organizations/:slug/tickets/:ticket_slug/commits
func (h *TicketHandler) LinkCommit(c *gin.Context) {
	slug := c.Param("ticket_slug")

	var req LinkCommitRequest
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

	if t.RepositoryID == nil {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Ticket has no repository")
		return
	}

	var commitURL, authorName, authorEmail *string
	if req.CommitURL != "" {
		commitURL = &req.CommitURL
	}
	if req.AuthorName != "" {
		authorName = &req.AuthorName
	}
	if req.AuthorEmail != "" {
		authorEmail = &req.AuthorEmail
	}

	commit, err := h.ticketService.LinkCommit(
		c.Request.Context(),
		tenant.OrganizationID,
		t.ID,
		*t.RepositoryID,
		nil, // podID
		req.CommitSHA,
		req.CommitMessage,
		commitURL,
		authorName,
		authorEmail,
		nil, // committedAt
	)
	if err != nil {
		apierr.InternalError(c, "Failed to link commit")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"commit": commit})
}

// UnlinkCommit unlinks a commit from a ticket
// DELETE /api/v1/organizations/:slug/tickets/:ticket_slug/commits/:commit_id
func (h *TicketHandler) UnlinkCommit(c *gin.Context) {
	slug := c.Param("ticket_slug")
	commitID, err := strconv.ParseInt(c.Param("commit_id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid commit ID")
		return
	}

	tenant := middleware.GetTenant(c)

	t, err := h.ticketService.GetTicketBySlug(c.Request.Context(), tenant.OrganizationID, slug)
	if err != nil {
		apierr.ResourceNotFound(c, "Ticket not found")
		return
	}

	_ = t // used for org-scoped lookup
	if err := h.ticketService.UnlinkCommit(c.Request.Context(), commitID); err != nil {
		apierr.InternalError(c, "Failed to unlink commit")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Commit unlinked"})
}
