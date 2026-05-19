package infra

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/invitation"
	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"gorm.io/gorm"
)

type invitationRepo struct {
	db *gorm.DB
}

func NewInvitationRepository(db *gorm.DB) invitation.Repository {
	return &invitationRepo{db: db}
}

func (r *invitationRepo) Create(ctx context.Context, inv *invitation.Invitation) error {
	return r.db.WithContext(ctx).Create(inv).Error
}

func (r *invitationRepo) GetByToken(ctx context.Context, token string) (*invitation.Invitation, error) {
	var inv invitation.Invitation
	if err := r.db.WithContext(ctx).Where("token = ?", token).First(&inv).Error; err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *invitationRepo) GetByID(ctx context.Context, id int64) (*invitation.Invitation, error) {
	var inv invitation.Invitation
	if err := r.db.WithContext(ctx).First(&inv, id).Error; err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *invitationRepo) GetByOrgAndEmail(ctx context.Context, orgID int64, email string) (*invitation.Invitation, error) {
	var inv invitation.Invitation
	if err := r.db.WithContext(ctx).Where("organization_id = ? AND email = ? AND accepted_at IS NULL", orgID, email).First(&inv).Error; err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *invitationRepo) ListByOrganization(ctx context.Context, orgID int64) ([]*invitation.Invitation, error) {
	var invitations []*invitation.Invitation
	if err := r.db.WithContext(ctx).Where("organization_id = ?", orgID).Order("created_at DESC").Find(&invitations).Error; err != nil {
		return nil, err
	}
	return invitations, nil
}

func (r *invitationRepo) ListPendingByEmail(ctx context.Context, email string) ([]*invitation.Invitation, error) {
	var invitations []*invitation.Invitation
	if err := r.db.WithContext(ctx).Where("email = ? AND accepted_at IS NULL AND expires_at > ?", email, time.Now()).
		Order("created_at DESC").Find(&invitations).Error; err != nil {
		return nil, err
	}
	return invitations, nil
}

func (r *invitationRepo) Update(ctx context.Context, inv *invitation.Invitation) error {
	return r.db.WithContext(ctx).Save(inv).Error
}

func (r *invitationRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&invitation.Invitation{}, id).Error
}

func (r *invitationRepo) DeleteExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).Where("expires_at < ? AND accepted_at IS NULL", time.Now()).Delete(&invitation.Invitation{}).Error
}

func (r *invitationRepo) CheckMemberExists(ctx context.Context, orgID int64, userID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&organization.Member{}).
		Where("organization_id = ? AND user_id = ?", orgID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *invitationRepo) CheckMemberExistsByEmail(ctx context.Context, orgID int64, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&organization.Member{}).
		Where("organization_id = ? AND user_id IN (SELECT id FROM users WHERE email = ?)", orgID, email).
		Count(&count).Error
	return count > 0, err
}

func (r *invitationRepo) GetOrganization(ctx context.Context, orgID int64) (*invitation.OrgInfo, error) {
	var org organization.Organization
	if err := r.db.WithContext(ctx).First(&org, orgID).Error; err != nil {
		return nil, err
	}
	return &invitation.OrgInfo{
		Name: org.Name,
		Slug: org.Slug,
	}, nil
}

func (r *invitationRepo) GetUserDisplayName(ctx context.Context, userID int64) (string, error) {
	var u user.User
	if err := r.db.WithContext(ctx).First(&u, userID).Error; err != nil {
		return "", err
	}
	if u.Name != nil && *u.Name != "" {
		return *u.Name, nil
	}
	return u.Username, nil
}

// AcceptInvitationAtomic atomically adds a member and marks the invitation as accepted
func (r *invitationRepo) AcceptInvitationAtomic(ctx context.Context, params *invitation.AcceptInvitationParams) (*invitation.AcceptInvitationResult, error) {
	var result invitation.AcceptInvitationResult

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var org organization.Organization
		if err := tx.First(&org, params.Invitation.OrganizationID).Error; err != nil {
			return err
		}
		result.OrganizationID = org.ID

		member := &organization.Member{
			OrganizationID: params.Invitation.OrganizationID,
			UserID:         params.UserID,
			Role:           params.Role,
		}
		if err := tx.Create(member).Error; err != nil {
			return err
		}
		result.MemberID = member.ID

		now := time.Now()
		params.Invitation.AcceptedAt = &now
		return tx.Save(params.Invitation).Error
	})

	return &result, err
}

var _ invitation.Repository = (*invitationRepo)(nil)
