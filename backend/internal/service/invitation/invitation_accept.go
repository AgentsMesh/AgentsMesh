package invitation

import (
	"context"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/organization"
	"gorm.io/gorm"
)

// AcceptResult contains the result of accepting an invitation
type AcceptResult struct {
	Organization *organization.Organization
	Member       *organization.Member
}

// Accept accepts an invitation and adds the user as a member
func (s *Service) Accept(ctx context.Context, token string, userID int64) (*AcceptResult, error) {
	inv, err := s.repo.GetByToken(token)
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
	var existingMember organization.Member
	err = s.db.WithContext(ctx).Where("organization_id = ? AND user_id = ?",
		inv.OrganizationID, userID).First(&existingMember).Error
	if err == nil {
		return nil, ErrAlreadyMember
	}

	var org organization.Organization
	var member *organization.Member

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get organization
		if err := tx.First(&org, inv.OrganizationID).Error; err != nil {
			return err
		}

		// Add user as member
		member = &organization.Member{
			OrganizationID: inv.OrganizationID,
			UserID:         userID,
			Role:           inv.Role,
		}
		if err := tx.Create(member).Error; err != nil {
			return err
		}

		// Mark invitation as accepted
		now := time.Now()
		inv.AcceptedAt = &now
		return tx.Save(inv).Error
	})

	if err != nil {
		return nil, err
	}

	return &AcceptResult{
		Organization: &org,
		Member:       member,
	}, nil
}

// Revoke revokes a pending invitation
func (s *Service) Revoke(ctx context.Context, invitationID int64) error {
	inv, err := s.repo.GetByID(invitationID)
	if err != nil {
		return ErrInvitationNotFound
	}

	if inv.IsAccepted() {
		return ErrInvitationAccepted
	}

	return s.repo.Delete(invitationID)
}

// CleanupExpired removes expired invitations
func (s *Service) CleanupExpired(ctx context.Context) error {
	return s.repo.DeleteExpired()
}
