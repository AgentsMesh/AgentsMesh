package user

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
)

// Interface defines the user service operations.
// This interface allows for easy mocking in tests.
type Interface interface {
	// User CRUD
	Create(ctx context.Context, req *CreateRequest) (*user.User, error)
	GetByID(ctx context.Context, id int64) (*user.User, error)
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	GetByUsername(ctx context.Context, username string) (*user.User, error)
	Update(ctx context.Context, id int64, updates map[string]interface{}) (*user.User, error)
	Delete(ctx context.Context, id int64) error

	// Authentication
	Authenticate(ctx context.Context, email, password string) (*user.User, error)
	UpdatePassword(ctx context.Context, id int64, password string) error

	// OAuth
	GetOrCreateByOAuth(ctx context.Context, provider, providerUserID, providerUsername, email, name, avatarURL string) (*user.User, bool, error)
	UpdateIdentityTokens(ctx context.Context, userID int64, provider, accessToken, refreshToken string, expiresAt *time.Time) error
	GetIdentity(ctx context.Context, userID int64, provider string) (*user.Identity, error)
	ListIdentities(ctx context.Context, userID int64) ([]*user.Identity, error)
	DeleteIdentity(ctx context.Context, userID int64, provider string) error

	// Search
	Search(ctx context.Context, query string, limit int) ([]*user.User, error)
}

// Ensure Service implements Interface
var _ Interface = (*Service)(nil)
