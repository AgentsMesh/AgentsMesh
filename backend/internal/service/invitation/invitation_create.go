package invitation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	invitationDomain "github.com/anthropics/agentsmesh/backend/internal/domain/invitation"
	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
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
func (s *Service) Create(ctx context.Context, req *CreateRequest) (*invitationDomain.Invitation, error) {
	// Validate role
	if req.Role != organization.RoleAdmin && req.Role != organization.RoleMember {
		return nil, ErrInvalidRole
	}

	// Check if user is already a member (by email)
	exists, err := s.repo.CheckMemberExistsByEmail(ctx, req.OrganizationID, req.Email)
	if err != nil {
		slog.Error("failed to check member existence", "org_id", req.OrganizationID, "email", req.Email, "error", err)
		return nil, err
	}
	if exists {
		return nil, ErrAlreadyMember
	}

	// Check for existing pending invitation
	existing, err := s.repo.GetByOrgAndEmail(ctx, req.OrganizationID, req.Email)
	if err == nil && existing.IsPending() {
		return nil, ErrPendingInvitation
	}

	// Generate unique token
	token, err := generateToken()
	if err != nil {
		slog.Error("failed to generate invitation token", "error", err)
		return nil, err
	}

	inv := &invitationDomain.Invitation{
		OrganizationID: req.OrganizationID,
		Email:          req.Email,
		Role:           req.Role,
		Token:          token,
		InvitedBy:      req.InviterID,
		ExpiresAt:      time.Now().AddDate(0, 0, InvitationValidDays),
	}

	if err := s.repo.Create(ctx, inv); err != nil {
		slog.Error("failed to create invitation", "org_id", req.OrganizationID, "email", req.Email, "error", err)
		return nil, err
	}

	// Send invitation email
	if s.emailService != nil {
		if err := s.emailService.SendOrgInvitationEmail(ctx, req.Email, req.OrgName, req.InviterName, token); err != nil {
			slog.Warn("failed to send invitation email", "org_id", req.OrganizationID, "email", req.Email, "error", err)
		}
	}

	slog.Info("invitation created", "org_id", req.OrganizationID, "email", req.Email, "role", req.Role, "inviter_id", req.InviterID)
	return inv, nil
}

// Resend resends an invitation email
func (s *Service) Resend(ctx context.Context, invitationID int64, inviterName, orgName string) error {
	inv, err := s.repo.GetByID(ctx, invitationID)
	if err != nil {
		return ErrInvitationNotFound
	}

	if inv.IsAccepted() {
		return ErrInvitationAccepted
	}

	// Extend expiration if needed
	if inv.IsExpired() || time.Until(inv.ExpiresAt) < 24*time.Hour {
		inv.ExpiresAt = time.Now().AddDate(0, 0, InvitationValidDays)
		if err := s.repo.Update(ctx, inv); err != nil {
			slog.Error("failed to extend invitation expiration", "invitation_id", invitationID, "error", err)
			return err
		}
	}

	// Send email
	if s.emailService != nil {
		return s.emailService.SendOrgInvitationEmail(ctx, inv.Email, orgName, inviterName, inv.Token)
	}

	slog.Info("invitation resent", "invitation_id", invitationID, "email", inv.Email)
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
