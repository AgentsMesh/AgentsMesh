package v1

import (
	"github.com/anthropics/agentsmesh/backend/internal/service/organization"
	"github.com/anthropics/agentsmesh/backend/internal/service/user"
)

type OrganizationHandler struct {
	orgService  *organization.Service
	userService *user.Service
}

func NewOrganizationHandler(orgService *organization.Service, userService *user.Service) *OrganizationHandler {
	return &OrganizationHandler{
		orgService:  orgService,
		userService: userService,
	}
}

type CreateOrganizationRequest struct {
	Name    string `json:"name" binding:"required,min=2,max=100"`
	Slug    string `json:"slug" binding:"required,min=2,max=100"`
	LogoURL string `json:"logo_url"`
}

type UpdateOrganizationRequest struct {
	Name    string `json:"name"`
	LogoURL string `json:"logo_url"`
}

type InviteMemberRequest struct {
	Email  string `json:"email"`
	UserID int64  `json:"user_id"`
	Role   string `json:"role" binding:"required,oneof=admin member"`
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin member"`
}
