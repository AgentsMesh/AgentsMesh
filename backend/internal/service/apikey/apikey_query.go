package apikey

import (
	"context"
	"fmt"

	apikeyDomain "github.com/AgentsMesh/AgentsMesh/backend/internal/domain/apikey"
	"gorm.io/gorm"
)

const (
	// defaultListLimit is the default number of API keys returned per page
	defaultListLimit = 50
	// maxListLimit is the maximum number of API keys that can be requested per page
	maxListLimit = 200
)

// ListAPIKeys lists API keys for an organization with optional filtering
func (s *Service) ListAPIKeys(ctx context.Context, filter *ListAPIKeysFilter) ([]apikeyDomain.APIKey, int64, error) {
	var keys []apikeyDomain.APIKey
	var total int64

	query := s.db.WithContext(ctx).Model(&apikeyDomain.APIKey{}).
		Where("organization_id = ?", filter.OrganizationID)

	if filter.IsEnabled != nil {
		query = query.Where("is_enabled = ?", *filter.IsEnabled)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count api keys: %w", err)
	}

	query = query.Order("created_at DESC")

	// Apply pagination with sensible defaults
	limit := filter.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}
	query = query.Limit(limit)

	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	if err := query.Find(&keys).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list api keys: %w", err)
	}

	return keys, total, nil
}

// GetAPIKey retrieves a single API key by ID with organization ownership verification
func (s *Service) GetAPIKey(ctx context.Context, id int64, orgID int64) (*apikeyDomain.APIKey, error) {
	var key apikeyDomain.APIKey
	if err := s.db.WithContext(ctx).Where("id = ? AND organization_id = ?", id, orgID).First(&key).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("failed to get api key: %w", err)
	}
	return &key, nil
}
