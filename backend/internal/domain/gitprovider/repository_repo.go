package gitprovider

import (
	"context"
)

type RepositoryRepo interface {
	FindByOrgAndSlug(ctx context.Context, orgID int64, providerType, providerBaseURL, slug string) (*Repository, error)

	Create(ctx context.Context, repo *Repository) error

	GetByID(ctx context.Context, id int64) (*Repository, error)

	Update(ctx context.Context, id int64, updates map[string]interface{}) error

	CountLoopRefs(ctx context.Context, repoID int64) (int64, error)

	SoftDelete(ctx context.Context, id int64) error

	HardDelete(ctx context.Context, id int64) error

	ListByOrganization(ctx context.Context, orgID int64) ([]*Repository, error)

	ListByOrganizationForUser(ctx context.Context, orgID int64, userID int64) ([]*Repository, error)

	GetByExternalID(ctx context.Context, providerType, providerBaseURL, externalID string) (*Repository, error)

	GetBySlug(ctx context.Context, orgID int64, providerType, providerBaseURL, slug string) (*Repository, error)

	FindByOrgSlug(ctx context.Context, orgID int64, slug string) (*Repository, error)

	GetMaxTicketNumber(ctx context.Context, repoID int64) (int, error)

	Save(ctx context.Context, repo *Repository) error

	ListMergeRequests(ctx context.Context, repoID int64, branch, state string) ([]MergeRequestRow, error)
}

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
