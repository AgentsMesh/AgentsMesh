package infra

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"gorm.io/gorm"
)

// Compile-time interface check
var _ organization.Repository = (*organizationRepo)(nil)

type organizationRepo struct {
	db *gorm.DB
}

// NewOrganizationRepository creates a new GORM-based organization repository
func NewOrganizationRepository(db *gorm.DB) organization.Repository {
	return &organizationRepo{db: db}
}

func (r *organizationRepo) GetByID(ctx context.Context, id int64) (*organization.Organization, error) {
	var org organization.Organization
	if err := r.db.WithContext(ctx).First(&org, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, organization.ErrNotFound
		}
		return nil, err
	}
	return &org, nil
}

func (r *organizationRepo) GetBySlug(ctx context.Context, slug string) (*organization.Organization, error) {
	var org organization.Organization
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&org).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, organization.ErrNotFound
		}
		return nil, err
	}
	return &org, nil
}

func (r *organizationRepo) SlugExists(ctx context.Context, slug string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&organization.Organization{}).Where("slug = ?", slug).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *organizationRepo) Update(ctx context.Context, id int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&organization.Organization{}).Where("id = ?", id).Updates(updates).Error
}

func (r *organizationRepo) ListByUser(ctx context.Context, userID int64) ([]*organization.Organization, error) {
	var orgs []*organization.Organization
	err := r.db.WithContext(ctx).
		Select("organizations.*, organization_members.role AS role").
		Joins("JOIN organization_members ON organization_members.organization_id = organizations.id").
		Where("organization_members.user_id = ?", userID).
		Find(&orgs).Error
	return orgs, err
}

func (r *organizationRepo) CreateWithMember(ctx context.Context, params *organization.CreateOrgParams) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(params.Organization).Error; err != nil {
			return err
		}

		// Set the organization ID on the member
		params.OwnerMember.OrganizationID = params.Organization.ID
		if err := tx.Create(params.OwnerMember).Error; err != nil {
			return err
		}

		if params.AfterCreate != nil {
			return params.AfterCreate(ctx, tx)
		}

		return nil
	})
}

func (r *organizationRepo) DeleteWithCleanup(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Application-level cleanup for tables without FK CASCADE
		if err := tx.Exec("DELETE FROM loop_runs WHERE organization_id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM loops WHERE organization_id = ?", id).Error; err != nil {
			return err
		}

		// Channel cleanup (FK removed in migration 000072)
		subq := "SELECT id FROM channels WHERE organization_id = ?"
		if err := tx.Exec("DELETE FROM channel_messages WHERE channel_id IN ("+subq+")", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM channel_members WHERE channel_id IN ("+subq+")", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM channel_read_states WHERE channel_id IN ("+subq+")", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM channel_pods WHERE channel_id IN ("+subq+")", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM channel_access WHERE channel_id IN ("+subq+")", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM pod_bindings WHERE channel_id IN ("+subq+")", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM channels WHERE organization_id = ?", id).Error; err != nil {
			return err
		}

		// Delete the org — remaining FK CASCADE handles other dependent tables
		return tx.Delete(&organization.Organization{}, id).Error
	})
}

func (r *organizationRepo) CreateMember(ctx context.Context, member *organization.Member) error {
	return r.db.WithContext(ctx).Create(member).Error
}

func (r *organizationRepo) GetMember(ctx context.Context, orgID, userID int64) (*organization.Member, error) {
	var member organization.Member
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND user_id = ?", orgID, userID).
		First(&member).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, organization.ErrMemberNotFound
		}
		return nil, err
	}
	return &member, nil
}

func (r *organizationRepo) DeleteMember(ctx context.Context, orgID, userID int64) error {
	return r.db.WithContext(ctx).
		Where("organization_id = ? AND user_id = ?", orgID, userID).
		Delete(&organization.Member{}).Error
}

func (r *organizationRepo) UpdateMemberRole(ctx context.Context, orgID, userID int64, role string) error {
	return r.db.WithContext(ctx).Model(&organization.Member{}).
		Where("organization_id = ? AND user_id = ?", orgID, userID).
		Update("role", role).Error
}

func (r *organizationRepo) ListMembers(ctx context.Context, orgID int64) ([]*organization.Member, error) {
	var members []*organization.Member
	err := r.db.WithContext(ctx).Where("organization_id = ?", orgID).Find(&members).Error
	return members, err
}

func (r *organizationRepo) ListMembersWithUser(ctx context.Context, orgID int64) ([]*organization.Member, error) {
	var members []*organization.Member
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("organization_id = ?", orgID).
		Find(&members).Error
	return members, err
}

func (r *organizationRepo) MemberExists(ctx context.Context, orgID, userID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&organization.Member{}).
		Where("organization_id = ? AND user_id = ?", orgID, userID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
