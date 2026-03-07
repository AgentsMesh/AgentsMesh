package promocode_test

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/domain/promocode"
	svc "github.com/anthropics/agentsmesh/backend/internal/service/promocode"
)

func TestService_Validate(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create test promo codes
	now := time.Now()
	past := now.AddDate(0, -1, 0)
	future := now.AddDate(0, 1, 0)

	// Active code
	db.Create(&promocode.PromoCode{
		Code:           "ACTIVE2024",
		Name:           "Active Promo",
		Type:           promocode.PromoTypeMedia,
		PlanName:       "pro",
		DurationMonths: 3,
		IsActive:       true,
		StartsAt:       past,
		ExpiresAt:      &future,
	})

	// Expired code
	db.Create(&promocode.PromoCode{
		Code:           "EXPIRED2024",
		Name:           "Expired Promo",
		Type:           promocode.PromoTypeMedia,
		PlanName:       "pro",
		DurationMonths: 3,
		IsActive:       true,
		StartsAt:       past,
		ExpiresAt:      &past,
	})

	// Disabled code - use raw SQL to ensure is_active = 0
	db.Exec(`INSERT INTO promo_codes (code, name, type, plan_name, duration_months, is_active, starts_at, max_uses_per_org) VALUES (?, ?, ?, ?, ?, 0, ?, 1)`,
		"DISABLED2024", "Disabled Promo", "media", "pro", 3, past)

	// Max uses reached
	db.Create(&promocode.PromoCode{
		Code:           "MAXUSED2024",
		Name:           "Max Used Promo",
		Type:           promocode.PromoTypeMedia,
		PlanName:       "pro",
		DurationMonths: 3,
		IsActive:       true,
		StartsAt:       past,
		MaxUses:        intPtr(10),
		UsedCount:      10,
	})

	tests := []struct {
		name      string
		code      string
		orgID     int64
		wantValid bool
	}{
		{
			name:      "valid active code",
			code:      "ACTIVE2024",
			orgID:     1,
			wantValid: true,
		},
		{
			name:      "nonexistent code",
			code:      "NOTEXIST",
			orgID:     1,
			wantValid: false,
		},
		{
			name:      "expired code",
			code:      "EXPIRED2024",
			orgID:     1,
			wantValid: false,
		},
		{
			name:      "disabled code",
			code:      "DISABLED2024",
			orgID:     1,
			wantValid: false,
		},
		{
			name:      "max uses reached",
			code:      "MAXUSED2024",
			orgID:     1,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.Validate(ctx, &svc.ValidateRequest{
				Code:           tt.code,
				OrganizationID: tt.orgID,
			})
			if err != nil {
				t.Errorf("Validate() error = %v", err)
				return
			}
			if resp.Valid != tt.wantValid {
				t.Errorf("Validate() valid = %v, want %v, messageCode = %v", resp.Valid, tt.wantValid, resp.MessageCode)
			}
		})
	}
}

