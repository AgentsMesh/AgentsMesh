package admin

import (
	"context"
	"fmt"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/organization"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/promocode"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
)

// ListPromoCodes lists promo codes with filtering and pagination
func (s *Service) ListPromoCodes(ctx context.Context, filter *PromoCodeListFilter) (*PromoCodeListResult, error) {
	query := s.db.Model(&promocode.PromoCode{})

	if filter.Type != nil {
		query = query.Where("type = ?", *filter.Type)
	}
	if filter.PlanName != nil {
		query = query.Where("plan_name = ?", *filter.PlanName)
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.Search != nil && *filter.Search != "" {
		search := "%" + *filter.Search + "%"
		query = query.Where("code ILIKE ? OR name ILIKE ?", search, search)
	}

	var total int64
	if err := query.Count(&total); err != nil {
		return nil, fmt.Errorf("failed to count promo codes: %w", err)
	}

	pagination := normalizePagination(filter.Page, filter.PageSize, total)

	var codes []*promocode.PromoCode
	if err := query.Order("created_at DESC").
		Offset(pagination.Offset).
		Limit(pagination.PageSize).
		Find(&codes); err != nil {
		return nil, fmt.Errorf("failed to list promo codes: %w", err)
	}

	return &PromoCodeListResult{
		Data:       codes,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: pagination.TotalPages,
	}, nil
}

// ListPromoCodeRedemptions lists redemptions for a promo code
func (s *Service) ListPromoCodeRedemptions(ctx context.Context, promoCodeID int64, page, pageSize int) (*RedemptionListResult, error) {
	// Check if promo code exists
	var code promocode.PromoCode
	if err := s.db.Model(&promocode.PromoCode{}).Where("id = ?", promoCodeID).First(&code); err != nil {
		return nil, ErrPromoCodeNotFound
	}

	query := s.db.Table("promo_code_redemptions").Where("promo_code_id = ?", promoCodeID)

	var total int64
	if err := query.Count(&total); err != nil {
		return nil, fmt.Errorf("failed to count redemptions: %w", err)
	}

	pagination := normalizePagination(page, pageSize, total)

	var redemptions []*promocode.Redemption
	if err := s.db.Model(&promocode.Redemption{}).
		Where("promo_code_id = ?", promoCodeID).
		Order("created_at DESC").
		Offset(pagination.Offset).
		Limit(pagination.PageSize).
		Find(&redemptions); err != nil {
		return nil, fmt.Errorf("failed to list redemptions: %w", err)
	}

	// Fetch user and organization details
	result := make([]*RedemptionWithDetails, len(redemptions))
	for i, r := range redemptions {
		detail := &RedemptionWithDetails{
			ID:             r.ID,
			PromoCodeID:    r.PromoCodeID,
			OrganizationID: r.OrganizationID,
			UserID:         r.UserID,
			PlanName:       r.PlanName,
			DurationMonths: r.DurationMonths,
			NewPeriodEnd:   r.NewPeriodEnd,
			IPAddress:      r.IPAddress,
			CreatedAt:      r.CreatedAt,
		}

		// Fetch user
		var u user.User
		if err := s.db.Model(&user.User{}).Where("id = ?", r.UserID).First(&u); err == nil {
			detail.User = &u
		}

		// Fetch organization
		var org organization.Organization
		if err := s.db.Model(&organization.Organization{}).Where("id = ?", r.OrganizationID).First(&org); err == nil {
			detail.Organization = &org
		}

		result[i] = detail
	}

	return &RedemptionListResult{
		Data:       result,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: pagination.TotalPages,
	}, nil
}
