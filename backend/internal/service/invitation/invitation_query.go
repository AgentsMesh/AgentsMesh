package invitation

import (
	"context"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/invitation"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/organization"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
)

// GetByToken retrieves an invitation by its token
func (s *Service) GetByToken(ctx context.Context, token string) (*invitation.Invitation, error) {
	inv, err := s.repo.GetByToken(token)
	if err != nil {
		return nil, ErrInvitationNotFound
	}
	return inv, nil
}

// GetByID retrieves an invitation by ID
func (s *Service) GetByID(ctx context.Context, id int64) (*invitation.Invitation, error) {
	inv, err := s.repo.GetByID(id)
	if err != nil {
		return nil, ErrInvitationNotFound
	}
	return inv, nil
}

// ListByOrganization lists all invitations for an organization
func (s *Service) ListByOrganization(ctx context.Context, orgID int64) ([]*invitation.Invitation, error) {
	return s.repo.ListByOrganization(orgID)
}

// ListPendingByEmail lists all pending invitations for an email
func (s *Service) ListPendingByEmail(ctx context.Context, email string) ([]*invitation.Invitation, error) {
	return s.repo.ListPendingByEmail(email)
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
	inv, err := s.repo.GetByToken(token)
	if err != nil {
		return nil, ErrInvitationNotFound
	}

	var org organization.Organization
	if err := s.db.WithContext(ctx).First(&org, inv.OrganizationID).Error; err != nil {
		return nil, err
	}

	var inviter user.User
	if err := s.db.WithContext(ctx).First(&inviter, inv.InvitedBy).Error; err != nil {
		return nil, err
	}

	inviterName := inviter.Username
	if inviter.Name != nil && *inviter.Name != "" {
		inviterName = *inviter.Name
	}

	return &InvitationInfo{
		ID:          inv.ID,
		Email:       inv.Email,
		Role:        inv.Role,
		OrgID:       org.ID,
		OrgName:     org.Name,
		OrgSlug:     org.Slug,
		InviterName: inviterName,
		ExpiresAt:   inv.ExpiresAt,
		IsExpired:   inv.IsExpired(),
	}, nil
}
