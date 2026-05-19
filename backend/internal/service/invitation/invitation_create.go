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

type CreateRequest struct {
	OrganizationID int64
	Email          string
	Role           string
	InviterID      int64
	InviterName    string
	OrgName        string
}

func (s *Service) Create(ctx context.Context, req *CreateRequest) (*invitationDomain.Invitation, error) {
	if req.Role != organization.RoleAdmin && req.Role != organization.RoleMember {
		return nil, ErrInvalidRole
	}

	exists, err := s.repo.CheckMemberExistsByEmail(ctx, req.OrganizationID, req.Email)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check member existence", "org_id", req.OrganizationID, "email", req.Email, "error", err)
		return nil, err
	}
	if exists {
		return nil, ErrAlreadyMember
	}

	existing, err := s.repo.GetByOrgAndEmail(ctx, req.OrganizationID, req.Email)
	if err == nil && existing.IsPending() {
		return nil, ErrPendingInvitation
	}

	token, err := generateToken()
	if err != nil {
		slog.ErrorContext(ctx, "failed to generate invitation token", "error", err)
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
		slog.ErrorContext(ctx, "failed to create invitation", "org_id", req.OrganizationID, "email", req.Email, "error", err)
		return nil, err
	}

	if s.emailService != nil {
		if err := s.emailService.SendOrgInvitationEmail(ctx, req.Email, req.OrgName, req.InviterName, token); err != nil {
			slog.WarnContext(ctx, "failed to send invitation email", "org_id", req.OrganizationID, "email", req.Email, "error", err)
		}
	}

	slog.InfoContext(ctx, "invitation created", "org_id", req.OrganizationID, "email", req.Email, "role", req.Role, "inviter_id", req.InviterID)
	return inv, nil
}

func (s *Service) Resend(ctx context.Context, invitationID int64, inviterName, orgName string) error {
	inv, err := s.repo.GetByID(ctx, invitationID)
	if err != nil {
		return ErrInvitationNotFound
	}

	if inv.IsAccepted() {
		return ErrInvitationAccepted
	}

	if inv.IsExpired() || time.Until(inv.ExpiresAt) < 24*time.Hour {
		inv.ExpiresAt = time.Now().AddDate(0, 0, InvitationValidDays)
		if err := s.repo.Update(ctx, inv); err != nil {
			slog.ErrorContext(ctx, "failed to extend invitation expiration", "invitation_id", invitationID, "error", err)
			return err
		}
	}

	if s.emailService != nil {
		return s.emailService.SendOrgInvitationEmail(ctx, inv.Email, orgName, inviterName, inv.Token)
	}

	slog.InfoContext(ctx, "invitation resent", "invitation_id", invitationID, "email", inv.Email)
	return nil
}

func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
