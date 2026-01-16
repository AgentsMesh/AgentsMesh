package billing

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
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

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS subscription_plans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL,
			price_per_seat_monthly REAL NOT NULL DEFAULT 0,
			included_pod_minutes INTEGER NOT NULL DEFAULT 0,
			price_per_extra_minute REAL NOT NULL DEFAULT 0,
			max_users INTEGER NOT NULL DEFAULT 0,
			max_runners INTEGER NOT NULL DEFAULT 0,
			max_concurrent_pods INTEGER NOT NULL DEFAULT 0,
			max_repositories INTEGER NOT NULL DEFAULT 0,
			features TEXT,
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
		CREATE TABLE IF NOT EXISTS usage_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			usage_type TEXT NOT NULL,
			quantity REAL NOT NULL DEFAULT 0,
			period_start DATETIME NOT NULL,
			period_end DATETIME NOT NULL,
			metadata TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create usage_records table: %v", err)
	}

	return db
}

func seedTestPlan(t *testing.T, db *gorm.DB) *billing.SubscriptionPlan {
	plan := &billing.SubscriptionPlan{
		Name:                "free",
		DisplayName:         "Free Plan",
		PricePerSeatMonthly: 0,
		IncludedPodMinutes:  100,
		PricePerExtraMinute: 0,
		MaxUsers:            5,
		MaxRunners:          1,
		MaxConcurrentPods:   2,
		MaxRepositories:     3,
		IsActive:            true,
	}
	if err := db.Create(plan).Error; err != nil {
		t.Fatalf("failed to seed plan: %v", err)
	}
	return plan
}

func TestNewService(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, "")

	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if service.stripeEnabled {
		t.Error("expected stripe to be disabled without key")
	}
}

func TestNewServiceWithStripeKey(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, "sk_test_fake_key")

	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if !service.stripeEnabled {
		t.Error("expected stripe to be enabled with key")
	}
}

func TestGetPlan(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, "")
	ctx := context.Background()

	seedTestPlan(t, db)

	plan, err := service.GetPlan(ctx, "free")
	if err != nil {
		t.Fatalf("failed to get plan: %v", err)
	}
	if plan.Name != "free" {
		t.Errorf("expected plan name 'free', got %s", plan.Name)
	}
}

func TestGetPlanNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, "")
	ctx := context.Background()

	_, err := service.GetPlan(ctx, "nonexistent")
	if err != ErrPlanNotFound {
		t.Errorf("expected ErrPlanNotFound, got %v", err)
	}
}

func TestListPlans(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, "")
	ctx := context.Background()

	seedTestPlan(t, db)

	// Add another plan
	db.Exec(`INSERT INTO subscription_plans (name, display_name, price_per_seat_monthly, included_pod_minutes, max_users, max_runners, max_repositories, is_active)
		VALUES ('pro', 'Pro Plan', 1000, 500, 20, 5, 50, 1)`)

	plans, err := service.ListPlans(ctx)
	if err != nil {
		t.Fatalf("failed to list plans: %v", err)
	}
	if len(plans) != 2 {
		t.Errorf("expected 2 plans, got %d", len(plans))
	}
}

func TestGetSubscription(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, "")
	ctx := context.Background()

	plan := seedTestPlan(t, db)

	now := time.Now()
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             plan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
	}
	db.Create(sub)

	result, err := service.GetSubscription(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get subscription: %v", err)
	}
	if result.OrganizationID != 1 {
		t.Errorf("expected org ID 1, got %d", result.OrganizationID)
	}
}

func TestGetSubscriptionNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, "")
	ctx := context.Background()

	_, err := service.GetSubscription(ctx, 999)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

func TestCreateSubscription(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, "")
	ctx := context.Background()

	seedTestPlan(t, db)

	sub, err := service.CreateSubscription(ctx, 1, "free")
	if err != nil {
		t.Fatalf("failed to create subscription: %v", err)
	}
	if sub.OrganizationID != 1 {
		t.Errorf("expected org ID 1, got %d", sub.OrganizationID)
	}
	if sub.Status != billing.SubscriptionStatusActive {
		t.Errorf("expected status active, got %s", sub.Status)
	}
}

func TestCreateSubscriptionPlanNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, "")
	ctx := context.Background()

	_, err := service.CreateSubscription(ctx, 1, "nonexistent")
	if err != ErrPlanNotFound {
		t.Errorf("expected ErrPlanNotFound, got %v", err)
	}
}

func TestUpdateSubscription(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, "")
	ctx := context.Background()

	// Create two plans
	seedTestPlan(t, db)
	db.Exec(`INSERT INTO subscription_plans (name, display_name, price_per_seat_monthly, included_pod_minutes, max_users, max_runners, max_repositories, is_active)
		VALUES ('pro', 'Pro Plan', 1000, 500, 20, 5, 50, 1)`)

	// Create subscription with free plan
	service.CreateSubscription(ctx, 1, "free")

	// Update to pro
	sub, err := service.UpdateSubscription(ctx, 1, "pro")
	if err != nil {
		t.Fatalf("failed to update subscription: %v", err)
	}
	if sub.Plan.Name != "pro" {
		t.Errorf("expected plan 'pro', got %s", sub.Plan.Name)
	}
}

func TestCancelSubscription(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "free")

	err := service.CancelSubscription(ctx, 1)
	if err != nil {
		t.Fatalf("failed to cancel subscription: %v", err)
	}

	sub, _ := service.GetSubscription(ctx, 1)
	if sub.Status != billing.SubscriptionStatusCanceled {
		t.Errorf("expected status canceled, got %s", sub.Status)
	}
}

func TestRecordUsage(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, "")
	ctx := context.Background()

	// Create subscription first
	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "free")

	err := service.RecordUsage(ctx, 1, "pod_minutes", 5.0, billing.UsageMetadata{})
	if err != nil {
		t.Fatalf("failed to record usage: %v", err)
	}
}

func TestGetUsage(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, "")
	ctx := context.Background()

	// Create subscription first
	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "free")

	service.RecordUsage(ctx, 1, "pod_minutes", 5.0, billing.UsageMetadata{})
	service.RecordUsage(ctx, 1, "pod_minutes", 3.0, billing.UsageMetadata{})

	usage, err := service.GetUsage(ctx, 1, "pod_minutes")
	if err != nil {
		t.Fatalf("failed to get usage: %v", err)
	}
	if usage != 8.0 {
		t.Errorf("expected usage 8.0, got %f", usage)
	}
}

func TestGetUsageHistory(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, "")
	ctx := context.Background()

	// Create subscription first
	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "free")

	service.RecordUsage(ctx, 1, "pod_minutes", 5.0, billing.UsageMetadata{})

	records, err := service.GetUsageHistory(ctx, 1, "pod_minutes", 10)
	if err != nil {
		t.Fatalf("failed to get usage history: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}
}

func TestCheckQuota(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "free")

	// Should not exceed quota (requestedAmount=1)
	err := service.CheckQuota(ctx, 1, "users", 1)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestGetPlanByID(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, "")
	ctx := context.Background()

	plan := seedTestPlan(t, db)

	result, err := service.GetPlanByID(ctx, plan.ID)
	if err != nil {
		t.Fatalf("failed to get plan by ID: %v", err)
	}
	if result.Name != "free" {
		t.Errorf("expected plan name 'free', got %s", result.Name)
	}
}

func TestRenewSubscription(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db, "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "free")

	err := service.RenewSubscription(ctx, 1)
	if err != nil {
		t.Fatalf("failed to renew subscription: %v", err)
	}

	// Verify subscription is still active after renewal
	sub, _ := service.GetSubscription(ctx, 1)
	if sub.Status != billing.SubscriptionStatusActive {
		t.Errorf("expected status active, got %s", sub.Status)
	}
}

func TestErrorVariables(t *testing.T) {
	if ErrSubscriptionNotFound.Error() != "subscription not found" {
		t.Errorf("unexpected error message: %s", ErrSubscriptionNotFound.Error())
	}
	if ErrPlanNotFound.Error() != "plan not found" {
		t.Errorf("unexpected error message: %s", ErrPlanNotFound.Error())
	}
	if ErrQuotaExceeded.Error() != "quota exceeded" {
		t.Errorf("unexpected error message: %s", ErrQuotaExceeded.Error())
	}
	if ErrInvalidPlan.Error() != "invalid plan" {
		t.Errorf("unexpected error message: %s", ErrInvalidPlan.Error())
	}
}
