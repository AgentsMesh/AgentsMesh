package apikey

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/pkg/slugkit"
)

// EnsureUniqueSlug derives an org-scoped slug from the API key's display name.
// Pre-Phase-2 rows left this column NULL; the backfill program fills them.
// New creates MUST funnel through this helper before repo.Create.
func (s *Service) EnsureUniqueSlug(ctx context.Context, orgID int64, nameSeed string) (string, error) {
	check := slugkit.FromExistsCheck(func(ctx context.Context, candidate string) (bool, error) {
		return s.repo.SlugExists(ctx, orgID, candidate)
	})
	return slugkit.GenerateUnique(ctx, nameSeed, check)
}
