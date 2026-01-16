package organization

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
)

// Interface defines the organization service operations.
// This interface allows for easy mocking in tests.
type Interface interface {
	// Organization CRUD
	Create(ctx context.Context, ownerID int64, req *CreateRequest) (*organization.Organization, error)
	GetByID(ctx context.Context, id int64) (*organization.Organization, error)
	GetBySlug(ctx context.Context, slug string) (middleware.OrganizationGetter, error)
	GetOrgBySlug(ctx context.Context, slug string) (*organization.Organization, error)
	Update(ctx context.Context, id int64, updates map[string]interface{}) (*organization.Organization, error)
	Delete(ctx context.Context, id int64) error

	// User organizations
	ListByUser(ctx context.Context, userID int64) ([]*organization.Organization, error)

	// Members
	AddMember(ctx context.Context, orgID, userID int64, role string) error
	RemoveMember(ctx context.Context, orgID, userID int64) error
	UpdateMemberRole(ctx context.Context, orgID, userID int64, role string) error
	GetMember(ctx context.Context, orgID, userID int64) (*organization.Member, error)
	ListMembers(ctx context.Context, orgID int64) ([]*user.User, error)

	// Role checks
	IsAdmin(ctx context.Context, orgID, userID int64) (bool, error)
	IsOwner(ctx context.Context, orgID, userID int64) (bool, error)
	IsMember(ctx context.Context, orgID, userID int64) (bool, error)
	GetUserRole(ctx context.Context, orgID, userID int64) (string, error)
	GetMemberRole(ctx context.Context, orgID, userID int64) (string, error)
}

// Ensure Service implements Interface
var _ Interface = (*Service)(nil)
