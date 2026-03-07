package organization

import (
	"context"
	"errors"
)

// Domain-level errors
var (
	ErrNotFound       = errors.New("organization not found")
	ErrMemberNotFound = errors.New("organization member not found")
)

// CreateOrgParams holds parameters for atomic organization creation
type CreateOrgParams struct {
	Organization *Organization
	OwnerMember  *Member
	// AfterCreate is an optional callback invoked within the transaction after org+member are created.
	// The tx parameter is the active transaction handle (GORM *gorm.DB at infra layer).
	AfterCreate func(ctx context.Context, tx interface{}) error
}

// Repository defines the persistence interface for organizations and members
type Repository interface {
	// Organization CRUD
	GetByID(ctx context.Context, id int64) (*Organization, error)
	GetBySlug(ctx context.Context, slug string) (*Organization, error)
	SlugExists(ctx context.Context, slug string) (bool, error)
	Update(ctx context.Context, id int64, updates map[string]interface{}) error
	ListByUser(ctx context.Context, userID int64) ([]*Organization, error)

	// CreateWithMember atomically creates an organization, adds the owner member,
	// and executes the optional AfterCreate callback within the same transaction.
	CreateWithMember(ctx context.Context, params *CreateOrgParams) error

	// DeleteWithCleanup atomically deletes an organization and cleans up
	// tables that lack FK CASCADE (e.g. loops, loop_runs).
	DeleteWithCleanup(ctx context.Context, id int64) error

	// Member operations
	CreateMember(ctx context.Context, member *Member) error
	GetMember(ctx context.Context, orgID, userID int64) (*Member, error)
	DeleteMember(ctx context.Context, orgID, userID int64) error
	UpdateMemberRole(ctx context.Context, orgID, userID int64, role string) error
	ListMembers(ctx context.Context, orgID int64) ([]*Member, error)
	ListMembersWithUser(ctx context.Context, orgID int64) ([]*Member, error)
	MemberExists(ctx context.Context, orgID, userID int64) (bool, error)
}