func TestService_Redeem(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create test promo code
	now := time.Now()
	past := now.AddDate(0, -1, 0)
	future := now.AddDate(0, 1, 0)

	db.Create(&promocode.PromoCode{
		Code:           "REDEEM2024",
		Name:           "Redeem Promo",
		Type:           promocode.PromoTypeMedia,
		PlanName:       "pro",
		DurationMonths: 3,
		IsActive:       true,
		StartsAt:       past,
		ExpiresAt:      &future,
		MaxUsesPerOrg:  1,
	})

	// Test: non-owner cannot redeem
	t.Run("non-owner cannot redeem", func(t *testing.T) {
		resp, err := service.Redeem(ctx, &svc.RedeemRequest{
			Code:           "REDEEM2024",
			OrganizationID: 1,
			UserID:         1,
			UserRole:       "member",
		})
		if err != nil {
			t.Errorf("Redeem() error = %v", err)
			return
		}
		if resp.Success {
			t.Error("Redeem() should fail for non-owner")
		}
	})

	// Test: owner can redeem
	t.Run("owner can redeem", func(t *testing.T) {
		resp, err := service.Redeem(ctx, &svc.RedeemRequest{
			Code:           "REDEEM2024",
			OrganizationID: 1,
			UserID:         1,
			UserRole:       "owner",
			IPAddress:      "127.0.0.1",
			UserAgent:      "test-agent",
		})
		if err != nil {
			t.Errorf("Redeem() error = %v", err)
			return
		}
		if !resp.Success {
			t.Errorf("Redeem() failed: %v", resp.MessageCode)
			return
		}
		if resp.PlanName != "pro" {
			t.Errorf("Redeem() plan_name = %v, want pro", resp.PlanName)
		}
		if resp.DurationMonths != 3 {
			t.Errorf("Redeem() duration_months = %v, want 3", resp.DurationMonths)
		}

		// Verify subscription was created
		var sub billing.Subscription
		if err := db.Where("organization_id = ?", 1).First(&sub).Error; err != nil {
			t.Errorf("Subscription not created: %v", err)
		}

		// Verify redemption record was created
		var redemption promocode.Redemption
		if err := db.Where("organization_id = ?", 1).First(&redemption).Error; err != nil {
			t.Errorf("Redemption record not created: %v", err)
		}

		// Verify promo code used_count was incremented
		var code promocode.PromoCode
		if err := db.Where("code = ?", "REDEEM2024").First(&code).Error; err != nil {
			t.Errorf("Failed to get promo code: %v", err)
		}
		if code.UsedCount != 1 {
			t.Errorf("PromoCode used_count = %v, want 1", code.UsedCount)
		}
	})

	// Test: cannot redeem same code twice
	t.Run("cannot redeem same code twice", func(t *testing.T) {
		resp, err := service.Redeem(ctx, &svc.RedeemRequest{
			Code:           "REDEEM2024",
			OrganizationID: 1,
			UserID:         1,
			UserRole:       "owner",
		})
		if err != nil {
			t.Errorf("Redeem() error = %v", err)
			return
		}
		if resp.Success {
			t.Error("Redeem() should fail for already used code")
		}
	})
}

func TestService_RedeemExtendsExistingSubscription(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	// Create test promo code
	now := time.Now()
	past := now.AddDate(0, -1, 0)
	future := now.AddDate(0, 1, 0)

	db.Create(&promocode.PromoCode{
		Code:           "EXTEND2024",
		Name:           "Extend Promo",
		Type:           promocode.PromoTypeMedia,
		PlanName:       "pro",
		DurationMonths: 2,
		IsActive:       true,
		StartsAt:       past,
		ExpiresAt:      &future,
	})

	// Create existing subscription
	existingEnd := now.AddDate(0, 1, 0) // 1 month from now
	db.Create(&billing.Subscription{
		OrganizationID:     2,
		PlanID:             2, // pro
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   existingEnd,
	})
	db.Exec(`INSERT INTO organizations (id, name, slug) VALUES (2, 'Test Org 2', 'test-org-2')`)

	t.Run("extends existing subscription", func(t *testing.T) {
		resp, err := service.Redeem(ctx, &svc.RedeemRequest{
			Code:           "EXTEND2024",
			OrganizationID: 2,
			UserID:         1,
			UserRole:       "owner",
		})
		if err != nil {
			t.Errorf("Redeem() error = %v", err)
			return
		}
		if !resp.Success {
			t.Errorf("Redeem() failed: %v", resp.MessageCode)
			return
		}

		// Verify subscription was extended (existing 1 month + 2 months = 3 months from now)
		var sub billing.Subscription
		if err := db.Where("organization_id = ?", 2).First(&sub).Error; err != nil {
			t.Errorf("Subscription not found: %v", err)
			return
		}

		expectedEnd := existingEnd.AddDate(0, 2, 0) // extend by 2 months
		// Allow 1 second tolerance for time comparison
		if sub.CurrentPeriodEnd.Sub(expectedEnd).Abs() > time.Second {
			t.Errorf("Subscription end = %v, want ~%v", sub.CurrentPeriodEnd, expectedEnd)
		}
	})
}
