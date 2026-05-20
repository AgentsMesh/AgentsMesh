package apikey

import (
	"errors"
	"context"
	"fmt"

	apikeyDomain "github.com/anthropics/agentsmesh/backend/internal/domain/apikey"
)

const (
	defaultListLimit = 50
	maxListLimit = 200
)

func (s *Service) ListAPIKeys(ctx context.Context, filter *ListAPIKeysFilter) ([]apikeyDomain.APIKey, int64, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	return s.repo.List(ctx, filter.OrganizationID, filter.IsEnabled, limit, filter.Offset)
}

func (s *Service) GetAPIKey(ctx context.Context, id int64, orgID int64) (*apikeyDomain.APIKey, error) {
	key, err := s.repo.GetByID(ctx, id, orgID)
	if err != nil {
		if errors.Is(err, apikeyDomain.ErrNotFound) {
			return nil, ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("failed to get api key: %w", err)
	}
	return key, nil
}

// GetAPIKeyBySlug is the post-Phase-4 lookup-by-identifier path. Prefer this
// for new callers; GetAPIKey by integer ID remains for back-compat with
// pre-slug REST routes.
func (s *Service) GetAPIKeyBySlug(ctx context.Context, orgID int64, slug string) (*apikeyDomain.APIKey, error) {
	key, err := s.repo.GetByOrgAndSlug(ctx, orgID, slug)
	if err != nil {
		if errors.Is(err, apikeyDomain.ErrNotFound) {
			return nil, ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("failed to get api key by slug: %w", err)
	}
	return key, nil
}
