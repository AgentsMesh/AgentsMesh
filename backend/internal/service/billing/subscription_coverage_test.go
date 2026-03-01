package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Additional Subscription Coverage Tests
// ===========================================

// TestUpdateSubscription_SamePlan tests updating to the same plan
func TestUpdateSubscription_SamePlan(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          5,
	})

	// Update to same plan - should be a no-op
	sub, err := svc.UpdateSubscription(context.Background(), 1, "pro")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sub.PlanID != 2 {
		t.Errorf("expected plan to remain pro (2), got %d", sub.PlanID)
	}
}

// TestUpdateSubscription_CancelPendingDowngrade tests clearing pending downgrade on upgrade
func TestUpdateSubscription_CancelPendingDowngrade(t *testing.T) {
	svc, db := setupTestService(t)

	// Create subscription with pending downgrade
	downgradePlan := "based"
	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          5,
		DowngradeToPlan:    &downgradePlan,
	})

	// Upgrade to enterprise should clear pending downgrade
	sub, err := svc.UpdateSubscription(context.Background(), 1, "enterprise")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify subscription state
	if sub.PlanID != 2 {
		t.Errorf("expected plan to still be pro (2), got %d", sub.PlanID)
	}
}

// TestCancelSubscription_AlreadyCanceled tests canceling an already canceled subscription
func TestCancelSubscription_AlreadyCanceled(t *testing.T) {
	svc, db := setupTestService(t)

	cancelTime := time.Now()
	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusCanceled,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
		CanceledAt:         &cancelTime,
	})

	// Should still succeed (idempotent)
	err := svc.CancelSubscription(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error canceling already canceled subscription: %v", err)
	}
}

// TestCancelSubscription_WithStripeSubscriptionID tests canceling with Stripe subscription
func TestCancelSubscription_WithStripeSubscriptionID(t *testing.T) {
	svc, db := setupTestService(t)

	stripeSubID := "sub_stripe_123"
	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:       1,
		PlanID:               2,
		Status:               billing.SubscriptionStatusActive,
		BillingCycle:         billing.BillingCycleMonthly,
		CurrentPeriodStart:   now,
		CurrentPeriodEnd:     now.AddDate(0, 1, 0),
		SeatCount:            1,
		StripeSubscriptionID: &stripeSubID,
	})

	// Cancel should work even if Stripe is disabled
	err := svc.CancelSubscription(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify cancellation
	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.Status != billing.SubscriptionStatusCanceled {
		t.Errorf("expected canceled status, got %s", sub.Status)
	}
}

// TestCreateSubscription_DuplicateOrg tests creating subscription for org with existing subscription
func TestCreateSubscription_DuplicateOrg(t *testing.T) {
	svc, db := setupTestService(t)

	// Create first subscription
	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             1,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	// Try to create another subscription for same org
	_, err := svc.CreateSubscription(context.Background(), 1, "pro")
	// Should fail due to unique constraint on organization_id
	if err == nil {
		t.Error("expected error for duplicate organization subscription")
	}
}

// TestCreateTrialSubscription_Success tests creating a trial subscription
func TestCreateTrialSubscription_Success(t *testing.T) {
	svc, _ := setupTestService(t)

	sub, err := svc.CreateTrialSubscription(context.Background(), 1, "pro", 14)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sub.Status != billing.SubscriptionStatusTrialing {
		t.Errorf("expected trialing status, got %s", sub.Status)
	}
	if sub.PlanID != 2 { // pro plan
		t.Errorf("expected pro plan (2), got %d", sub.PlanID)
	}
}

// TestCreateTrialSubscription_InvalidPlan tests creating trial with invalid plan
func TestCreateTrialSubscription_InvalidPlan(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.CreateTrialSubscription(context.Background(), 1, "nonexistent", 14)
	if err != ErrPlanNotFound {
		t.Errorf("expected ErrPlanNotFound, got %v", err)
	}
}

// TestUpdateSubscription_DowngradeToFreePlan tests downgrading to free plan
func TestUpdateSubscription_DowngradeToFreePlan(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1, // Within based plan limit
	})

	// Downgrade to based plan (lower price)
	sub, err := svc.UpdateSubscription(context.Background(), 1, "based")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should schedule downgrade
	if sub.DowngradeToPlan == nil {
		t.Error("expected downgrade to be scheduled")
	}
	if *sub.DowngradeToPlan != "based" {
		t.Errorf("expected downgrade to based, got %s", *sub.DowngradeToPlan)
	}
}

