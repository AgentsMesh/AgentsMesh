package sso

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	ssoprovider "github.com/anthropics/agentsmesh/backend/pkg/auth/sso"
	"gorm.io/gorm"
)

const (
	// samlRequestIDPrefix is the Redis key prefix for SAML AuthnRequest ID storage.
	// Key format: saml:reqid:{state} → requestID
	samlRequestIDPrefix = "saml:reqid:"
	// samlRequestIDTTL limits how long a SAML authentication flow can take.
	samlRequestIDTTL = 10 * time.Minute
)

// GetAuthURL returns the authorization URL for an OIDC or SAML provider.
// For SAML, it also stores the AuthnRequest ID in Redis (keyed by state)
// to enable InResponseTo validation on the ACS callback.
func (s *Service) GetAuthURL(ctx context.Context, domain string, protocol sso.Protocol, state string) (string, error) {
	cfg, err := s.repo.GetByDomainAndProtocol(ctx, strings.ToLower(domain), protocol)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrConfigNotFound
		}
		return "", fmt.Errorf("failed to query SSO config: %w", err)
	}
	if !cfg.IsEnabled {
		return "", fmt.Errorf("SSO config is disabled for domain %s", domain)
	}

	// SAML: use GetAuthURLWithRequestID to capture the AuthnRequest ID
	if protocol == sso.ProtocolSAML {
		samlProvider, err := s.buildSAMLProvider(cfg)
		if err != nil {
			return "", fmt.Errorf("failed to build SAML provider: %w", err)
		}
		authURL, requestID, err := samlProvider.GetAuthURLWithRequestID(ctx, state)
		if err != nil {
			return "", err
		}
		// Store the request ID in Redis for InResponseTo validation on ACS callback.
		// Non-fatal: if Redis is unavailable, SP-initiated flow still works
		// (AllowIDPInitiated=true accepts responses without InResponseTo match),
		// but we lose the replay protection for SP-initiated flows.
		if err := s.storeSAMLRequestID(ctx, state, requestID); err != nil {
			slog.Warn("failed to store SAML request ID for InResponseTo validation",
				"domain", domain, "error", err)
		}
		return authURL, nil
	}

	provider, err := s.buildProvider(ctx, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to build SSO provider: %w", err)
	}

	return provider.GetAuthURL(ctx, state)
}

// HandleCallback processes the IdP callback and returns user info.
// For SAML, if params contains "RelayState", the stored AuthnRequest ID is
// retrieved from Redis and passed to the provider for InResponseTo validation.
func (s *Service) HandleCallback(ctx context.Context, domain string, protocol sso.Protocol, params map[string]string) (*ssoprovider.UserInfo, int64, error) {
	cfg, err := s.repo.GetByDomainAndProtocol(ctx, strings.ToLower(domain), protocol)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, ErrConfigNotFound
		}
		return nil, 0, fmt.Errorf("failed to query SSO config: %w", err)
	}
	if !cfg.IsEnabled {
		return nil, 0, fmt.Errorf("SSO config is disabled for domain %s", domain)
	}

	// For SAML SP-initiated flows, retrieve the stored AuthnRequest ID
	// so the provider can validate InResponseTo in the SAML response.
	if protocol == sso.ProtocolSAML {
		if relayState := params["RelayState"]; relayState != "" {
			if requestID, err := s.retrieveSAMLRequestID(ctx, relayState); err == nil && requestID != "" {
				params["possibleRequestIDs"] = requestID
			}
			// Non-fatal: if retrieval fails (Redis down, expired, etc.),
			// AllowIDPInitiated=true still accepts the response.
		}
	}

	provider, err := s.buildProvider(ctx, cfg)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to build SSO provider: %w", err)
	}

	userInfo, err := provider.HandleCallback(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("SSO callback failed: %w", err)
	}
	if userInfo == nil {
		return nil, 0, fmt.Errorf("SSO callback returned no user info")
	}

	return userInfo, cfg.ID, nil
}

