package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/admin"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/promocode"
)

// ActivatePromoCode activates a promo code
func (s *Service) ActivatePromoCode(ctx context.Context, id int64, adminUserID int64) error {
	var code promocode.PromoCode
	if err := s.db.Model(&promocode.PromoCode{}).Where("id = ?", id).First(&code); err != nil {
		return ErrPromoCodeNotFound
	}

	oldData := code
	code.IsActive = true
	code.UpdatedAt = time.Now()

	if err := s.db.Save(&code); err != nil {
		return fmt.Errorf("failed to activate promo code: %w", err)
	}

	// Create audit log
	s.createPromoCodeAuditLog(ctx, adminUserID, admin.AuditActionActivate, code.ID, &oldData, &code)

	return nil
}

// DeactivatePromoCode deactivates a promo code
func (s *Service) DeactivatePromoCode(ctx context.Context, id int64, adminUserID int64) error {
	var code promocode.PromoCode
	if err := s.db.Model(&promocode.PromoCode{}).Where("id = ?", id).First(&code); err != nil {
		return ErrPromoCodeNotFound
	}

	oldData := code
	code.IsActive = false
	code.UpdatedAt = time.Now()

	if err := s.db.Save(&code); err != nil {
		return fmt.Errorf("failed to deactivate promo code: %w", err)
	}

	// Create audit log
	s.createPromoCodeAuditLog(ctx, adminUserID, admin.AuditActionDeactivate, code.ID, &oldData, &code)

	return nil
}
