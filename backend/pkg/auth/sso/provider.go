package sso

import (
	"context"
	"errors"
)

var (
	ErrNotSupported  = errors.New("operation not supported for this protocol")
	ErrAuthFailed    = errors.New("authentication failed")
	ErrInvalidConfig = errors.New("invalid SSO configuration")
)

// UserInfo represents authenticated user information from an SSO provider
type UserInfo struct {
	ExternalID string   // IdP subject / NameID / LDAP DN
	Email      string
	Username   string
	Name       string
	AvatarURL  string
	Groups     []string // Reserved for future use
}

// Provider defines the interface for SSO authentication providers
type Provider interface {
	// GetAuthURL returns the IdP authorization URL (OIDC/SAML only)
	GetAuthURL(ctx context.Context, state string) (string, error)
	// HandleCallback processes the IdP callback response (OIDC/SAML only)
	HandleCallback(ctx context.Context, params map[string]string) (*UserInfo, error)
	// Authenticate performs direct authentication (LDAP only)
	Authenticate(ctx context.Context, username, password string) (*UserInfo, error)
}
