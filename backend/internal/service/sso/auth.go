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
	samlRequestIDPrefix = "saml:reqid:"
	samlRequestIDTTL = 10 * time.Minute
)

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

	if protocol == sso.ProtocolSAML {
		samlProvider, err := s.buildSAMLProvider(cfg)
		if err != nil {
			return "", fmt.Errorf("failed to build SAML provider: %w", err)
		}
		authURL, requestID, err := samlProvider.GetAuthURLWithRequestID(ctx, state)
		if err != nil {
			return "", err
		}
		if err := s.storeSAMLRequestID(ctx, state, requestID); err != nil {
			slog.WarnContext(ctx, "failed to store SAML request ID for InResponseTo validation",
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

	if protocol == sso.ProtocolSAML {
		if relayState := params["RelayState"]; relayState != "" {
			if requestID, err := s.retrieveSAMLRequestID(ctx, relayState); err == nil && requestID != "" {
				params["possibleRequestIDs"] = requestID
			}
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

func (s *Service) IsPasswordLoginAllowed(ctx context.Context, email string, isSystemAdmin bool) (bool, error) {
	if isSystemAdmin {
		return true, nil
	}

	domain := extractDomain(email)
	if domain == "" {
		return true, nil
	}

	enforced, err := s.repo.HasEnforcedSSO(ctx, domain)
	if err != nil {
		return true, nil
	}

	return !enforced, nil
}

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

func SSOProviderName(protocol sso.Protocol, configID int64) string {
	return fmt.Sprintf("sso_%s_%d", protocol, configID)
}

func (s *Service) storeSAMLRequestID(ctx context.Context, state, requestID string) error {
	if s.redis == nil {
		return nil // graceful degradation: skip tracking when Redis is not configured
	}
	key := samlRequestIDPrefix + state
	return s.redis.Set(ctx, key, requestID, samlRequestIDTTL).Err()
}

func (s *Service) retrieveSAMLRequestID(ctx context.Context, state string) (string, error) {
	if s.redis == nil {
		return "", nil
	}
	key := samlRequestIDPrefix + state
	return s.redis.GetDel(ctx, key).Result()
}

func extractDomain(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(parts[1])
}
