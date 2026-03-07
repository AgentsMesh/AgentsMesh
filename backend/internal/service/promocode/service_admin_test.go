package promocode_test

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/promocode"
	svc "github.com/anthropics/agentsmesh/backend/internal/service/promocode"
)

func TestService_Deactivate(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create test promo code
	code := &promocode.PromoCode{
		Code:           "DEACTIVATE2024",
		Name:           "Deactivate Test",
		Type:           promocode.PromoTypeMedia,
		PlanName:       "pro",
		DurationMonths: 1,
		IsActive:       true,
		StartsAt:       time.Now(),
	}
	db.Create(code)

	t.Run("deactivate existing code", func(t *testing.T) {
		err := service.Deactivate(ctx, code.ID)
		if err != nil {
			t.Errorf("Deactivate() error = %v", err)
			return
		}

		// Verify it's deactivated
		var updated promocode.PromoCode
		db.First(&updated, code.ID)
		if updated.IsActive {
			t.Error("PromoCode should be deactivated")
		}
	})

	t.Run("deactivate nonexistent code", func(t *testing.T) {
		err := service.Deactivate(ctx, 99999)
		if err == nil {
			t.Error("Deactivate() should fail for nonexistent code")
		}
	})
}

func TestService_List(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create test promo codes
	now := time.Now()
	for i := 1; i <= 5; i++ {
		db.Create(&promocode.PromoCode{
			Code:           "LIST" + string(rune('A'+i-1)),
			Name:           "List Test " + string(rune('A'+i-1)),
			Type:           promocode.PromoTypeMedia,
			PlanName:       "pro",
			DurationMonths: i,
			IsActive:       i%2 == 1, // odd ones are active
			StartsAt:       now,
		})
	}

	t.Run("list all", func(t *testing.T) {
		codes, total, err := service.List(ctx, &promocode.ListFilter{
			Page:     1,
			PageSize: 10,
		})
		if err != nil {
			t.Errorf("List() error = %v", err)
			return
		}
		if total != 5 {
			t.Errorf("List() total = %v, want 5", total)
		}
		if len(codes) != 5 {
			t.Errorf("List() len = %v, want 5", len(codes))
		}
	})

	t.Run("list active only", func(t *testing.T) {
		active := true
		codes, total, err := service.List(ctx, &promocode.ListFilter{
			IsActive: &active,
			Page:     1,
			PageSize: 10,
		})
		if err != nil {
			t.Errorf("List() error = %v", err)
			return
		}
		// Count active codes
		activeCount := 0
		for _, c := range codes {
			if c.IsActive {
				activeCount++
			}
		}
		if int64(activeCount) != total {
			t.Errorf("List() active count mismatch: %d codes returned but total=%d", activeCount, total)
		}
	})

	t.Run("list with pagination", func(t *testing.T) {
		codes, total, err := service.List(ctx, &promocode.ListFilter{
			Page:     1,
			PageSize: 2,
		})
		if err != nil {
			t.Errorf("List() error = %v", err)
			return
		}
		if total != 5 {
			t.Errorf("List() total = %v, want 5", total)
		}
		if len(codes) != 2 {
			t.Errorf("List() len = %v, want 2", len(codes))
		}
	})
}

// Silence unused import warning for svc package
var _ = svc.ErrPromoCodeNotFound
