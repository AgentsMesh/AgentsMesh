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

type UserInfo struct {
	ExternalID string   // IdP subject / NameID / LDAP DN
	Email      string
	Username   string
	Name       string
	AvatarURL  string
	Groups     []string
}

type Provider interface {
	GetAuthURL(ctx context.Context, state string) (string, error)
	HandleCallback(ctx context.Context, params map[string]string) (*UserInfo, error)
	Authenticate(ctx context.Context, username, password string) (*UserInfo, error)
}
