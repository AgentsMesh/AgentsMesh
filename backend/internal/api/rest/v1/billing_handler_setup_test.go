package v1

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupBillingTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	createBillingTestTables(db)
	seedBillingTestData(db)

	return db
}

func createBillingTestTables(db *gorm.DB) {
	db.Exec(`CREATE TABLE IF NOT EXISTS subscription_plans (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		display_name TEXT NOT NULL,
		tier TEXT DEFAULT 'based',
		price_per_seat_monthly REAL NOT NULL DEFAULT 0,
		price_per_seat_yearly REAL DEFAULT 0,
		included_pod_minutes INTEGER NOT NULL DEFAULT 0,
		price_per_extra_minute REAL NOT NULL DEFAULT 0,
		max_users INTEGER NOT NULL,
		max_runners INTEGER NOT NULL,
		max_concurrent_pods INTEGER NOT NULL DEFAULT 0,
		max_repositories INTEGER NOT NULL,
		features BLOB,
		is_active INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS subscriptions (
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
		alipay_agreement_no TEXT,
		wechat_contract_id TEXT,
		lemonsqueezy_customer_id TEXT,
		lemonsqueezy_subscription_id TEXT,
		canceled_at DATETIME,
		cancel_at_period_end INTEGER NOT NULL DEFAULT 0,
		frozen_at DATETIME,
		downgrade_to_plan TEXT,
		next_billing_cycle TEXT,
		custom_quotas TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS plan_prices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		plan_id INTEGER NOT NULL,
		currency TEXT NOT NULL DEFAULT 'usd',
		price_per_seat_monthly REAL NOT NULL DEFAULT 0,
		price_per_seat_yearly REAL DEFAULT 0,
		stripe_price_id_monthly TEXT,
		stripe_price_id_yearly TEXT,
		lemonsqueezy_variant_id_monthly TEXT,
		lemonsqueezy_variant_id_yearly TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(plan_id, currency)
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS usage_records (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		resource_type TEXT NOT NULL,
		quantity INTEGER NOT NULL,
		period_start DATETIME NOT NULL,
		period_end DATETIME NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS invoices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		organization_id INTEGER NOT NULL,
		invoice_no TEXT NOT NULL UNIQUE,
		amount REAL NOT NULL,
		currency TEXT NOT NULL DEFAULT 'usd',
		status TEXT NOT NULL DEFAULT 'pending',
		due_date DATETIME,
		paid_at DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
}

func seedBillingTestData(db *gorm.DB) {
	db.Exec(`INSERT INTO subscription_plans (id, name, display_name, tier, price_per_seat_monthly, max_users, max_runners, max_repositories, max_concurrent_pods, features, is_active)
		VALUES (1, 'based', 'Based', 'based', 0, 1, 1, 3, 1, X'7B7D', 1)`)
	db.Exec(`INSERT INTO subscription_plans (id, name, display_name, tier, price_per_seat_monthly, max_users, max_runners, max_repositories, max_concurrent_pods, features, is_active)
		VALUES (2, 'pro', 'Pro', 'pro', 20, 10, 10, 20, 5, X'7B7D', 1)`)
	db.Exec(`INSERT INTO subscription_plans (id, name, display_name, tier, price_per_seat_monthly, max_users, max_runners, max_repositories, max_concurrent_pods, features, is_active)
		VALUES (3, 'enterprise', 'Enterprise', 'enterprise', 40, 50, 100, -1, 20, X'7B7D', 1)`)

	db.Exec(`INSERT INTO plan_prices (plan_id, currency, price_per_seat_monthly, price_per_seat_yearly)
		VALUES (1, 'usd', 0, 0)`)
	db.Exec(`INSERT INTO plan_prices (plan_id, currency, price_per_seat_monthly, price_per_seat_yearly)
		VALUES (2, 'usd', 20, 200)`)
	db.Exec(`INSERT INTO plan_prices (plan_id, currency, price_per_seat_monthly, price_per_seat_yearly)
		VALUES (3, 'usd', 40, 400)`)
}

func setupBillingHandler(t *testing.T) (*BillingHandler, *gorm.DB, *gin.Engine) {
	db := setupBillingTestDB(t)
	billingSvc := billing.NewService(infra.NewBillingRepository(db), "")
	handler := NewBillingHandler(billingSvc)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	return handler, db, router
}

func setBillingTenantContext(c *gin.Context, orgID int64, userID int64, role string) {
	tc := &middleware.TenantContext{
		OrganizationID:   orgID,
		OrganizationSlug: "test-org",
		UserID:           userID,
		UserRole:         role,
	}
	c.Set("tenant", tc)
}
