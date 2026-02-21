package admin

import (
	"context"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

// OrganizationListQuery represents query parameters for organization listing
type OrganizationListQuery struct {
	Search   string
	Page     int
	PageSize int
}

// OrganizationListResponse represents paginated organization list response
type OrganizationListResponse struct {
	Data       []organization.Organization `json:"data"`
	Total      int64                       `json:"total"`
	Page       int                         `json:"page"`
	PageSize   int                         `json:"page_size"`
	TotalPages int                         `json:"total_pages"`
}

// ListOrganizations retrieves organizations with filtering and pagination
func (s *Service) ListOrganizations(ctx context.Context, query *OrganizationListQuery) (*OrganizationListResponse, error) {
	db := s.db.Model(&organization.Organization{})

	// Apply filters
	if query.Search != "" {
		searchPattern := "%" + query.Search + "%"
		db = db.Where("name ILIKE ? OR slug ILIKE ?", searchPattern, searchPattern)
	}

	// Count total
	var total int64
	if err := db.Count(&total); err != nil {
		return nil, err
	}

	// Apply pagination using helper
	p := normalizePagination(query.Page, query.PageSize, total)

	var orgs []organization.Organization
	if err := db.
		Order("created_at DESC").
		Limit(p.PageSize).
		Offset(p.Offset).
		Find(&orgs); err != nil {
		return nil, err
	}

	return &OrganizationListResponse{
		Data:       orgs,
		Total:      total,
		Page:       p.Page,
		PageSize:   p.PageSize,
		TotalPages: p.TotalPages,
	}, nil
}

// GetOrganization retrieves an organization by ID
func (s *Service) GetOrganization(ctx context.Context, orgID int64) (*organization.Organization, error) {
	var org organization.Organization
	if err := s.db.First(&org, orgID); err != nil {
		return nil, ErrOrganizationNotFound
	}
	return &org, nil
}

// GetOrganizationWithMembers retrieves an organization with its members
func (s *Service) GetOrganizationWithMembers(ctx context.Context, orgID int64) (*organization.Organization, []organization.Member, error) {
	var org organization.Organization
	if err := s.db.First(&org, orgID); err != nil {
		return nil, nil, ErrOrganizationNotFound
	}

	var members []organization.Member
	if err := s.db.Where("organization_id = ?", orgID).Preload("User").Find(&members); err != nil {
		return nil, nil, err
	}

	return &org, members, nil
}

// UpdateOrganizationSubscriptionStatus updates the subscription_status redundant field on the organizations table
func (s *Service) UpdateOrganizationSubscriptionStatus(ctx context.Context, orgID int64, status string) error {
	var org organization.Organization
	if err := s.db.First(&org, orgID); err != nil {
		return err
	}
	org.SubscriptionStatus = status
	return s.db.Save(&org)
}

// DeleteOrganization deletes an organization after checking for active runners
func (s *Service) DeleteOrganization(ctx context.Context, orgID int64) error {
	var org organization.Organization
	if err := s.db.First(&org, orgID); err != nil {
		return ErrOrganizationNotFound
	}

	// Check for active runners before deletion
	var runnerCount int64
	if err := s.db.Model(&runner.Runner{}).Where("organization_id = ?", orgID).Count(&runnerCount); err != nil {
		return fmt.Errorf("failed to check runners: %w", err)
	}
	if runnerCount > 0 {
		return ErrOrganizationHasActiveRunner
	}

	return s.db.Delete(&org)
}
