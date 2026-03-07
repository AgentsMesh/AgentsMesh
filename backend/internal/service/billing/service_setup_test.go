package billing

import (
	"net/http/httptest"
	"testing"

	billingdomain "github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/gin-gonic/gin"
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

	// Create all required tables
	tables := getTestDBTableStatements()

	for _, sql := range tables {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatalf("failed to create table: %v", err)
		}
	}

	return db
}

// newTestRepo creates a BillingRepository from *gorm.DB for tests.
func newTestRepo(db *gorm.DB) billingdomain.BillingRepository {
	return infra.NewBillingRepository(db)
}

// setupTestService creates a test service with seeded plans
func setupTestService(t *testing.T) (*Service, *gorm.DB) {
	db := setupTestDB(t)
	svc := NewService(newTestRepo(db), "")

	// Seed standard plans: based (ID=1), pro (ID=2), enterprise (ID=3)
	seedTestPlan(t, db)
	seedProPlan(t, db)
	seedEnterprisePlan(t, db)

	return svc, db
}

func createTestGinContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/", nil)
	return c, w
}

func getTestDBTableStatements() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS subscription_plans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL,
			price_per_seat_monthly REAL NOT NULL DEFAULT 0,
			price_per_seat_yearly REAL NOT NULL DEFAULT 0,
			included_pod_minutes INTEGER NOT NULL DEFAULT 0,
			price_per_extra_minute REAL NOT NULL DEFAULT 0,
			max_users INTEGER NOT NULL DEFAULT 0,
			max_runners INTEGER NOT NULL DEFAULT 0,
			max_concurrent_pods INTEGER NOT NULL DEFAULT 0,
			max_repositories INTEGER NOT NULL DEFAULT 0,
			features TEXT,
			stripe_price_id_monthly TEXT,
			stripe_price_id_yearly TEXT,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS subscriptions (
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
		)`,
		`CREATE TABLE IF NOT EXISTS usage_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			usage_type TEXT NOT NULL,
			quantity REAL NOT NULL DEFAULT 0,
			period_start DATETIME NOT NULL,
			period_end DATETIME NOT NULL,
			metadata TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		getPaymentOrdersTableSQL(),
		getPaymentTransactionsTableSQL(),
		getInvoicesTableSQL(),
		getAuxiliaryTablesSQL(),
	}
}
