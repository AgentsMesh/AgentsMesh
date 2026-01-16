package invitation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/invitation"
	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/internal/infra/email"
	"gorm.io/gorm"
)

var (
	ErrInvitationNotFound  = errors.New("invitation not found")
	ErrInvitationExpired   = errors.New("invitation has expired")
	ErrInvitationAccepted  = errors.New("invitation already accepted")
	ErrAlreadyMember       = errors.New("user is already a member of this organization")
	ErrPendingInvitation   = errors.New("a pending invitation already exists for this email")
	ErrInvalidRole         = errors.New("invalid role")
	ErrNotAuthorized       = errors.New("not authorized to manage invitations")
)

const (
	// InvitationValidDays is the number of days an invitation is valid
	InvitationValidDays = 7
)

// Service handles invitation operations
type Service struct {
	db           *gorm.DB
	repo         invitation.Repository
	emailService email.Service
}

// NewService creates a new invitation service
func NewService(db *gorm.DB, emailService email.Service) *Service {
	return &Service{
		db:           db,
		repo:         invitation.NewRepository(db),
		emailService: emailService,
	}
}

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

// CleanupExpired removes expired invitations
func (s *Service) CleanupExpired(ctx context.Context) error {
	return s.repo.DeleteExpired()
}

// GetInvitationInfo returns information about an invitation for display
type InvitationInfo struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	OrgID        int64     `json:"organization_id"`
	OrgName      string    `json:"organization_name"`
	OrgSlug      string    `json:"organization_slug"`
	InviterName  string    `json:"inviter_name"`
	ExpiresAt    time.Time `json:"expires_at"`
	IsExpired    bool      `json:"is_expired"`
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

// generateToken generates a secure random token
func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