// AuthenticateLDAP performs LDAP authentication
func (s *Service) AuthenticateLDAP(ctx context.Context, domain, username, password string) (*ssoprovider.UserInfo, int64, error) {
	cfg, err := s.repo.GetByDomainAndProtocol(ctx, strings.ToLower(domain), sso.ProtocolLDAP)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, ErrConfigNotFound
		}
		return nil, 0, fmt.Errorf("failed to query SSO config: %w", err)
	}
	if !cfg.IsEnabled {
		return nil, 0, fmt.Errorf("SSO config is disabled for domain %s", domain)
	}

	provider, err := s.buildProvider(ctx, cfg)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to build LDAP provider: %w", err)
	}

	userInfo, err := provider.Authenticate(ctx, username, password)
	if err != nil {
		return nil, 0, fmt.Errorf("LDAP authentication failed: %w", err)
	}
	if userInfo == nil {
		return nil, 0, fmt.Errorf("LDAP authentication returned no user info")
	}

	return userInfo, cfg.ID, nil
}

// IsPasswordLoginAllowed checks if password login is allowed for an email
// Returns (allowed, error). If enforce_sso is active and user is not system_admin, returns false.
func (s *Service) IsPasswordLoginAllowed(ctx context.Context, email string, isSystemAdmin bool) (bool, error) {
	// System admins are always allowed to use password login
	if isSystemAdmin {
		return true, nil
	}

	domain := extractDomain(email)
	if domain == "" {
		return true, nil
	}

	enforced, err := s.repo.HasEnforcedSSO(ctx, domain)
	if err != nil {
		// On error, allow login (fail-open to prevent lockout)
		return true, nil
	}

	return !enforced, nil
}

// TestConnection tests connectivity to the SSO provider
func (s *Service) TestConnection(ctx context.Context, id int64) error {
	cfg, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrConfigNotFound
		}
		return fmt.Errorf("failed to query SSO config: %w", err)
	}

	switch cfg.Protocol {
	case sso.ProtocolOIDC:
		return s.testOIDCConnection(ctx, cfg)
	case sso.ProtocolSAML:
		return s.testSAMLConnection(cfg)
	case sso.ProtocolLDAP:
		return s.testLDAPConnection(cfg)
	default:
		return ErrInvalidProtocol
	}
}

// GetSAMLMetadata returns the SP metadata XML for a domain's SAML config
func (s *Service) GetSAMLMetadata(ctx context.Context, domain string) ([]byte, error) {
	cfg, err := s.repo.GetByDomainAndProtocol(ctx, strings.ToLower(domain), sso.ProtocolSAML)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrConfigNotFound
		}
		return nil, fmt.Errorf("failed to query SSO config: %w", err)
	}

	provider, err := s.buildSAMLProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build SAML provider: %w", err)
	}

	return provider.GenerateMetadata()
}

// SSOProviderName returns the provider name string used in user_identities
func SSOProviderName(protocol sso.Protocol, configID int64) string {
	return fmt.Sprintf("sso_%s_%d", protocol, configID)
}

// storeSAMLRequestID stores a SAML AuthnRequest ID in Redis, keyed by the
// RelayState (our state parameter). This enables InResponseTo validation
// when the ACS callback arrives.
func (s *Service) storeSAMLRequestID(ctx context.Context, state, requestID string) error {
	if s.redis == nil {
		return nil // graceful degradation: skip tracking when Redis is not configured
	}
	key := samlRequestIDPrefix + state
	return s.redis.Set(ctx, key, requestID, samlRequestIDTTL).Err()
}

// retrieveSAMLRequestID retrieves and deletes the stored AuthnRequest ID
// for a given RelayState. Uses GetDel for atomicity (single-use).
func (s *Service) retrieveSAMLRequestID(ctx context.Context, state string) (string, error) {
	if s.redis == nil {
		return "", nil
	}
	key := samlRequestIDPrefix + state
	return s.redis.GetDel(ctx, key).Result()
}

// extractDomain extracts the domain from an email address
func extractDomain(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(parts[1])
}
