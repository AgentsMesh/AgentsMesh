package user

import (
	"context"
	"errors"
)

// Domain-level errors
var (
	ErrNotFound         = errors.New("user not found")
	ErrIdentityNotFound = errors.New("identity not found")
)

// Repository defines the persistence interface for users, identities,
// git credentials, and repository providers.
type Repository interface {
	// User CRUD
	CreateUser(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	EmailExists(ctx context.Context, email string) (bool, error)
	UsernameExists(ctx context.Context, username string) (bool, error)
	UpdateUser(ctx context.Context, id int64, updates map[string]interface{}) error
	UpdateUserField(ctx context.Context, id int64, field string, value interface{}) error
	DeleteUser(ctx context.Context, id int64) error
	SearchUsers(ctx context.Context, query string, limit int) ([]*User, error)

	// Auth token fields (verification, reset)
	GetByVerificationToken(ctx context.Context, token string) (*User, error)
	GetByResetToken(ctx context.Context, token string) (*User, error)

	// Identity (OAuth)
	GetIdentityByProviderUser(ctx context.Context, provider, providerUserID string) (*Identity, error)
	GetIdentity(ctx context.Context, userID int64, provider string) (*Identity, error)
	CreateIdentity(ctx context.Context, identity *Identity) error
	UpdateIdentityFields(ctx context.Context, userID int64, provider string, updates map[string]interface{}) error
	ListIdentities(ctx context.Context, userID int64) ([]*Identity, error)
	DeleteIdentity(ctx context.Context, userID int64, provider string) error

	// Git Credentials
	CreateGitCredential(ctx context.Context, credential *GitCredential) error
	GetGitCredentialWithProvider(ctx context.Context, userID, credentialID int64) (*GitCredential, error)
	ListGitCredentialsWithProvider(ctx context.Context, userID int64) ([]*GitCredential, error)
	UpdateGitCredential(ctx context.Context, credential *GitCredential, updates map[string]interface{}) error
	DeleteGitCredential(ctx context.Context, userID, credentialID int64) (int64, error)
	GitCredentialNameExists(ctx context.Context, userID int64, name string, excludeID *int64) (bool, error)
	ClearUserDefaultCredential(ctx context.Context, userID, credentialID int64) error
	SetDefaultGitCredential(ctx context.Context, userID, credentialID int64) error
	ClearAllDefaultGitCredentials(ctx context.Context, userID int64) error
	GetDefaultGitCredential(ctx context.Context, userID int64) (*GitCredential, error)

	// Repository Providers
	CreateRepositoryProvider(ctx context.Context, provider *RepositoryProvider) error
	GetRepositoryProvider(ctx context.Context, userID, providerID int64) (*RepositoryProvider, error)
	GetRepositoryProviderWithIdentity(ctx context.Context, userID, providerID int64) (*RepositoryProvider, error)
	GetRepositoryProviderByTypeAndURL(ctx context.Context, userID int64, providerType, baseURL string) (*RepositoryProvider, error)
	ListRepositoryProviders(ctx context.Context, userID int64) ([]*RepositoryProvider, error)
	UpdateRepositoryProvider(ctx context.Context, provider *RepositoryProvider, updates map[string]interface{}) error
	DeleteRepositoryProvider(ctx context.Context, userID, providerID int64) (int64, error)
	RepositoryProviderNameExists(ctx context.Context, userID int64, name string, excludeID *int64) (bool, error)
	GetRepositoryProviderByIdentityID(ctx context.Context, userID, identityID int64) (*RepositoryProvider, error)
	SetDefaultRepositoryProvider(ctx context.Context, userID, providerID int64) error
}
