package v1

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	billingSvc "github.com/anthropics/agentsmesh/backend/internal/service/billing"
	invitationSvc "github.com/anthropics/agentsmesh/backend/internal/service/invitation"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

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

	if tc.UserRole != organization.RoleOwner && tc.UserRole != organization.RoleAdmin {
		apierr.ForbiddenAdmin(c)
		return
	}

	if h.billingService != nil {
		if err := h.billingService.CheckSeatAvailability(c.Request.Context(), tc.OrganizationID, 1); err != nil {
			if errors.Is(err, billingSvc.ErrQuotaExceeded) {
				apierr.PaymentRequired(c, apierr.NO_AVAILABLE_SEATS, "No available seats. Please purchase more seats to invite members.")
				return
			}
			if errors.Is(err, billingSvc.ErrSubscriptionFrozen) {
				apierr.PaymentRequired(c, apierr.SUBSCRIPTION_FROZEN, "Your subscription has expired. Please renew to continue.")
				return
			}
			apierr.InternalError(c, "Failed to check seat availability")
			return
		}
	}

	inviter, err := h.userService.GetByID(c.Request.Context(), tc.UserID)
	if err != nil {
		apierr.InternalError(c, "Failed to get user info")
		return
	}

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
		switch {
		case errors.Is(err, invitationSvc.ErrAlreadyMember):
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "User is already a member of this organization")
		case errors.Is(err, invitationSvc.ErrPendingInvitation):
			apierr.Conflict(c, apierr.ALREADY_EXISTS, "A pending invitation already exists for this email")
		case errors.Is(err, invitationSvc.ErrInvalidRole):
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

func (h *InvitationHandler) RevokeInvitation(c *gin.Context) {
	tc := middleware.GetTenant(c)
	if tc == nil {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "Tenant context not found")
		return
	}

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
		switch {
		case errors.Is(err, invitationSvc.ErrInvitationAccepted):
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Cannot revoke an accepted invitation")
		default:
			apierr.InternalError(c, "Failed to revoke invitation")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation revoked successfully"})
}

func (h *InvitationHandler) ResendInvitation(c *gin.Context) {
	tc := middleware.GetTenant(c)
	if tc == nil {
		apierr.Unauthorized(c, apierr.AUTH_REQUIRED, "Tenant context not found")
		return
	}

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

	inv, err := h.invitationService.GetByID(c.Request.Context(), id)
	if err != nil {
		apierr.ResourceNotFound(c, "Invitation not found")
		return
	}

	if inv.OrganizationID != tc.OrganizationID {
		apierr.ResourceNotFound(c, "Invitation not found")
		return
	}

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
		switch {
		case errors.Is(err, invitationSvc.ErrInvitationAccepted):
			apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Cannot resend an accepted invitation")
		default:
			apierr.InternalError(c, "Failed to resend invitation")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation resent successfully"})
}
