package v1

import (
	"regexp"

	"github.com/anthropics/agentsmesh/backend/internal/service/organization"
	"github.com/anthropics/agentsmesh/backend/internal/service/user"
)

// slugRegex validates organization slug: lowercase letters, numbers, and hyphens
// Must start and end with alphanumeric, no consecutive hyphens
var slugRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// OrganizationHandler handles organization-related requests
type OrganizationHandler struct {
	orgService  *organization.Service
	userService *user.Service
}

// NewOrganizationHandler creates a new organization handler
func NewOrganizationHandler(orgService *organization.Service, userService *user.Service) *OrganizationHandler {
	return &OrganizationHandler{
		orgService:  orgService,
		userService: userService,
	}
}

// CreateOrganizationRequest represents organization creation request
type CreateOrganizationRequest struct {
	Name    string `json:"name" binding:"required,min=2,max=100"`
	Slug    string `json:"slug" binding:"required,min=2,max=100"`
	LogoURL string `json:"logo_url"`
}

// UpdateOrganizationRequest represents organization update request
type UpdateOrganizationRequest struct {
	Name    string `json:"name"`
	LogoURL string `json:"logo_url"`
}

// InviteMemberRequest represents member invitation request
// Supports both email-based invitation and direct user_id addition
type InviteMemberRequest struct {
	Email  string `json:"email"`
	UserID int64  `json:"user_id"`
	Role   string `json:"role" binding:"required,oneof=admin member"`
}

// UpdateMemberRoleRequest represents role update request
type UpdateMemberRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin member"`
}
