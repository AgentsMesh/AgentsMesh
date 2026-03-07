package apikey

import (
	"context"
	"errors"
)

// Domain-level errors
var (
	ErrNotFound = errors.New("api key not found")
)

// Repository defines the persistence interface for API keys
type Repository interface {
	// Create persists a new API key
	Create(ctx context.Context, key *APIKey) error

	// GetByID retrieves an API key by ID scoped to organization
	GetByID(ctx context.Context, id int64, orgID int64) (*APIKey, error)

	// GetByKeyHash retrieves an API key by its hash (for validation)
	GetByKeyHash(ctx context.Context, keyHash string) (*APIKey, error)

	// List returns API keys for an organization with optional filtering and pagination
	List(ctx context.Context, orgID int64, isEnabled *bool, limit, offset int) ([]APIKey, int64, error)

	// Update applies partial updates to an API key
	Update(ctx context.Context, key *APIKey, updates map[string]interface{}) error

	// Delete permanently removes an API key
	Delete(ctx context.Context, key *APIKey) error

	// UpdateLastUsed sets the last_used_at timestamp
	UpdateLastUsed(ctx context.Context, id int64) error

	// CheckDuplicateName returns true if a key with the given name exists in the org.
	// If excludeID is non-nil, that key ID is excluded from the check (for updates).
	CheckDuplicateName(ctx context.Context, orgID int64, name string, excludeID *int64) (bool, error)
}
