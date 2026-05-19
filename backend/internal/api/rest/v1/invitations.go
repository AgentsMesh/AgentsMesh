package v1

import (
	billingSvc "github.com/anthropics/agentsmesh/backend/internal/service/billing"
	invitationSvc "github.com/anthropics/agentsmesh/backend/internal/service/invitation"
	orgSvc "github.com/anthropics/agentsmesh/backend/internal/service/organization"
	userSvc "github.com/anthropics/agentsmesh/backend/internal/service/user"
	"github.com/gin-gonic/gin"
)

type InvitationHandler struct {
	invitationService *invitationSvc.Service
	orgService        *orgSvc.Service
	userService       *userSvc.Service
	billingService    *billingSvc.Service
}

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

func (h *InvitationHandler) RegisterRoutes(rg *gin.RouterGroup, authMw gin.HandlerFunc) {
	rg.GET("/invitations/:token", h.GetInvitationByToken)

	auth := rg.Group("")
	auth.Use(authMw)
	{
		auth.POST("/invitations/:token/accept", h.AcceptInvitation)
		auth.GET("/invitations/pending", h.ListPendingInvitations)
	}

}

func (h *InvitationHandler) RegisterOrgRoutes(rg *gin.RouterGroup) {
	rg.GET("/invitations", h.ListOrgInvitations)
	rg.POST("/invitations", h.CreateInvitation)
	rg.DELETE("/invitations/:id", h.RevokeInvitation)
	rg.POST("/invitations/:id/resend", h.ResendInvitation)
}

type CreateInvitationRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" binding:"required,oneof=admin member"`
}
