package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
)

// ===========================================
// Subscription Cancellation Tests
// ===========================================

// TestAdditionalHandleSubscriptionCanceled_Stripe tests cancellation with Stripe provider
func TestAdditionalHandleSubscriptionCanceled_Stripe(t *testing.T) {
	svc, db := setupTestService(t)

	stripeSubID := "sub_stripe_test"
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

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_stripe_cancel_test",
		EventType:      "customer.subscription.deleted",
		Provider:       billing.PaymentProviderStripe,
		SubscriptionID: stripeSubID,
	}

	err := svc.HandleSubscriptionCanceled(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.Status != billing.SubscriptionStatusCanceled {
		t.Errorf("expected canceled status, got %s", sub.Status)
	}
}

// TestAdditionalHandleSubscriptionCanceled_LemonSqueezy tests cancellation with LemonSqueezy
func TestAdditionalHandleSubscriptionCanceled_LemonSqueezy(t *testing.T) {
	svc, db := setupTestService(t)

	lsSubID := "ls_sub_cancel_test"
	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:             1,
		PlanID:                     2,
		Status:                     billing.SubscriptionStatusActive,
		BillingCycle:               billing.BillingCycleMonthly,
		CurrentPeriodStart:         now,
		CurrentPeriodEnd:           now.AddDate(0, 1, 0),
		SeatCount:                  1,
		LemonSqueezySubscriptionID: &lsSubID,
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_ls_cancel_test",
		EventType:      billing.WebhookEventLSSubscriptionCancelled,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: lsSubID,
	}

	err := svc.HandleSubscriptionCanceled(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.Status != billing.SubscriptionStatusCanceled {
		t.Errorf("expected canceled status, got %s", sub.Status)
	}
	if sub.CanceledAt == nil {
		t.Error("expected CanceledAt to be set")
	}
}

// ===========================================
// Billing Cycle Tests
// ===========================================

// TestAdditionalSetNextBillingCycle_ToMonthly tests setting next billing cycle
func TestAdditionalSetNextBillingCycle_ToMonthly(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleYearly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(1, 0, 0),
		SeatCount:          1,
	})

	err := svc.SetNextBillingCycle(context.Background(), 1, billing.BillingCycleMonthly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.NextBillingCycle == nil || *sub.NextBillingCycle != billing.BillingCycleMonthly {
		t.Error("expected NextBillingCycle to be set to monthly")
	}
}
