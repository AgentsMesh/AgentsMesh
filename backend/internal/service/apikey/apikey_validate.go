package apikey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	apikeyDomain "github.com/anthropics/agentsmesh/backend/internal/domain/apikey"
)

const (
	cachePrefix = "apikey:hash:"
	cacheTTL    = 5 * time.Minute
)

// cachedKeyData stores the full validation data needed for cache re-checks.
// Includes mutable fields (IsEnabled, ExpiresAt) to detect stale cache entries.
type cachedKeyData struct {
	APIKeyID       int64      `json:"api_key_id"`
	OrganizationID int64      `json:"organization_id"`
	CreatedBy      int64      `json:"created_by"`
	Scopes         []string   `json:"scopes"`
	KeyName        string     `json:"key_name"`
	IsEnabled      bool       `json:"is_enabled"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
}

// ValidateKey validates an API key and returns associated information.
// Uses Redis cache to reduce DB queries on high-frequency calls.
func (s *Service) ValidateKey(ctx context.Context, rawKey string) (*ValidateResult, error) {
	// Hash the raw key
	hashBytes := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hashBytes[:])

	// Try Redis cache first
	if cached, err := s.getFromCache(ctx, keyHash); err == nil && cached != nil {
		// Re-validate mutable state from cached data
		if !cached.IsEnabled {
			return nil, ErrAPIKeyDisabled
		}
		if cached.ExpiresAt != nil && time.Now().After(*cached.ExpiresAt) {
			// Key expired since caching — invalidate and return error
			s.invalidateCache(ctx, keyHash)
			return nil, ErrAPIKeyExpired
		}
		return &ValidateResult{
			APIKeyID:       cached.APIKeyID,
			OrganizationID: cached.OrganizationID,
			CreatedBy:      cached.CreatedBy,
			Scopes:         cached.Scopes,
			KeyName:        cached.KeyName,
		}, nil
	}

	// Cache miss: query DB
	key, err := s.repo.GetByKeyHash(ctx, keyHash)
	if err != nil {
		if err == apikeyDomain.ErrNotFound {
			return nil, ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("failed to validate api key: %w", err)
	}

	// Always cache the DB result (including disabled/expired state)
	// so that cache re-validation works correctly
	cached := &cachedKeyData{
		APIKeyID:       key.ID,
		OrganizationID: key.OrganizationID,
		CreatedBy:      key.CreatedBy,
		Scopes:         key.Scopes.ToStrings(),
		KeyName:        key.Name,
		IsEnabled:      key.IsEnabled,
		ExpiresAt:      key.ExpiresAt,
	}
	s.setCache(ctx, keyHash, cached)

	if !key.IsEnabled {
		return nil, ErrAPIKeyDisabled
	}
	if key.IsExpired() {
		return nil, ErrAPIKeyExpired
	}

	return &ValidateResult{
		APIKeyID:       key.ID,
		OrganizationID: key.OrganizationID,
		CreatedBy:      key.CreatedBy,
		Scopes:         key.Scopes.ToStrings(),
		KeyName:        key.Name,
	}, nil
}

// getFromCache retrieves cached key data from Redis
func (s *Service) getFromCache(ctx context.Context, keyHash string) (*cachedKeyData, error) {
	if s.redisClient == nil {
		return nil, fmt.Errorf("redis not available")
	}

	data, err := s.redisClient.Get(ctx, cachePrefix+keyHash).Bytes()
	if err != nil {
		return nil, err
	}

	var cached cachedKeyData
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, err
	}

	return &cached, nil
}

// setCache stores key data in Redis
func (s *Service) setCache(ctx context.Context, keyHash string, cached *cachedKeyData) {
	if s.redisClient == nil {
		return
	}

	data, err := json.Marshal(cached)
	if err != nil {
		slog.Warn("Failed to marshal api key cache", "error", err)
		return
	}

	if err := s.redisClient.Set(ctx, cachePrefix+keyHash, data, cacheTTL).Err(); err != nil {
		slog.Warn("Failed to set api key cache", "error", err)
	}
}

// invalidateCache removes cached key data from Redis
func (s *Service) invalidateCache(ctx context.Context, keyHash string) {
	if s.redisClient == nil {
		return
	}

	if err := s.redisClient.Del(ctx, cachePrefix+keyHash).Err(); err != nil {
		slog.Warn("Failed to invalidate api key cache", "error", err)
	}
}
