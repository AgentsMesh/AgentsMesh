package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
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
// GET /api/v1/organizations/:slug/tickets/:identifier/commits
func (h *TicketHandler) ListCommits(c *gin.Context) {
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

	commits, err := h.ticketService.ListCommits(c.Request.Context(), t.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list commits"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"commits": commits})
}

// LinkCommit links a commit to a ticket
// POST /api/v1/organizations/:slug/tickets/:identifier/commits
func (h *TicketHandler) LinkCommit(c *gin.Context) {
	identifier := c.Param("identifier")

	var req LinkCommitRequest
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

	if t.RepositoryID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticket has no repository"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to link commit"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"commit": commit})
}

// UnlinkCommit unlinks a commit from a ticket
// DELETE /api/v1/organizations/:slug/tickets/:identifier/commits/:commit_id
func (h *TicketHandler) UnlinkCommit(c *gin.Context) {
	identifier := c.Param("identifier")
	commitID, err := strconv.ParseInt(c.Param("commit_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid commit ID"})
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

	if err := h.ticketService.UnlinkCommit(c.Request.Context(), commitID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unlink commit"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Commit unlinked"})
}
