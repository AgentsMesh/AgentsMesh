package invitation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/invitation"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/organization"
)

// CreateRequest represents an invitation creation request
type CreateRequest struct {
	OrganizationID int64
	Email          string
	Role           string
	InviterID      int64
	InviterName    string
	OrgName        string
}

// Create creates a new invitation and sends an email
func (s *Service) Create(ctx context.Context, req *CreateRequest) (*invitation.Invitation, error) {
	// Validate role
	if req.Role != organization.RoleAdmin && req.Role != organization.RoleMember {
		return nil, ErrInvalidRole
	}

	// Check if user is already a member
	var existingMember organization.Member
	err := s.db.WithContext(ctx).Where("organization_id = ? AND user_id IN (SELECT id FROM users WHERE email = ?)",
		req.OrganizationID, req.Email).First(&existingMember).Error
	if err == nil {
		return nil, ErrAlreadyMember
	}

	// Check for existing pending invitation
	existing, err := s.repo.GetByOrgAndEmail(req.OrganizationID, req.Email)
	if err == nil && existing.IsPending() {
		return nil, ErrPendingInvitation
	}

	// Generate unique token
	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	inv := &invitation.Invitation{
		OrganizationID: req.OrganizationID,
		Email:          req.Email,
		Role:           req.Role,
		Token:          token,
		InvitedBy:      req.InviterID,
		ExpiresAt:      time.Now().AddDate(0, 0, InvitationValidDays),
	}

	if err := s.repo.Create(inv); err != nil {
		return nil, err
	}

	// Send invitation email
	if s.emailService != nil {
		if err := s.emailService.SendOrgInvitationEmail(ctx, req.Email, req.OrgName, req.InviterName, token); err != nil {
			// Log error but don't fail the invitation creation
			// The invitation can still be accessed via the token
		}
	}

	return inv, nil
}

// Resend resends an invitation email
func (s *Service) Resend(ctx context.Context, invitationID int64, inviterName, orgName string) error {
	inv, err := s.repo.GetByID(invitationID)
	if err != nil {
		return ErrInvitationNotFound
	}

	if inv.IsAccepted() {
		return ErrInvitationAccepted
	}

	// Extend expiration if needed
	if inv.IsExpired() || time.Until(inv.ExpiresAt) < 24*time.Hour {
		inv.ExpiresAt = time.Now().AddDate(0, 0, InvitationValidDays)
		if err := s.repo.Update(inv); err != nil {
			return err
		}
	}

	// Send email
	if s.emailService != nil {
		return s.emailService.SendOrgInvitationEmail(ctx, inv.Email, orgName, inviterName, inv.Token)
	}

	return nil
}

// generateToken generates a secure random token
func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
