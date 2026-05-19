package organization

import (
	"context"
	"errors"
	"fmt"

	orgDomain "github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/pkg/slugkit"
)

func (s *Service) CreatePersonal(ctx context.Context, ownerID int64, username, displayName string) (*orgDomain.Organization, error) {
	fallbackSlug := fmt.Sprintf("user-%d-workspace", ownerID)
	seeds := personalSlugSeeds(username, fallbackSlug)
	check := slugkit.FromExistsCheck(s.repo.SlugExists)
	name := personalWorkspaceName(displayName, username)

	var lastErr error
	for i, seed := range seeds {
		isFinal := i == len(seeds)-1

		finalSlug, err := slugkit.GenerateUnique(ctx, seed, check)
		if err != nil {
			lastErr = err
			if isFinal {
				return nil, err
			}
			continue
		}

		org, err := s.Create(ctx, ownerID, &CreateRequest{
			Name: name,
			Slug: finalSlug,
		})
		if err == nil {
			return org, nil
		}
		if !errors.Is(err, ErrSlugAlreadyExists) {
			return nil, err
		}
		lastErr = err
	}
	return nil, lastErr
}

func personalSlugSeeds(username, fallbackSlug string) []string {
	sanitized := slugkit.Sanitize(username)
	if sanitized == "" {
		return []string{fallbackSlug}
	}
	rawSlug := fmt.Sprintf("%s-workspace", sanitized)
	return []string{rawSlug, rawSlug, fallbackSlug}
}

func personalWorkspaceName(displayName, username string) string {
	name := displayName
	if name == "" {
		name = username
	}
	return fmt.Sprintf("%s's Workspace", name)
}
