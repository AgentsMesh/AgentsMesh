package admin

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/admin"
	"github.com/anthropics/agentsmesh/backend/internal/domain/promocode"
)

func (s *Service) GetPromoCode(ctx context.Context, id int64) (*promocode.PromoCode, error) {
	var code promocode.PromoCode
	if err := s.db.Model(&promocode.PromoCode{}).Where("id = ?", id).First(&code); err != nil {
		return nil, ErrPromoCodeNotFound
	}
	return &code, nil
}

func (s *Service) CreatePromoCode(ctx context.Context, code *promocode.PromoCode, adminUserID int64) error {
	var existing promocode.PromoCode
	if err := s.db.Model(&promocode.PromoCode{}).Where("code = ?", code.Code).First(&existing); err == nil {
		return ErrPromoCodeAlreadyExists
	}

	if err := s.db.Create(code); err != nil {
		slog.ErrorContext(ctx, "admin: failed to create promo code", "code", code.Code, "error", err)
		return fmt.Errorf("failed to create promo code: %w", err)
	}

	slog.InfoContext(ctx, "admin: promo code created", "code_id", code.ID, "code", code.Code, "admin_user_id", adminUserID)
	s.createPromoCodeAuditLog(ctx, adminUserID, admin.AuditActionCreate, code.ID, nil, code)

	return nil
}

func (s *Service) UpdatePromoCode(ctx context.Context, id int64, input *PromoCodeUpdateInput, adminUserID int64) (*promocode.PromoCode, error) {
	var code promocode.PromoCode
	if err := s.db.Model(&promocode.PromoCode{}).Where("id = ?", id).First(&code); err != nil {
		return nil, ErrPromoCodeNotFound
	}

	oldData := code

	if input.Name != nil {
		code.Name = *input.Name
	}
	if input.Description != nil {
		code.Description = *input.Description
	}
	if input.MaxUses != nil {
		code.MaxUses = input.MaxUses
	}
	if input.MaxUsesPerOrg != nil {
		code.MaxUsesPerOrg = *input.MaxUsesPerOrg
	}
	if input.ClearExpiresAt {
		code.ExpiresAt = nil
	} else if input.ExpiresAt != nil {
		code.ExpiresAt = input.ExpiresAt
	}

	code.UpdatedAt = time.Now()

	if err := s.db.Save(&code); err != nil {
		slog.ErrorContext(ctx, "admin: failed to update promo code", "code_id", id, "error", err)
		return nil, fmt.Errorf("failed to update promo code: %w", err)
	}

	slog.InfoContext(ctx, "admin: promo code updated", "code_id", id, "admin_user_id", adminUserID)
	s.createPromoCodeAuditLog(ctx, adminUserID, admin.AuditActionUpdate, code.ID, &oldData, &code)

	return &code, nil
}

func (s *Service) DeletePromoCode(ctx context.Context, id int64, adminUserID int64) error {
	var code promocode.PromoCode
	if err := s.db.Model(&promocode.PromoCode{}).Where("id = ?", id).First(&code); err != nil {
		return ErrPromoCodeNotFound
	}

	var redemptionCount int64
	if err := s.db.Table("promo_code_redemptions").Where("promo_code_id = ?", id).Count(&redemptionCount); err != nil {
		return fmt.Errorf("failed to count redemptions: %w", err)
	}
	if redemptionCount > 0 {
		return ErrPromoCodeHasRedemptions
	}

	if err := s.db.Delete(&promocode.PromoCode{}, id); err != nil {
		slog.ErrorContext(ctx, "admin: failed to delete promo code", "code_id", id, "error", err)
		return fmt.Errorf("failed to delete promo code: %w", err)
	}

	slog.InfoContext(ctx, "admin: promo code deleted", "code_id", id, "admin_user_id", adminUserID)
	s.createPromoCodeAuditLog(ctx, adminUserID, admin.AuditActionDelete, id, &code, nil)

	return nil
}
