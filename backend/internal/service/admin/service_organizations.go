package admin

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/infra/database"
)

type OrganizationListQuery struct {
	Search   string
	Page     int
	PageSize int
}

type OrganizationListResponse struct {
	Data       []organization.Organization `json:"data"`
	Total      int64                       `json:"total"`
	Page       int                         `json:"page"`
	PageSize   int                         `json:"page_size"`
	TotalPages int                         `json:"total_pages"`
}

func (s *Service) ListOrganizations(ctx context.Context, query *OrganizationListQuery) (*OrganizationListResponse, error) {
	db := s.db.Model(&organization.Organization{})

	if query.Search != "" {
		searchPattern := "%" + query.Search + "%"
		db = db.Where("name ILIKE ? OR slug ILIKE ?", searchPattern, searchPattern)
	}

	var total int64
	if err := db.Count(&total); err != nil {
		return nil, err
	}

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

func (s *Service) GetOrganization(ctx context.Context, orgID int64) (*organization.Organization, error) {
	var org organization.Organization
	if err := s.db.First(&org, orgID); err != nil {
		return nil, ErrOrganizationNotFound
	}
	return &org, nil
}

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

func (s *Service) UpdateOrganizationSubscriptionStatus(ctx context.Context, orgID int64, status string) error {
	var org organization.Organization
	if err := s.db.First(&org, orgID); err != nil {
		return err
	}
	org.SubscriptionStatus = status
	if err := s.db.Save(&org); err != nil {
		slog.ErrorContext(ctx, "admin: failed to update org subscription status", "org_id", orgID, "status", status, "error", err)
		return err
	}
	slog.InfoContext(ctx, "admin: org subscription status updated", "org_id", orgID, "status", status)
	return nil
}

func (s *Service) DeleteOrganization(ctx context.Context, orgID int64) error {
	var org organization.Organization
	if err := s.db.First(&org, orgID); err != nil {
		return ErrOrganizationNotFound
	}

	var runnerCount int64
	if err := s.db.Model(&runner.Runner{}).Where("organization_id = ?", orgID).Count(&runnerCount); err != nil {
		return fmt.Errorf("failed to check runners: %w", err)
	}
	if runnerCount > 0 {
		return ErrOrganizationHasActiveRunner
	}

	return s.db.Transaction(func(tx database.DB) error {
		gormTx := tx.GormDB()
		gormTx.Exec("DELETE FROM loop_runs WHERE organization_id = ?", orgID)
		gormTx.Exec("DELETE FROM loops WHERE organization_id = ?", orgID)

		if err := tx.Delete(&org); err != nil {
			slog.ErrorContext(ctx, "admin: failed to delete organization", "org_id", orgID, "error", err)
			return err
		}
		slog.InfoContext(ctx, "admin: organization deleted", "org_id", orgID)
		return nil
	})
}
