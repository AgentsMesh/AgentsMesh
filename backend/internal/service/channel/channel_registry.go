package channel

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/pkg/slugkit"
)

// EnsureUniqueSlug derives a slug for a new channel from its display name.
// Channels created before Phase 2 left this column NULL; backfill program
// populates them. New channel creates MUST funnel through this helper
// before calling repo.Create.
func (s *Service) EnsureUniqueSlug(ctx context.Context, orgID int64, nameSeed string) (string, error) {
	check := slugkit.FromExistsCheck(func(ctx context.Context, candidate string) (bool, error) {
		return s.repo.SlugExists(ctx, orgID, candidate)
	})
	return slugkit.GenerateUnique(ctx, nameSeed, check)
}
