package apikey

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("api key not found")
)

type Repository interface {
	Create(ctx context.Context, key *APIKey) error

	GetByID(ctx context.Context, id int64, orgID int64) (*APIKey, error)

	GetByKeyHash(ctx context.Context, keyHash string) (*APIKey, error)

	GetByOrgAndSlug(ctx context.Context, orgID int64, slug string) (*APIKey, error)

	List(ctx context.Context, orgID int64, isEnabled *bool, limit, offset int) ([]APIKey, int64, error)

	Update(ctx context.Context, key *APIKey, updates map[string]interface{}) error

	Delete(ctx context.Context, key *APIKey) error

	UpdateLastUsed(ctx context.Context, id int64) error

	CheckDuplicateName(ctx context.Context, orgID int64, name string, excludeID *int64) (bool, error)

	SlugExists(ctx context.Context, orgID int64, slug string) (bool, error)
}
