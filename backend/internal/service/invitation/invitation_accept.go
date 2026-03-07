package invitation

import (
	"context"

	invitationDomain "github.com/anthropics/agentsmesh/backend/internal/domain/invitation"
)

// AcceptResult contains the result of accepting an invitation
type AcceptResult struct {
	Organization *AcceptOrgInfo
}

// AcceptOrgInfo contains organization info returned on invitation acceptance
type AcceptOrgInfo struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Accept accepts an invitation and adds the user as a member
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

	// Check if user is already a member
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

	// Fetch org info for the response
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

// Revoke revokes a pending invitation
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

// CleanupExpired removes expired invitations
func (s *Service) CleanupExpired(ctx context.Context) error {
	return s.repo.DeleteExpired(ctx)
}
