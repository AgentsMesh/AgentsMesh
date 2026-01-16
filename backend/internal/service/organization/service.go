package organization

import (
	"context"
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"gorm.io/gorm"
)

var (
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrSlugAlreadyExists    = errors.New("organization slug already exists")
	ErrNotOrganizationAdmin = errors.New("not an organization admin")
	ErrCannotRemoveOwner    = errors.New("cannot remove organization owner")
)

// Service handles organization operations
type Service struct {
	db *gorm.DB
}

// NewService creates a new organization service
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// CreateRequest represents organization creation request
type CreateRequest struct {
	Name    string
	Slug    string
	LogoURL string
}

// Create creates a new organization
func (s *Service) Create(ctx context.Context, ownerID int64, req *CreateRequest) (*organization.Organization, error) {
	// Check if slug already exists
	var existing organization.Organization
	if err := s.db.WithContext(ctx).Where("slug = ?", req.Slug).First(&existing).Error; err == nil {
		return nil, ErrSlugAlreadyExists
	}

	org := &organization.Organization{
		Name:               req.Name,
		Slug:               req.Slug,
		SubscriptionPlan:   "free",
		SubscriptionStatus: "active",
	}
	if req.LogoURL != "" {
		org.LogoURL = &req.LogoURL
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(org).Error; err != nil {
			return err
		}

		// Add owner as organization member
		member := &organization.Member{
			OrganizationID: org.ID,
			UserID:         ownerID,
			Role:           organization.RoleOwner,
		}
		return tx.Create(member).Error
	})

	if err != nil {
		return nil, err
	}

	return org, nil
}

// GetByID returns an organization by ID
func (s *Service) GetByID(ctx context.Context, id int64) (*organization.Organization, error) {
	var org organization.Organization
	if err := s.db.WithContext(ctx).First(&org, id).Error; err != nil {
		return nil, ErrOrganizationNotFound
	}
	return &org, nil
}

// GetBySlug returns an organization by slug (implements middleware.OrganizationService)
func (s *Service) GetBySlug(ctx context.Context, slug string) (middleware.OrganizationGetter, error) {
	var org organization.Organization
	if err := s.db.WithContext(ctx).Where("slug = ?", slug).First(&org).Error; err != nil {
		return nil, ErrOrganizationNotFound
	}
	return &org, nil
}

// GetOrgBySlug returns an organization by slug (returns concrete type for internal use)
func (s *Service) GetOrgBySlug(ctx context.Context, slug string) (*organization.Organization, error) {
	var org organization.Organization
	if err := s.db.WithContext(ctx).Where("slug = ?", slug).First(&org).Error; err != nil {
		return nil, ErrOrganizationNotFound
	}
	return &org, nil
}

// Update updates an organization
func (s *Service) Update(ctx context.Context, id int64, updates map[string]interface{}) (*organization.Organization, error) {
	if err := s.db.WithContext(ctx).Model(&organization.Organization{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

// Delete deletes an organization
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&organization.Organization{}, id).Error
}

// ListByUser returns organizations for a user
func (s *Service) ListByUser(ctx context.Context, userID int64) ([]*organization.Organization, error) {
	var orgs []*organization.Organization
	err := s.db.WithContext(ctx).
		Joins("JOIN organization_members ON organization_members.organization_id = organizations.id").
		Where("organization_members.user_id = ?", userID).
		Find(&orgs).Error
	return orgs, err
}

// AddMember adds a member to an organization
func (s *Service) AddMember(ctx context.Context, orgID, userID int64, role string) error {
	member := &organization.Member{
		OrganizationID: orgID,
		UserID:         userID,
		Role:           role,
	}
	return s.db.WithContext(ctx).Create(member).Error
}

// RemoveMember removes a member from an organization
func (s *Service) RemoveMember(ctx context.Context, orgID, userID int64) error {
	// Check if user is owner
	var member organization.Member
	if err := s.db.WithContext(ctx).Where("organization_id = ? AND user_id = ?", orgID, userID).First(&member).Error; err == nil {
		if member.Role == organization.RoleOwner {
			return ErrCannotRemoveOwner
		}
	}
	return s.db.WithContext(ctx).Where("organization_id = ? AND user_id = ?", orgID, userID).Delete(&organization.Member{}).Error
}

// UpdateMemberRole updates a member's role
func (s *Service) UpdateMemberRole(ctx context.Context, orgID, userID int64, role string) error {
	return s.db.WithContext(ctx).Model(&organization.Member{}).
		Where("organization_id = ? AND user_id = ?", orgID, userID).
		Update("role", role).Error
}

// GetMember returns a member
func (s *Service) GetMember(ctx context.Context, orgID, userID int64) (*organization.Member, error) {
	var member organization.Member
	if err := s.db.WithContext(ctx).Where("organization_id = ? AND user_id = ?", orgID, userID).First(&member).Error; err != nil {
		return nil, err
	}
	return &member, nil
}

// ListMembers returns members of an organization
func (s *Service) ListMembers(ctx context.Context, orgID int64) ([]*user.User, error) {
	var users []*user.User
	err := s.db.WithContext(ctx).
		Joins("JOIN organization_members ON organization_members.user_id = users.id").
		Where("organization_members.organization_id = ?", orgID).
		Find(&users).Error
	return users, err
}

// IsAdmin checks if a user is an admin of the organization
func (s *Service) IsAdmin(ctx context.Context, orgID, userID int64) (bool, error) {
	var member organization.Member
	if err := s.db.WithContext(ctx).Where("organization_id = ? AND user_id = ?", orgID, userID).First(&member).Error; err != nil {
		return false, nil
	}
	return member.Role == organization.RoleOwner || member.Role == organization.RoleAdmin, nil
}

// IsOwner checks if a user is the owner of the organization
func (s *Service) IsOwner(ctx context.Context, orgID, userID int64) (bool, error) {
	var member organization.Member
	if err := s.db.WithContext(ctx).Where("organization_id = ? AND user_id = ?", orgID, userID).First(&member).Error; err != nil {
		return false, nil
	}
	return member.Role == organization.RoleOwner, nil
}

// IsMember checks if a user is a member of the organization
func (s *Service) IsMember(ctx context.Context, orgID, userID int64) (bool, error) {
	var count int64
	s.db.WithContext(ctx).Model(&organization.Member{}).Where("organization_id = ? AND user_id = ?", orgID, userID).Count(&count)
	return count > 0, nil
}

// GetUserRole returns the user's role in the organization
func (s *Service) GetUserRole(ctx context.Context, orgID, userID int64) (string, error) {
	var member organization.Member
	if err := s.db.WithContext(ctx).Where("organization_id = ? AND user_id = ?", orgID, userID).First(&member).Error; err != nil {
		return "", err
	}
	return member.Role, nil
}

// GetMemberRole returns the user's role in the organization (alias for GetUserRole)
func (s *Service) GetMemberRole(ctx context.Context, orgID, userID int64) (string, error) {
	return s.GetUserRole(ctx, orgID, userID)
}
