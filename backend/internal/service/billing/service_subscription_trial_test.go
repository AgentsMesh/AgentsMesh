package billing

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// ===========================================
// Trial Subscription Tests
// ===========================================

func TestCreateTrialSubscription(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)

	sub, err := service.CreateTrialSubscription(ctx, 1, "based", 30)
	if err != nil {
		t.Fatalf("failed to create trial subscription: %v", err)
	}
	if sub.Status != billing.SubscriptionStatusTrialing {
		t.Errorf("expected status trialing, got %s", sub.Status)
	}
	if sub.SeatCount != 1 {
		t.Errorf("expected seat count 1, got %d", sub.SeatCount)
	}

	// Verify period is 30 days
	expectedEnd := sub.CurrentPeriodStart.AddDate(0, 0, 30)
	if !sub.CurrentPeriodEnd.Truncate(time.Second).Equal(expectedEnd.Truncate(time.Second)) {
		t.Errorf("expected 30 day trial period, got %v", sub.CurrentPeriodEnd.Sub(sub.CurrentPeriodStart))
	}
}

func TestCreateTrialSubscriptionDefaultDays(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)

	// Pass 0 days - should use default
	sub, err := service.CreateTrialSubscription(ctx, 1, "based", 0)
	if err != nil {
		t.Fatalf("failed to create trial subscription: %v", err)
	}

	// Verify period uses default (30 days)
	expectedEnd := sub.CurrentPeriodStart.AddDate(0, 0, billing.DefaultTrialDays)
	if !sub.CurrentPeriodEnd.Truncate(time.Second).Equal(expectedEnd.Truncate(time.Second)) {
		t.Errorf("expected default trial period, got %v", sub.CurrentPeriodEnd.Sub(sub.CurrentPeriodStart))
	}
}

func TestCreateTrialSubscriptionPlanNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	_, err := service.CreateTrialSubscription(ctx, 1, "nonexistent", 30)
	if err != ErrPlanNotFound {
		t.Errorf("expected ErrPlanNotFound, got %v", err)
	}
}

func TestActivateTrialSubscription(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)

	// Create trial subscription
	service.CreateTrialSubscription(ctx, 1, "based", 30)

	// Activate with monthly billing
	err := service.ActivateTrialSubscription(ctx, 1, billing.BillingCycleMonthly)
	if err != nil {
		t.Fatalf("failed to activate trial: %v", err)
	}

	sub, _ := service.GetSubscription(ctx, 1)
	if sub.Status != billing.SubscriptionStatusActive {
		t.Errorf("expected status active, got %s", sub.Status)
	}
	if sub.BillingCycle != billing.BillingCycleMonthly {
		t.Errorf("expected billing cycle monthly, got %s", sub.BillingCycle)
	}
}

func TestActivateTrialSubscriptionYearly(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)

	// Create trial subscription
	service.CreateTrialSubscription(ctx, 1, "based", 30)

	// Activate with yearly billing
	err := service.ActivateTrialSubscription(ctx, 1, billing.BillingCycleYearly)
	if err != nil {
		t.Fatalf("failed to activate trial: %v", err)
	}

	sub, _ := service.GetSubscription(ctx, 1)
	if sub.Status != billing.SubscriptionStatusActive {
		t.Errorf("expected status active, got %s", sub.Status)
	}
	if sub.BillingCycle != billing.BillingCycleYearly {
		t.Errorf("expected billing cycle yearly, got %s", sub.BillingCycle)
	}

	// Verify period is 1 year
	expectedEnd := sub.CurrentPeriodStart.AddDate(1, 0, 0)
	if !sub.CurrentPeriodEnd.Truncate(time.Second).Equal(expectedEnd.Truncate(time.Second)) {
		t.Errorf("expected 1 year period for yearly billing")
	}
}

func TestActivateTrialSubscriptionAlreadyActive(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)

	// Create active subscription (not trialing)
	service.CreateSubscription(ctx, 1, "based")

	// Try to activate - should do nothing (already active)
	err := service.ActivateTrialSubscription(ctx, 1, billing.BillingCycleMonthly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := service.GetSubscription(ctx, 1)
	if sub.Status != billing.SubscriptionStatusActive {
		t.Errorf("expected status to remain active")
	}
}

func TestActivateTrialSubscriptionNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	err := service.ActivateTrialSubscription(ctx, 999, billing.BillingCycleMonthly)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}
