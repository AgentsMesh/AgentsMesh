package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/pkg/slugkit"
)

// EnsureUniqueUsername coerces arbitrary external strings (OAuth login,
// email local-part, SAML attribute, OIDC preferred_username, LDAP uid, …)
// into a username satisfying slugkit.Validate AND unique within
// users.username. All external-data ingress paths MUST funnel through this
// helper instead of writing the field directly.
//
// seeds are tried in priority order; first sanitized form that GenerateUnique
// can place (incl. -2/-3 suffix retry) wins. Empty/whitespace seeds are
// skipped silently. If every seed exhausts retries, fall back to a random
// "user-{8hex}" handle.
func (s *Service) EnsureUniqueUsername(ctx context.Context, seeds []string) (string, error) {
	check := slugkit.FromExistsCheck(s.repo.UsernameExists)
	if u, ok := slugkit.TrySeeds(ctx, seeds, check); ok {
		return u, nil
	}
	return randomFallbackUsername(ctx, check)
}

func randomFallbackUsername(ctx context.Context, check slugkit.UniquenessChecker) (string, error) {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("random username seed: %w", err)
	}
	return slugkit.GenerateUnique(ctx, "user-"+hex.EncodeToString(buf), check)
}
