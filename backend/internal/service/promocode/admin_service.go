package promocode

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/promocode"
)

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

func (s *Service) Create(ctx context.Context, req *CreateRequest) (*promocode.PromoCode, error) {
	code := strings.ToUpper(strings.TrimSpace(req.Code))

	existing, _ := s.repo.GetByCode(ctx, code)
	if existing != nil {
		return nil, ErrPromoCodeAlreadyExists
	}

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
		slog.ErrorContext(ctx, "failed to create promo code", "code", code, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "promo code created", "code_id", promoCode.ID, "code", code, "plan", req.PlanName)
	return promoCode, nil
}

func (s *Service) List(ctx context.Context, filter *promocode.ListFilter) ([]*promocode.PromoCode, int64, error) {
	return s.repo.List(ctx, filter)
}

func (s *Service) GetByID(ctx context.Context, id int64) (*promocode.PromoCode, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Deactivate(ctx context.Context, id int64) error {
	promoCode, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrPromoCodeNotFound
	}
	promoCode.IsActive = false
	if err := s.repo.Update(ctx, promoCode); err != nil {
		slog.ErrorContext(ctx, "failed to deactivate promo code", "code_id", id, "error", err)
		return err
	}
	slog.InfoContext(ctx, "promo code deactivated", "code_id", id)
	return nil
}

func (s *Service) Activate(ctx context.Context, id int64) error {
	promoCode, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrPromoCodeNotFound
	}
	promoCode.IsActive = true
	if err := s.repo.Update(ctx, promoCode); err != nil {
		slog.ErrorContext(ctx, "failed to activate promo code", "code_id", id, "error", err)
		return err
	}
	slog.InfoContext(ctx, "promo code activated", "code_id", id)
	return nil
}
