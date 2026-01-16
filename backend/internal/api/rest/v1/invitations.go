package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	billingSvc "github.com/anthropics/agentsmesh/backend/internal/service/billing"
	invitationSvc "github.com/anthropics/agentsmesh/backend/internal/service/invitation"
	orgSvc "github.com/anthropics/agentsmesh/backend/internal/service/organization"
	userSvc "github.com/anthropics/agentsmesh/backend/internal/service/user"
	"github.com/gin-gonic/gin"
)

// InvitationHandler handles invitation-related requests
type InvitationHandler struct {
	invitationService *invitationSvc.Service
	orgService        *orgSvc.Service
	userService       *userSvc.Service
	billingService    *billingSvc.Service
}

// NewInvitationHandler creates a new invitation handler
func NewInvitationHandler(
	invitationService *invitationSvc.Service,
	orgService *orgSvc.Service,
	userService *userSvc.Service,
	billingService *billingSvc.Service,
) *InvitationHandler {
	return &InvitationHandler{
		invitationService: invitationService,
		orgService:        orgService,
		userService:       userService,
		billingService:    billingService,
	}
}

// RegisterRoutes registers invitation routes
func (h *InvitationHandler) RegisterRoutes(rg *gin.RouterGroup, authMw gin.HandlerFunc) {
	// Public routes (token-based access)
	rg.GET("/invitations/:token", h.GetInvitationByToken)

	// Authenticated routes
	auth := rg.Group("")
	auth.Use(authMw)
	{
		auth.POST("/invitations/:token/accept", h.AcceptInvitation)
		auth.GET("/invitations/pending", h.ListPendingInvitations)
	}

	// Organization-scoped routes (require org membership)
	// These are registered separately in the org routes
}

// RegisterOrgRoutes registers organization-scoped invitation routes
func (h *InvitationHandler) RegisterOrgRoutes(rg *gin.RouterGroup) {
	rg.GET("/invitations", h.ListOrgInvitations)
	rg.POST("/invitations", h.CreateInvitation)
	rg.DELETE("/invitations/:id", h.RevokeInvitation)
	rg.POST("/invitations/:id/resend", h.ResendInvitation)
}

// CreateInvitationRequest represents an invitation creation request
type CreateInvitationRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" binding:"required,oneof=admin member"`
}

// CreateInvitation creates a new invitation
// POST /api/v1/organizations/:org/invitations
func (h *InvitationHandler) CreateInvitation(c *gin.Context) {
	var req CreateInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tc := middleware.GetTenant(c)
	if tc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant context not found"})
		return
	}

	// Only admins and owners can invite
	if tc.UserRole != organization.RoleOwner && tc.UserRole != organization.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins and owners can invite members"})
		return
	}

	// Check seat quota before inviting
	if h.billingService != nil {
		if err := h.billingService.CheckQuota(c.Request.Context(), tc.OrganizationID, "users", 1); err != nil {
			if err == billingSvc.ErrQuotaExceeded {
				c.JSON(http.StatusPaymentRequired, gin.H{
					"error": "Seat quota exceeded. Please upgrade your plan to invite more members.",
					"code":  "SEAT_QUOTA_EXCEEDED",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check quota"})
			return
		}
	}

	// Get inviter info
	inviter, err := h.userService.GetByID(c.Request.Context(), tc.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}

	// Get org info
	org, err := h.orgService.GetByID(c.Request.Context(), tc.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get organization info"})
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
			c.JSON(http.StatusConflict, gin.H{"error": "User is already a member of this organization"})
		case invitationSvc.ErrPendingInvitation:
			c.JSON(http.StatusConflict, gin.H{"error": "A pending invitation already exists for this email"})
		case invitationSvc.ErrInvalidRole:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create invitation"})
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant context not found"})
		return
	}

	invitations, err := h.invitationService.ListByOrganization(c.Request.Context(), tc.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list invitations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"invitations": invitations})
}

// RevokeInvitation revokes a pending invitation
// DELETE /api/v1/organizations/:org/invitations/:id
func (h *InvitationHandler) RevokeInvitation(c *gin.Context) {
	tc := middleware.GetTenant(c)
	if tc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant context not found"})
		return
	}

	// Only admins and owners can revoke
	if tc.UserRole != organization.RoleOwner && tc.UserRole != organization.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins and owners can revoke invitations"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid invitation ID"})
		return
	}

	// Verify invitation belongs to this org
	inv, err := h.invitationService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invitation not found"})
		return
	}

	if inv.OrganizationID != tc.OrganizationID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invitation not found"})
		return
	}

	if err := h.invitationService.Revoke(c.Request.Context(), id); err != nil {
		switch err {
		case invitationSvc.ErrInvitationAccepted:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot revoke an accepted invitation"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke invitation"})
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant context not found"})
		return
	}

	// Only admins and owners can resend
	if tc.UserRole != organization.RoleOwner && tc.UserRole != organization.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins and owners can resend invitations"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid invitation ID"})
		return
	}

	// Verify invitation belongs to this org
	inv, err := h.invitationService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invitation not found"})
		return
	}

	if inv.OrganizationID != tc.OrganizationID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invitation not found"})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot resend an accepted invitation"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resend invitation"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation resent successfully"})
}

// GetInvitationByToken gets invitation info by token (public)
// GET /api/v1/invitations/:token
func (h *InvitationHandler) GetInvitationByToken(c *gin.Context) {
	token := c.Param("token")

	info, err := h.invitationService.GetInvitationInfo(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invitation not found or expired"})
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
			c.JSON(http.StatusNotFound, gin.H{"error": "Invitation not found"})
		case invitationSvc.ErrInvitationExpired:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invitation has expired"})
		case invitationSvc.ErrInvitationAccepted:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invitation already accepted"})
		case invitationSvc.ErrAlreadyMember:
			c.JSON(http.StatusConflict, gin.H{"error": "You are already a member of this organization"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to accept invitation"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}

	invitations, err := h.invitationService.ListPendingByEmail(c.Request.Context(), user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list invitations"})
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
