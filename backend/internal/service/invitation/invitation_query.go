package invitation

import (
	"context"
	"time"

	invitationDomain "github.com/anthropics/agentsmesh/backend/internal/domain/invitation"
)

// GetByToken retrieves an invitation by its token
func (s *Service) GetByToken(ctx context.Context, token string) (*invitationDomain.Invitation, error) {
	inv, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return nil, ErrInvitationNotFound
	}
	return inv, nil
}

// GetByID retrieves an invitation by ID
func (s *Service) GetByID(ctx context.Context, id int64) (*invitationDomain.Invitation, error) {
	inv, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrInvitationNotFound
	}
	return inv, nil
}

// ListByOrganization lists all invitations for an organization
func (s *Service) ListByOrganization(ctx context.Context, orgID int64) ([]*invitationDomain.Invitation, error) {
	return s.repo.ListByOrganization(ctx, orgID)
}

// ListPendingByEmail lists all pending invitations for an email
func (s *Service) ListPendingByEmail(ctx context.Context, email string) ([]*invitationDomain.Invitation, error) {
	return s.repo.ListPendingByEmail(ctx, email)
}

// GetInvitationInfo returns information about an invitation for display
type InvitationInfo struct {
	ID          int64     `json:"id"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	OrgID       int64     `json:"organization_id"`
	OrgName     string    `json:"organization_name"`
	OrgSlug     string    `json:"organization_slug"`
	InviterName string    `json:"inviter_name"`
	ExpiresAt   time.Time `json:"expires_at"`
	IsExpired   bool      `json:"is_expired"`
}

// GetInvitationInfo retrieves detailed invitation info for display
func (s *Service) GetInvitationInfo(ctx context.Context, token string) (*InvitationInfo, error) {
	inv, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return nil, ErrInvitationNotFound
	}

	orgInfo, err := s.repo.GetOrganization(ctx, inv.OrganizationID)
	if err != nil {
		return nil, err
	}

	inviterName, err := s.repo.GetUserDisplayName(ctx, inv.InvitedBy)
	if err != nil {
		return nil, err
	}

	return &InvitationInfo{
		ID:          inv.ID,
		Email:       inv.Email,
		Role:        inv.Role,
		OrgID:       inv.OrganizationID,
		OrgName:     orgInfo.Name,
		OrgSlug:     orgInfo.Slug,
		InviterName: inviterName,
		ExpiresAt:   inv.ExpiresAt,
		IsExpired:   inv.IsExpired(),
	}, nil
}
