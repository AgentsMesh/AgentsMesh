package webhooks

import (
	"log/slog"
	"os"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	billingService "github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ===========================================
// Integration Test Setup
// ===========================================

func setupIntegrationDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Create required tables
	db.Exec(`CREATE TABLE IF NOT EXISTS subscription_plans (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		display_name TEXT,
		tier TEXT DEFAULT 'based',
		price_per_seat_monthly REAL DEFAULT 0,
		price_per_seat_yearly REAL DEFAULT 0,
		max_users INTEGER DEFAULT 1,
		max_runners INTEGER DEFAULT 1,
		max_repositories INTEGER DEFAULT 10,
		max_concurrent_pods INTEGER DEFAULT 1,
		included_pod_minutes INTEGER DEFAULT 100,
		max_storage_gb INTEGER DEFAULT 1,
		stripe_price_id_monthly TEXT,
		stripe_price_id_yearly TEXT,
		features TEXT,
		is_active BOOLEAN DEFAULT TRUE
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS subscriptions (
		id INTEGER PRIMARY KEY,
		organization_id INTEGER NOT NULL,
		plan_id INTEGER,
		status TEXT DEFAULT 'active',
		stripe_subscription_id TEXT,
		stripe_customer_id TEXT,
		billing_cycle TEXT DEFAULT 'monthly',
		seat_count INTEGER DEFAULT 1,
		auto_renew BOOLEAN DEFAULT TRUE,
		current_period_start DATETIME,
		current_period_end DATETIME,
		cancel_at_period_end BOOLEAN DEFAULT FALSE,
		downgrade_to_plan TEXT,
		next_billing_cycle TEXT,
		frozen_at DATETIME,
		canceled_at DATETIME,
		custom_quotas TEXT,
		payment_provider TEXT,
		payment_method TEXT,
		alipay_agreement_no TEXT,
		wechat_contract_id TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS payment_orders (
		id INTEGER PRIMARY KEY,
		organization_id INTEGER NOT NULL,
		order_no TEXT UNIQUE,
		external_order_no TEXT,
		order_type TEXT,
		plan_id INTEGER,
		billing_cycle TEXT,
		seats INTEGER DEFAULT 1,
		amount REAL,
		actual_amount REAL,
		payment_provider TEXT,
		payment_method TEXT,
		status TEXT,
		paid_at DATETIME,
		failure_reason TEXT,
		created_by_id INTEGER,
		created_at DATETIME,
		updated_at DATETIME
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS payment_transactions (
		id INTEGER PRIMARY KEY,
		payment_order_id INTEGER,
		transaction_type TEXT,
		external_transaction_id TEXT,
		amount REAL,
		currency TEXT,
		status TEXT,
		webhook_event_id TEXT,
		webhook_event_type TEXT,
		raw_payload TEXT,
		created_at DATETIME
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS usage_records (
		id INTEGER PRIMARY KEY,
		subscription_id INTEGER NOT NULL,
		usage_type TEXT,
		amount REAL,
		metadata TEXT,
		created_at DATETIME
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS organization_members (
		organization_id INTEGER,
		user_id INTEGER,
		role TEXT
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS runners (
		id INTEGER PRIMARY KEY,
		organization_id INTEGER,
		name TEXT
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS repositories (
		id INTEGER PRIMARY KEY,
		organization_id INTEGER,
		name TEXT
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS pods (
		id INTEGER PRIMARY KEY,
		organization_id INTEGER,
		name TEXT,
		status TEXT
	)`)

	// Seed test data
	db.Exec(`INSERT INTO subscription_plans (name, display_name, tier, price_per_seat_monthly, price_per_seat_yearly, max_users, max_runners, max_repositories, max_concurrent_pods, included_pod_minutes, is_active)
		VALUES ('based', 'Based', 'based', 0, 0, 5, 1, 3, 2, 100, TRUE)`)
	db.Exec(`INSERT INTO subscription_plans (name, display_name, tier, price_per_seat_monthly, price_per_seat_yearly, max_users, max_runners, max_repositories, max_concurrent_pods, included_pod_minutes, is_active)
		VALUES ('pro', 'Pro', 'pro', 19.99, 199.90, 50, 10, 100, 10, 5000, TRUE)`)
	db.Exec(`INSERT INTO subscription_plans (name, display_name, tier, price_per_seat_monthly, price_per_seat_yearly, max_users, max_runners, max_repositories, max_concurrent_pods, included_pod_minutes, is_active)
		VALUES ('enterprise', 'Enterprise', 'enterprise', 99.99, 999.90, -1, -1, -1, -1, -1, TRUE)`)

	return db
}

func integrationTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func createMockRouter(t *testing.T) (*WebhookRouter, *gorm.DB, *payment.Factory) {
	db := setupIntegrationDB(t)
	logger := integrationTestLogger()
	registry := NewHandlerRegistry(logger)
	SetupDefaultHandlers(registry, logger)

	// Create full config with mock payment enabled
	cfg := &config.Config{
		PrimaryDomain: "localhost:3000",
		UseHTTPS:      false,
		Payment: config.PaymentConfig{
			DeploymentType: config.DeploymentGlobal,
			MockEnabled:    true,
			MockBaseURL:    "http://localhost:3000",
		},
	}

	// Create billing service with mock (uses full config for URL derivation)
	billingSvc := billingService.NewServiceWithConfig(infra.NewBillingRepository(db), cfg)
	factory := billingSvc.GetPaymentFactory()

	return &WebhookRouter{
		db:             db,
		cfg:            cfg,
		logger:         logger,
		registry:       registry,
		billingSvc:     billingSvc,
		paymentFactory: factory,
	}, db, factory
}
