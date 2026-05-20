package apikey

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/apikey"
)

var (
	ErrAPIKeyNotFound    = errors.New("api key not found")
	ErrAPIKeyDisabled    = errors.New("api key is disabled")
	ErrAPIKeyExpired     = errors.New("api key has expired")
	ErrInvalidScope      = errors.New("invalid scope")
	ErrInsufficientScope = errors.New("insufficient scope for this operation")
	ErrDuplicateKeyName  = errors.New("api key name already exists in this organization")
	ErrNameEmpty         = errors.New("api key name is required")
	ErrNameTooLong       = errors.New("api key name exceeds 255 characters")
	ErrScopesRequired    = errors.New("at least one scope is required")
	ErrInvalidExpiresIn  = errors.New("expires_in must be between 300 (5 min) and 94608000 (3 years)")
)

type ValidateResult struct {
	APIKeyID       int64
	OrganizationID int64
	CreatedBy      int64
	Scopes         []string
	KeyName        string
}

type CreateAPIKeyRequest struct {
	OrganizationID int64
	CreatedBy      int64
	Name           string
	Description    *string
	Scopes         []string
	ExpiresIn      *int // seconds, nil = never expires
}

type CreateAPIKeyResponse struct {
	APIKey *apikey.APIKey
	RawKey string // Only returned once at creation time
}

type UpdateAPIKeyRequest struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Scopes      []string `json:"scopes,omitempty"`
	IsEnabled   *bool    `json:"is_enabled,omitempty"`
}

type ListAPIKeysFilter struct {
	OrganizationID int64
	IsEnabled      *bool
	Limit          int
	Offset         int
}

type Interface interface {
	CreateAPIKey(ctx context.Context, req *CreateAPIKeyRequest) (*CreateAPIKeyResponse, error)
	ValidateKey(ctx context.Context, rawKey string) (*ValidateResult, error)
	ListAPIKeys(ctx context.Context, filter *ListAPIKeysFilter) ([]apikey.APIKey, int64, error)
	GetAPIKey(ctx context.Context, id int64, orgID int64) (*apikey.APIKey, error)
	GetAPIKeyBySlug(ctx context.Context, orgID int64, slug string) (*apikey.APIKey, error)
	UpdateAPIKey(ctx context.Context, id int64, orgID int64, req *UpdateAPIKeyRequest) (*apikey.APIKey, error)
	RevokeAPIKey(ctx context.Context, id int64, orgID int64) error
	DeleteAPIKey(ctx context.Context, id int64, orgID int64) error
	UpdateLastUsed(ctx context.Context, id int64) error
}
