package auth

import (
	"context"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
)

type SSOLoginRequest struct {
	ProviderName string
	ExternalID   string
	Username     string
	Email        string
	Name         string
	AvatarURL    string
}

// SSOLogin authenticates a user via SSO identity, records the login, and returns tokens.
// It mirrors OAuthLogin but for SSO providers (LDAP, SAML, OIDC).
func (s *Service) SSOLogin(ctx context.Context, req *SSOLoginRequest) (*user.User, *TokenPair, error) {
	u, _, err := s.userService.GetOrCreateByOAuth(ctx, req.ProviderName, req.ExternalID, req.Username, req.Email, req.Name, req.AvatarURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create/get user: %w", err)
	}

	if !u.IsActive {
		return nil, nil, ErrUserDisabled
	}

	s.userService.RecordLogin(ctx, u.ID)

	tokens, err := s.GenerateTokenPair(u, 0, "")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return u, tokens, nil
}
