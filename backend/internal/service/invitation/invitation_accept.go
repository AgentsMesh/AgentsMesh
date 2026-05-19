package invitation

import (
	"context"

	invitationDomain "github.com/anthropics/agentsmesh/backend/internal/domain/invitation"
)

type AcceptResult struct {
	Organization *AcceptOrgInfo
}

type AcceptOrgInfo struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (s *Service) Accept(ctx context.Context, token string, userID int64) (*AcceptResult, error) {
	inv, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return nil, ErrInvitationNotFound
	}

	if inv.IsAccepted() {
		return nil, ErrInvitationAccepted
	}

	if inv.IsExpired() {
		return nil, ErrInvitationExpired
	}

	exists, err := s.repo.CheckMemberExists(ctx, inv.OrganizationID, userID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrAlreadyMember
	}

	// Atomically add member and mark invitation as accepted
	_, err = s.repo.AcceptInvitationAtomic(ctx, &invitationDomain.AcceptInvitationParams{
		Invitation: inv,
		UserID:     userID,
		Role:       inv.Role,
	})
	if err != nil {
		return nil, err
	}

	orgInfo, err := s.repo.GetOrganization(ctx, inv.OrganizationID)
	if err != nil {
		return nil, err
	}

	return &AcceptResult{
		Organization: &AcceptOrgInfo{
			ID:   inv.OrganizationID,
			Name: orgInfo.Name,
			Slug: orgInfo.Slug,
		},
	}, nil
}

func (s *Service) Revoke(ctx context.Context, invitationID int64) error {
	inv, err := s.repo.GetByID(ctx, invitationID)
	if err != nil {
		return ErrInvitationNotFound
	}

	if inv.IsAccepted() {
		return ErrInvitationAccepted
	}

	return s.repo.Delete(ctx, invitationID)
}

func (s *Service) CleanupExpired(ctx context.Context) error {
	return s.repo.DeleteExpired(ctx)
}
