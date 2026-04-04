package gitprovider

import (
	"context"
)

// RepositoryRepo is the data access interface for Repository entities.
// Named distinctly from the domain entity Repository to avoid confusion.
type RepositoryRepo interface {
	// FindByOrgAndSlug returns a repository matching org + provider + slug.
	// Returns (nil, nil) when not found.
	FindByOrgAndSlug(ctx context.Context, orgID int64, providerType, providerBaseURL, slug string) (*Repository, error)

	// Create persists a new repository.
	Create(ctx context.Context, repo *Repository) error

	// GetByID returns a repository by primary key.
	// Returns (nil, nil) when not found.
	GetByID(ctx context.Context, id int64) (*Repository, error)

	// Update applies the given column updates to the repository with the given ID.
	Update(ctx context.Context, id int64, updates map[string]interface{}) error

	// CountLoopRefs returns the number of loops referencing the given repository.
	CountLoopRefs(ctx context.Context, repoID int64) (int64, error)

	// SoftDelete sets deleted_at on the repository.
	SoftDelete(ctx context.Context, id int64) error

	// HardDelete permanently removes the repository.
	HardDelete(ctx context.Context, id int64) error

	// ListByOrganization returns all active, non-deleted repositories for an org.
	ListByOrganization(ctx context.Context, orgID int64) ([]*Repository, error)

	// ListByOrganizationForUser returns active repositories visible to a user
	// (organization-visible + private ones imported by the user).
	ListByOrganizationForUser(ctx context.Context, orgID int64, userID int64) ([]*Repository, error)

	// GetByExternalID looks up a repository by provider type, base URL, and external ID.
	// Returns (nil, nil) when not found.
	GetByExternalID(ctx context.Context, providerType, providerBaseURL, externalID string) (*Repository, error)

	// GetBySlug looks up a repository by org + provider + slug.
	// Returns (nil, nil) when not found.
	GetBySlug(ctx context.Context, orgID int64, providerType, providerBaseURL, slug string) (*Repository, error)

	// FindByOrgSlug looks up a repository by org + slug (ignoring provider).
	// Used by AgentFile REPO resolution where only slug is known.
	// Returns (nil, nil) when not found.
	FindByOrgSlug(ctx context.Context, orgID int64, slug string) (*Repository, error)

	// GetMaxTicketNumber returns the highest ticket number for a repository.
	GetMaxTicketNumber(ctx context.Context, repoID int64) (int, error)

	// Save persists all fields of the given repository (insert or update).
	Save(ctx context.Context, repo *Repository) error

	// ListMergeRequests returns merge requests for a repository with optional filters.
	ListMergeRequests(ctx context.Context, repoID int64, branch, state string) ([]MergeRequestRow, error)
}

// MergeRequestRow is a read-only projection returned by ListMergeRequests.
type MergeRequestRow struct {
	ID             int64
	MRIID          int
	Title          string
	State          string
	MRURL          string
	SourceBranch   string
	TargetBranch   string
	PipelineStatus *string
	PipelineID     *int64
	PipelineURL    *string
	TicketID       *int64
	PodID          *int64
}
