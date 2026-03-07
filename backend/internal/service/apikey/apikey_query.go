package apikey

import (
	"context"
	"fmt"

	apikeyDomain "github.com/anthropics/agentsmesh/backend/internal/domain/apikey"
)

const (
	// defaultListLimit is the default number of API keys returned per page
	defaultListLimit = 50
	// maxListLimit is the maximum number of API keys that can be requested per page
	maxListLimit = 200
)

// ListAPIKeys lists API keys for an organization with optional filtering
func (s *Service) ListAPIKeys(ctx context.Context, filter *ListAPIKeysFilter) ([]apikeyDomain.APIKey, int64, error) {
	// Apply pagination with sensible defaults
	limit := filter.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	return s.repo.List(ctx, filter.OrganizationID, filter.IsEnabled, limit, filter.Offset)
}

// GetAPIKey retrieves a single API key by ID with organization ownership verification
func (s *Service) GetAPIKey(ctx context.Context, id int64, orgID int64) (*apikeyDomain.APIKey, error) {
	key, err := s.repo.GetByID(ctx, id, orgID)
	if err != nil {
		if err == apikeyDomain.ErrNotFound {
			return nil, ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("failed to get api key: %w", err)
	}
	return key, nil
}
