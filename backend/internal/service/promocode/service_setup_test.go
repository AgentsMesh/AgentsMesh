package promocode_test

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/promocode"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	svc "github.com/anthropics/agentsmesh/backend/internal/service/promocode"
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
			is_system_admin INTEGER NOT NULL DEFAULT 0,
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
			subscription_plan TEXT NOT NULL DEFAULT 'based',
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
			payment_provider TEXT,
			payment_method TEXT,
			auto_renew INTEGER NOT NULL DEFAULT 0,
			seat_count INTEGER NOT NULL DEFAULT 1,
			stripe_customer_id TEXT,
			stripe_subscription_id TEXT,
			lemonsqueezy_customer_id TEXT,
			lemonsqueezy_subscription_id TEXT,
			alipay_agreement_no TEXT,
			wechat_contract_id TEXT,
			canceled_at DATETIME,
			cancel_at_period_end INTEGER NOT NULL DEFAULT 0,
			frozen_at DATETIME,
			downgrade_to_plan TEXT,
			next_billing_cycle TEXT,
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
	db.Exec(`INSERT INTO subscription_plans (id, name, display_name, max_users, max_runners, max_repositories, features) VALUES (1, 'based', 'Based', 1, 1, 3, X'7B7D')`)        // {} as hex
	db.Exec(`INSERT INTO subscription_plans (id, name, display_name, max_users, max_runners, max_repositories, price_per_seat_monthly, features) VALUES (2, 'pro', 'Pro', 5, 10, 10, 20, X'7B7D')`)
	db.Exec(`INSERT INTO subscription_plans (id, name, display_name, max_users, max_runners, max_repositories, price_per_seat_monthly, features) VALUES (3, 'enterprise', 'Enterprise', 50, 100, -1, 40, X'7B7D')`)

	return db
}

// newTestService creates a Service with real infra implementations for integration testing
func newTestService(db *gorm.DB) *svc.Service {
	return svc.NewService(infra.NewPromocodeRepository(db), infra.NewGormBillingProvider(db))
}

// Helper function
func intPtr(i int) *int {
	return &i
}

func TestService_Create(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	tests := []struct {
		name    string
		req     *svc.CreateRequest
		wantErr bool
	}{
		{
			name: "create valid promo code",
			req: &svc.CreateRequest{
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
			req: &svc.CreateRequest{
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
			req: &svc.CreateRequest{
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
			got, err := service.Create(ctx, tt.req)
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
