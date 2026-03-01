package v1

import (
	"net/http"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	invitationSvc "github.com/AgentsMesh/AgentsMesh/backend/internal/service/invitation"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// GetInvitationByToken gets invitation info by token (public)
// GET /api/v1/invitations/:token
func (h *InvitationHandler) GetInvitationByToken(c *gin.Context) {
	token := c.Param("token")

	info, err := h.invitationService.GetInvitationInfo(c.Request.Context(), token)
	if err != nil {
		apierr.ResourceNotFound(c, "Invitation not found or expired")
		return
	}

	c.JSON(http.StatusOK, gin.H{"invitation": info})
}

// AcceptInvitation accepts an invitation
// POST /api/v1/invitations/:token/accept
func (h *InvitationHandler) AcceptInvitation(c *gin.Context) {
	token := c.Param("token")
	userID := middleware.GetUserID(c)

	result, err := h.invitationService.Accept(c.Request.Context(), token, userID)
	if err != nil {
		switch err {
		case invitationSvc.ErrInvitationNotFound:
			apierr.ResourceNotFound(c, "Invitation not found")
		case invitationSvc.ErrInvitationExpired:
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Invitation has expired")
		case invitationSvc.ErrInvitationAccepted:
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Invitation already accepted")
		case invitationSvc.ErrAlreadyMember:
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "You are already a member of this organization")
		default:
			apierr.InternalError(c, "Failed to accept invitation")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Successfully joined the organization",
		"organization": result.Organization,
	})
}

// ListPendingInvitations lists pending invitations for the current user
// GET /api/v1/invitations/pending
func (h *InvitationHandler) ListPendingInvitations(c *gin.Context) {
	userID := middleware.GetUserID(c)

	user, err := h.userService.GetByID(c.Request.Context(), userID)
	if err != nil {
		apierr.InternalError(c, "Failed to get user info")
		return
	}

	invitations, err := h.invitationService.ListPendingByEmail(c.Request.Context(), user.Email)
	if err != nil {
		apierr.InternalError(c, "Failed to list invitations")
		return
	}

	// Enrich with org info
	var enriched []map[string]interface{}
	for _, inv := range invitations {
		org, err := h.orgService.GetByID(c.Request.Context(), inv.OrganizationID)
		if err != nil {
			continue
		}
		enriched = append(enriched, map[string]interface{}{
			"id":                inv.ID,
			"organization_id":   inv.OrganizationID,
			"organization_name": org.Name,
			"organization_slug": org.Slug,
			"role":              inv.Role,
			"expires_at":        inv.ExpiresAt,
			"token":             inv.Token,
		})
	}

	c.JSON(http.StatusOK, gin.H{"invitations": enriched})
}
