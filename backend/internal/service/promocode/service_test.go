package promocode

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/domain/promocode"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Create tables for SQLite
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			username TEXT NOT NULL UNIQUE,
			name TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS organizations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			subscription_plan TEXT NOT NULL DEFAULT 'free',
			subscription_status TEXT NOT NULL DEFAULT 'active',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create organizations table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS subscription_plans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL,
			price_per_seat_monthly REAL NOT NULL DEFAULT 0,
			included_pod_minutes INTEGER NOT NULL DEFAULT 0,
			price_per_extra_minute REAL NOT NULL DEFAULT 0,
			max_users INTEGER NOT NULL,
			max_runners INTEGER NOT NULL,
			max_concurrent_pods INTEGER NOT NULL DEFAULT 0,
			max_repositories INTEGER NOT NULL,
			features BLOB,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create subscription_plans table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS subscriptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL UNIQUE,
			plan_id INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			billing_cycle TEXT NOT NULL DEFAULT 'monthly',
			current_period_start DATETIME NOT NULL,
			current_period_end DATETIME NOT NULL,
			stripe_customer_id TEXT,
			stripe_subscription_id TEXT,
			custom_quotas TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create subscriptions table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS promo_codes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			description TEXT,
			type TEXT NOT NULL,
			plan_name TEXT NOT NULL,
			duration_months INTEGER NOT NULL,
			max_uses INTEGER,
			used_count INTEGER NOT NULL DEFAULT 0,
			max_uses_per_org INTEGER NOT NULL DEFAULT 1,
			starts_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_by_id INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create promo_codes table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS promo_code_redemptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			promo_code_id INTEGER NOT NULL,
			organization_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			plan_name TEXT NOT NULL,
			duration_months INTEGER NOT NULL,
			previous_plan_name TEXT,
			previous_period_end DATETIME,
			new_period_end DATETIME NOT NULL,
			ip_address TEXT,
			user_agent TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create promo_code_redemptions table: %v", err)
	}

	// Insert test data with proper BLOB for features
	db.Exec(`INSERT INTO users (id, email, username, name) VALUES (1, 'test@example.com', 'testuser', 'Test User')`)
	db.Exec(`INSERT INTO organizations (id, name, slug) VALUES (1, 'Test Org', 'test-org')`)
	db.Exec(`INSERT INTO subscription_plans (id, name, display_name, max_users, max_runners, max_repositories, features) VALUES (1, 'free', 'Free', 1, 1, 3, X'7B7D')`)        // {} as hex
	db.Exec(`INSERT INTO subscription_plans (id, name, display_name, max_users, max_runners, max_repositories, price_per_seat_monthly, features) VALUES (2, 'pro', 'Pro', 5, 10, 10, 20, X'7B7D')`)
	db.Exec(`INSERT INTO subscription_plans (id, name, display_name, max_users, max_runners, max_repositories, price_per_seat_monthly, features) VALUES (3, 'enterprise', 'Enterprise', 50, 100, -1, 40, X'7B7D')`)

	return db
}

func TestService_Create(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		req     *CreateRequest
		wantErr bool
	}{
		{
			name: "create valid promo code",
			req: &CreateRequest{
				Code:           "TEST2024",
				Name:           "Test Promo",
				Description:    "Test promo code",
				Type:           promocode.PromoTypeMedia,
				PlanName:       "pro",
				DurationMonths: 3,
				CreatedByID:    1,
			},
			wantErr: false,
		},
		{
			name: "create with max uses",
			req: &CreateRequest{
				Code:           "LIMITED100",
				Name:           "Limited Promo",
				Type:           promocode.PromoTypeCampaign,
				PlanName:       "pro",
				DurationMonths: 1,
				MaxUses:        intPtr(100),
				CreatedByID:    1,
			},
			wantErr: false,
		},
		{
			name: "create with invalid plan",
			req: &CreateRequest{
				Code:           "INVALID",
				Name:           "Invalid",
				Type:           promocode.PromoTypeInternal,
				PlanName:       "nonexistent",
				DurationMonths: 1,
				CreatedByID:    1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.Create(ctx, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got == nil {
					t.Error("Create() returned nil")
					return
				}
				if got.Code != tt.req.Code {
					t.Errorf("Create() code = %v, want %v", got.Code, tt.req.Code)
				}
				if got.PlanName != tt.req.PlanName {
					t.Errorf("Create() plan_name = %v, want %v", got.PlanName, tt.req.PlanName)
				}
			}
		})
	}
}

func TestService_Validate(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
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
			resp, err := svc.Validate(ctx, &ValidateRequest{
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
	svc := NewService(db)
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
		resp, err := svc.Redeem(ctx, &RedeemRequest{
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
		resp, err := svc.Redeem(ctx, &RedeemRequest{
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
		resp, err := svc.Redeem(ctx, &RedeemRequest{
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
	svc := NewService(db)
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
		resp, err := svc.Redeem(ctx, &RedeemRequest{
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

func TestService_Deactivate(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
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
		err := svc.Deactivate(ctx, code.ID)
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
		err := svc.Deactivate(ctx, 99999)
		if err == nil {
			t.Error("Deactivate() should fail for nonexistent code")
		}
	})
}

func TestService_List(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
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
		codes, total, err := svc.List(ctx, &promocode.ListFilter{
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
		codes, total, err := svc.List(ctx, &promocode.ListFilter{
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
		codes, total, err := svc.List(ctx, &promocode.ListFilter{
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

// Helper function
func intPtr(i int) *int {
	return &i
}
