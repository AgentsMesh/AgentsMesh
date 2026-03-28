package organization

import (
	"context"
	"errors"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	orgDomain "github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
)

var (
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrSlugAlreadyExists    = errors.New("organization slug already exists")
	ErrNotOrganizationAdmin = errors.New("not an organization admin")
	ErrCannotRemoveOwner    = errors.New("cannot remove organization owner")
)

// BillingService interface for creating trial subscriptions
type BillingService interface {
	CreateTrialSubscription(ctx context.Context, orgID int64, planName string, trialDays int) (*billing.Subscription, error)
	CreateTrialSubscriptionTx(ctx context.Context, rawTx interface{}, orgID int64, planName string, trialDays int) (*billing.Subscription, error)
}

// Service handles organization operations
type Service struct {
	repo           orgDomain.Repository
	billingService BillingService
}

// NewService creates a new organization service
func NewService(repo orgDomain.Repository) *Service {
	return &Service{repo: repo}
}

// NewServiceWithBilling creates a new organization service with billing support
func NewServiceWithBilling(repo orgDomain.Repository, billingService BillingService) *Service {
	return &Service{
		repo:           repo,
		billingService: billingService,
	}
}

// CreateRequest represents organization creation request
type CreateRequest struct {
	Name    string
	Slug    string
	LogoURL string
}

// Create creates a new organization with trial subscription
func (s *Service) Create(ctx context.Context, ownerID int64, req *CreateRequest) (*orgDomain.Organization, error) {
	// Check if slug already exists
	exists, err := s.repo.SlugExists(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrSlugAlreadyExists
	}

	org := &orgDomain.Organization{
		Name:               req.Name,
		Slug:               req.Slug,
		SubscriptionPlan:   billing.PlanBased,
		SubscriptionStatus: billing.SubscriptionStatusTrialing,
	}
	if req.LogoURL != "" {
		org.LogoURL = &req.LogoURL
	}

	member := &orgDomain.Member{
		UserID: ownerID,
		Role:   orgDomain.RoleOwner,
	}

	params := &orgDomain.CreateOrgParams{
		Organization: org,
		OwnerMember:  member,
	}

	// Add billing callback if billing service is available
	if s.billingService != nil {
		params.AfterCreate = func(ctx context.Context, tx interface{}) error {
			_, err := s.billingService.CreateTrialSubscriptionTx(ctx, tx, org.ID, billing.PlanBased, billing.DefaultTrialDays)
			return err
		}
	}

	if err := s.repo.CreateWithMember(ctx, params); err != nil {
		slog.Error("failed to create organization", "slug", req.Slug, "owner_id", ownerID, "error", err)
		return nil, err
	}

	slog.Info("organization created", "org_id", org.ID, "slug", req.Slug, "owner_id", ownerID)
	return org, nil
}

// GetByID returns an organization by ID
func (s *Service) GetByID(ctx context.Context, id int64) (*orgDomain.Organization, error) {
	org, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == orgDomain.ErrNotFound {
			return nil, ErrOrganizationNotFound
		}
		return nil, err
	}
	return org, nil
}

// GetBySlug returns an organization by slug (implements middleware.OrganizationService)
func (s *Service) GetBySlug(ctx context.Context, slug string) (middleware.OrganizationGetter, error) {
	org, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		if err == orgDomain.ErrNotFound {
			return nil, ErrOrganizationNotFound
		}
		return nil, err
	}
	return org, nil
}

// GetOrgBySlug returns an organization by slug (returns concrete type for internal use)
func (s *Service) GetOrgBySlug(ctx context.Context, slug string) (*orgDomain.Organization, error) {
	org, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		if err == orgDomain.ErrNotFound {
			return nil, ErrOrganizationNotFound
		}
		return nil, err
	}
	return org, nil
}

// Update updates an organization
func (s *Service) Update(ctx context.Context, id int64, updates map[string]interface{}) (*orgDomain.Organization, error) {
	if err := s.repo.Update(ctx, id, updates); err != nil {
		slog.Error("failed to update organization", "org_id", id, "error", err)
		return nil, err
	}
	slog.Info("organization updated", "org_id", id)
	return s.GetByID(ctx, id)
}

