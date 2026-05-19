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

type Service struct {
	repo          sso.Repository
	encryptionKey string
	config        *config.Config
	redis         *redis.Client // optional, enables SAML request ID tracking

	providerFactory     func(ctx context.Context, cfg *sso.Config) (ssoprovider.Provider, error)
	samlProviderFactory func(cfg *sso.Config) (*ssoprovider.SAMLProvider, error)
}

func NewService(repo sso.Repository, encryptionKey string, cfg *config.Config) *Service {
	return &Service{
		repo:          repo,
		encryptionKey: encryptionKey,
		config:        cfg,
	}
}

func NewServiceWithRedis(repo sso.Repository, encryptionKey string, cfg *config.Config, redisClient *redis.Client) *Service {
	return &Service{
		repo:          repo,
		encryptionKey: encryptionKey,
		config:        cfg,
		redis:         redisClient,
	}
}

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

func (s *Service) DeleteConfig(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrConfigNotFound
		}
		slog.ErrorContext(ctx, "failed to delete SSO config", "config_id", id, "error", err)
		return fmt.Errorf("failed to delete SSO config: %w", err)
	}
	slog.InfoContext(ctx, "SSO config deleted", "config_id", id)
	return nil
}

func (s *Service) GetEnabledConfigs(ctx context.Context, domain string) ([]*sso.Config, error) {
	return s.repo.GetEnabledByDomain(ctx, strings.ToLower(domain))
}

func (s *Service) HasEnforcedSSO(ctx context.Context, domain string) (bool, error) {
	return s.repo.HasEnforcedSSO(ctx, strings.ToLower(domain))
}

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
