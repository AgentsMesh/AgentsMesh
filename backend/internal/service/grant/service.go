package grant

import (
	"context"
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/grant"
)

var (
	ErrGrantNotFound = errors.New("grant not found")
	ErrSelfGrant     = errors.New("cannot grant access to yourself")
	ErrInvalidType   = errors.New("invalid resource type")
)

type Service struct {
	repo grant.Repository
}

func NewService(repo grant.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GrantAccess(ctx context.Context, orgID int64, resourceType, resourceID string, userID, grantedBy int64) (*grant.ResourceGrant, error) {
	if userID == grantedBy {
		return nil, ErrSelfGrant
	}
	if !isValidType(resourceType) {
		return nil, ErrInvalidType
	}

	g := &grant.ResourceGrant{
		OrganizationID: orgID,
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		UserID:         userID,
		GrantedBy:      grantedBy,
		CreatedAt:      time.Now(),
	}
	if err := s.repo.Create(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

func (s *Service) RevokeAccess(ctx context.Context, resourceType, resourceID string, grantID int64) error {
	return s.repo.Delete(ctx, resourceType, resourceID, grantID)
}

func (s *Service) ListGrants(ctx context.Context, resourceType, resourceID string) ([]*grant.ResourceGrant, error) {
	return s.repo.ListByResource(ctx, resourceType, resourceID)
}

func (s *Service) GetGrantedUserIDs(ctx context.Context, resourceType, resourceID string) ([]int64, error) {
	return s.repo.GetGrantedUserIDs(ctx, resourceType, resourceID)
}

func (s *Service) GetGrantedResourceIDs(ctx context.Context, resourceType string, userID int64, orgID int64) ([]string, error) {
	return s.repo.GetGrantedResourceIDs(ctx, resourceType, userID, orgID)
}

func (s *Service) CleanupByResource(ctx context.Context, resourceType, resourceID string) error {
	return s.repo.DeleteByResource(ctx, resourceType, resourceID)
}

func isValidType(t string) bool {
	return t == grant.TypePod || t == grant.TypeRunner || t == grant.TypeRepository
}