// Delete deletes an organization.
//
// Tables with FK ON DELETE CASCADE are cleaned up automatically by PostgreSQL.
// Tables without FK (loops, loop_runs) require explicit application-level cleanup.
func (s *Service) Delete(ctx context.Context, id int64) error {
	if err := s.repo.DeleteWithCleanup(ctx, id); err != nil {
		slog.Error("failed to delete organization", "org_id", id, "error", err)
		return err
	}
	slog.Info("organization deleted", "org_id", id)
	return nil
}

// ListByUser returns organizations for a user
func (s *Service) ListByUser(ctx context.Context, userID int64) ([]*orgDomain.Organization, error) {
	return s.repo.ListByUser(ctx, userID)
}

// AddMember adds a member to an organization
func (s *Service) AddMember(ctx context.Context, orgID, userID int64, role string) error {
	member := &orgDomain.Member{
		OrganizationID: orgID,
		UserID:         userID,
		Role:           role,
	}
	if err := s.repo.CreateMember(ctx, member); err != nil {
		slog.Error("failed to add member", "org_id", orgID, "user_id", userID, "role", role, "error", err)
		return err
	}
	slog.Info("member added to organization", "org_id", orgID, "user_id", userID, "role", role)
	return nil
}

// RemoveMember removes a member from an organization
func (s *Service) RemoveMember(ctx context.Context, orgID, userID int64) error {
	// Check if user is owner
	member, err := s.repo.GetMember(ctx, orgID, userID)
	if err == nil && member.Role == orgDomain.RoleOwner {
		return ErrCannotRemoveOwner
	}
	if err := s.repo.DeleteMember(ctx, orgID, userID); err != nil {
		slog.Error("failed to remove member", "org_id", orgID, "user_id", userID, "error", err)
		return err
	}
	slog.Info("member removed from organization", "org_id", orgID, "user_id", userID)
	return nil
}

// UpdateMemberRole updates a member's role
func (s *Service) UpdateMemberRole(ctx context.Context, orgID, userID int64, role string) error {
	return s.repo.UpdateMemberRole(ctx, orgID, userID, role)
}

// GetMember returns a member
func (s *Service) GetMember(ctx context.Context, orgID, userID int64) (*orgDomain.Member, error) {
	return s.repo.GetMember(ctx, orgID, userID)
}

// ListMembers returns members of an organization with user details
func (s *Service) ListMembers(ctx context.Context, orgID int64) ([]*orgDomain.Member, error) {
	return s.repo.ListMembersWithUser(ctx, orgID)
}

// IsAdmin checks if a user is an admin of the organization
func (s *Service) IsAdmin(ctx context.Context, orgID, userID int64) (bool, error) {
	member, err := s.repo.GetMember(ctx, orgID, userID)
	if err != nil {
		return false, nil
	}
	return member.Role == orgDomain.RoleOwner || member.Role == orgDomain.RoleAdmin, nil
}

// IsOwner checks if a user is the owner of the organization
func (s *Service) IsOwner(ctx context.Context, orgID, userID int64) (bool, error) {
	member, err := s.repo.GetMember(ctx, orgID, userID)
	if err != nil {
		return false, nil
	}
	return member.Role == orgDomain.RoleOwner, nil
}

// IsMember checks if a user is a member of the organization
func (s *Service) IsMember(ctx context.Context, orgID, userID int64) (bool, error) {
	return s.repo.MemberExists(ctx, orgID, userID)
}

// GetUserRole returns the user's role in the organization
func (s *Service) GetUserRole(ctx context.Context, orgID, userID int64) (string, error) {
	member, err := s.repo.GetMember(ctx, orgID, userID)
	if err != nil {
		return "", err
	}
	return member.Role, nil
}

// GetMemberRole returns the user's role in the organization (alias for GetUserRole)
func (s *Service) GetMemberRole(ctx context.Context, orgID, userID int64) (string, error) {
	return s.GetUserRole(ctx, orgID, userID)
}
