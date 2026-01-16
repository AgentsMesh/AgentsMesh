package promocode

import (
	"context"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/promocode"
)

// CreateRequest represents a create promo code request
type CreateRequest struct {
	Code           string
	Name           string
	Description    string
	Type           promocode.PromoCodeType
	PlanName       string
	DurationMonths int
	MaxUses        *int
	MaxUsesPerOrg  int
	StartsAt       *time.Time
	ExpiresAt      *time.Time
	CreatedByID    int64
}

// Create creates a new promo code (Admin only)
func (s *Service) Create(ctx context.Context, req *CreateRequest) (*promocode.PromoCode, error) {
	code := strings.ToUpper(strings.TrimSpace(req.Code))

	// Check if code already exists
	existing, _ := s.repo.GetByCode(ctx, code)
	if existing != nil {
		return nil, ErrPromoCodeAlreadyExists
	}

	// Validate plan via billing provider
	if _, err := s.billing.GetPlanByName(ctx, req.PlanName); err != nil {
		return nil, ErrInvalidPlan
	}

	startsAt := time.Now()
	if req.StartsAt != nil {
		startsAt = *req.StartsAt
	}

	maxUsesPerOrg := 1
	if req.MaxUsesPerOrg > 0 {
		maxUsesPerOrg = req.MaxUsesPerOrg
	}

	promoCode := &promocode.PromoCode{
		Code:           code,
		Name:           req.Name,
		Description:    req.Description,
		Type:           req.Type,
		PlanName:       req.PlanName,
		DurationMonths: req.DurationMonths,
		MaxUses:        req.MaxUses,
		MaxUsesPerOrg:  maxUsesPerOrg,
		StartsAt:       startsAt,
		ExpiresAt:      req.ExpiresAt,
		IsActive:       true,
		CreatedByID:    &req.CreatedByID,
	}

	if err := s.repo.Create(ctx, promoCode); err != nil {
		return nil, err
	}

	return promoCode, nil
}

// List lists promo codes with filtering (Admin only)
func (s *Service) List(ctx context.Context, filter *promocode.ListFilter) ([]*promocode.PromoCode, int64, error) {
	return s.repo.List(ctx, filter)
}

// GetByID gets a promo code by ID (Admin only)
func (s *Service) GetByID(ctx context.Context, id int64) (*promocode.PromoCode, error) {
	return s.repo.GetByID(ctx, id)
}

// Deactivate deactivates a promo code (Admin only)
func (s *Service) Deactivate(ctx context.Context, id int64) error {
	promoCode, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrPromoCodeNotFound
	}
	promoCode.IsActive = false
	return s.repo.Update(ctx, promoCode)
}

// Activate activates a promo code (Admin only)
func (s *Service) Activate(ctx context.Context, id int64) error {
	promoCode, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrPromoCodeNotFound
	}
	promoCode.IsActive = true
	return s.repo.Update(ctx, promoCode)
}
