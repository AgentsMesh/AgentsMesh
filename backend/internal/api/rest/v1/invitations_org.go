package v1

import (
	"net/http"
	"strconv"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/organization"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	billingSvc "github.com/AgentsMesh/AgentsMesh/backend/internal/service/billing"
	invitationSvc "github.com/AgentsMesh/AgentsMesh/backend/internal/service/invitation"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// CreateInvitation creates a new invitation
// POST /api/v1/organizations/:org/invitations
func (h *InvitationHandler) CreateInvitation(c *gin.Context) {
	var req CreateInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tc := middleware.GetTenant(c)
	if tc == nil {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "Tenant context not found")
		return
	}

	// Only admins and owners can invite
	if tc.UserRole != organization.RoleOwner && tc.UserRole != organization.RoleAdmin {
		apierr.ForbiddenAdmin(c)
		return
	}

	// Check seat availability before inviting
	// This checks purchased seats vs used seats (not plan limits)
	if h.billingService != nil {
		if err := h.billingService.CheckSeatAvailability(c.Request.Context(), tc.OrganizationID, 1); err != nil {
			if err == billingSvc.ErrQuotaExceeded {
				apierr.PaymentRequired(c, apierr.NO_AVAILABLE_SEATS, "No available seats. Please purchase more seats to invite members.")
				return
			}
			if err == billingSvc.ErrSubscriptionFrozen {
				apierr.PaymentRequired(c, apierr.SUBSCRIPTION_FROZEN, "Your subscription has expired. Please renew to continue.")
				return
			}
			apierr.InternalError(c, "Failed to check seat availability")
			return
		}
	}

	// Get inviter info
	inviter, err := h.userService.GetByID(c.Request.Context(), tc.UserID)
	if err != nil {
		apierr.InternalError(c, "Failed to get user info")
		return
	}

	// Get org info
	org, err := h.orgService.GetByID(c.Request.Context(), tc.OrganizationID)
	if err != nil {
		apierr.InternalError(c, "Failed to get organization info")
		return
	}

	inviterName := inviter.Username
	if inviter.Name != nil && *inviter.Name != "" {
		inviterName = *inviter.Name
	}

	inv, err := h.invitationService.Create(c.Request.Context(), &invitationSvc.CreateRequest{
		OrganizationID: tc.OrganizationID,
		Email:          req.Email,
		Role:           req.Role,
		InviterID:      tc.UserID,
		InviterName:    inviterName,
		OrgName:        org.Name,
	})

	if err != nil {
		switch err {
		case invitationSvc.ErrAlreadyMember:
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "User is already a member of this organization")
		case invitationSvc.ErrPendingInvitation:
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "A pending invitation already exists for this email")
		case invitationSvc.ErrInvalidRole:
			apierr.InvalidInput(c, "Invalid role")
		default:
			apierr.InternalError(c, "Failed to create invitation")
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Invitation sent successfully",
		"invitation": inv,
	})
}

// ListOrgInvitations lists all invitations for an organization
// GET /api/v1/organizations/:org/invitations
func (h *InvitationHandler) ListOrgInvitations(c *gin.Context) {
	tc := middleware.GetTenant(c)
	if tc == nil {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "Tenant context not found")
		return
	}

	invitations, err := h.invitationService.ListByOrganization(c.Request.Context(), tc.OrganizationID)
	if err != nil {
		apierr.InternalError(c, "Failed to list invitations")
		return
	}

	c.JSON(http.StatusOK, gin.H{"invitations": invitations})
}

// RevokeInvitation revokes a pending invitation
// DELETE /api/v1/organizations/:org/invitations/:id
func (h *InvitationHandler) RevokeInvitation(c *gin.Context) {
	tc := middleware.GetTenant(c)
	if tc == nil {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "Tenant context not found")
		return
	}

	// Only admins and owners can revoke
	if tc.UserRole != organization.RoleOwner && tc.UserRole != organization.RoleAdmin {
		apierr.ForbiddenAdmin(c)
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid invitation ID")
		return
	}

	// Verify invitation belongs to this org
	inv, err := h.invitationService.GetByID(c.Request.Context(), id)
	if err != nil {
		apierr.ResourceNotFound(c, "Invitation not found")
		return
	}

	if inv.OrganizationID != tc.OrganizationID {
		apierr.ResourceNotFound(c, "Invitation not found")
		return
	}

	if err := h.invitationService.Revoke(c.Request.Context(), id); err != nil {
		switch err {
		case invitationSvc.ErrInvitationAccepted:
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Cannot revoke an accepted invitation")
		default:
			apierr.InternalError(c, "Failed to revoke invitation")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation revoked successfully"})
}

// ResendInvitation resends an invitation email
// POST /api/v1/organizations/:org/invitations/:id/resend
func (h *InvitationHandler) ResendInvitation(c *gin.Context) {
	tc := middleware.GetTenant(c)
	if tc == nil {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "Tenant context not found")
		return
	}

	// Only admins and owners can resend
	if tc.UserRole != organization.RoleOwner && tc.UserRole != organization.RoleAdmin {
		apierr.ForbiddenAdmin(c)
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid invitation ID")
		return
	}

	// Verify invitation belongs to this org
	inv, err := h.invitationService.GetByID(c.Request.Context(), id)
	if err != nil {
		apierr.ResourceNotFound(c, "Invitation not found")
		return
	}

	if inv.OrganizationID != tc.OrganizationID {
		apierr.ResourceNotFound(c, "Invitation not found")
		return
	}

	// Get inviter and org info
	inviter, _ := h.userService.GetByID(c.Request.Context(), tc.UserID)
	org, _ := h.orgService.GetByID(c.Request.Context(), tc.OrganizationID)

	inviterName := "Someone"
	if inviter != nil {
		inviterName = inviter.Username
		if inviter.Name != nil && *inviter.Name != "" {
			inviterName = *inviter.Name
		}
	}

	orgName := "the organization"
	if org != nil {
		orgName = org.Name
	}

	if err := h.invitationService.Resend(c.Request.Context(), id, inviterName, orgName); err != nil {
		switch err {
		case invitationSvc.ErrInvitationAccepted:
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Cannot resend an accepted invitation")
		default:
			apierr.InternalError(c, "Failed to resend invitation")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation resent successfully"})
}
