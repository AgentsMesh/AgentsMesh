package sso

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/sso"
	ssoprovider "github.com/anthropics/agentsmesh/backend/pkg/auth/sso"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Service handles SSO configuration management
type Service struct {
	repo          sso.Repository
	encryptionKey string
	config        *config.Config
	redis         *redis.Client // optional, enables SAML request ID tracking

	// Testing hooks — override provider construction (nil in production).
	// These enable unit testing auth flows without real IdP connections.
	providerFactory     func(ctx context.Context, cfg *sso.Config) (ssoprovider.Provider, error)
	samlProviderFactory func(cfg *sso.Config) (*ssoprovider.SAMLProvider, error)
}

// NewService creates a new SSO service (without Redis — SAML request ID tracking disabled)
func NewService(repo sso.Repository, encryptionKey string, cfg *config.Config) *Service {
	return &Service{
		repo:          repo,
		encryptionKey: encryptionKey,
		config:        cfg,
	}
}

// NewServiceWithRedis creates a new SSO service with Redis for SAML request ID tracking
func NewServiceWithRedis(repo sso.Repository, encryptionKey string, cfg *config.Config, redisClient *redis.Client) *Service {
	return &Service{
		repo:          repo,
		encryptionKey: encryptionKey,
		config:        cfg,
		redis:         redisClient,
	}
}

// GetConfig returns an SSO config by ID
func (s *Service) GetConfig(ctx context.Context, id int64) (*sso.Config, error) {
	cfg, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrConfigNotFound
		}
		return nil, fmt.Errorf("failed to get SSO config: %w", err)
	}
	return cfg, nil
}

// ListConfigs returns all SSO configs with pagination and optional filtering
func (s *Service) ListConfigs(ctx context.Context, query *sso.ListQuery, page, pageSize int) ([]*sso.Config, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	return s.repo.List(ctx, query, offset, pageSize)
}

// DeleteConfig deletes an SSO configuration
func (s *Service) DeleteConfig(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrConfigNotFound
		}
		slog.Error("failed to delete SSO config", "config_id", id, "error", err)
		return fmt.Errorf("failed to delete SSO config: %w", err)
	}
	slog.Info("SSO config deleted", "config_id", id)
	return nil
}

// GetEnabledConfigs returns enabled SSO configs for a domain (for discover API)
func (s *Service) GetEnabledConfigs(ctx context.Context, domain string) ([]*sso.Config, error) {
	return s.repo.GetEnabledByDomain(ctx, strings.ToLower(domain))
}

// HasEnforcedSSO checks if a domain has enforce_sso enabled
func (s *Service) HasEnforcedSSO(ctx context.Context, domain string) (bool, error) {
	return s.repo.HasEnforcedSSO(ctx, strings.ToLower(domain))
}

// DecryptSecret decrypts an encrypted field (helper for provider creation)
func (s *Service) DecryptSecret(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}
	decrypted, err := crypto.DecryptWithKey(encrypted, s.encryptionKey)
	if err != nil {
		slog.Error("failed to decrypt SSO secret", "error", err)
		return "", err
	}
	return decrypted, nil
}
