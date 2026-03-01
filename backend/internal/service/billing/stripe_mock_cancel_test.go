package billing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Tests Using Mock Stripe Client - Cancel
// ===========================================

// TestCancelSubscription_WithStripe tests cancellation with Stripe enabled
func TestCancelSubscription_WithStripe(t *testing.T) {
	svc, db, mockClient := setupTestServiceWithMockStripe(t)

	stripeSubID := "sub_test123"
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

	err := svc.CancelSubscription(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockClient.CancelSubscriptionCalls) != 1 {
		t.Errorf("expected 1 CancelSubscription call, got %d", len(mockClient.CancelSubscriptionCalls))
	}

	call := mockClient.CancelSubscriptionCalls[0]
	if call.ID != stripeSubID {
		t.Errorf("expected subscription ID '%s', got '%s'", stripeSubID, call.ID)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.Status != billing.SubscriptionStatusCanceled {
		t.Errorf("expected canceled status, got %s", sub.Status)
	}
}

// TestCancelSubscription_StripeError tests cancellation when Stripe API fails
func TestCancelSubscription_StripeError(t *testing.T) {
	svc, db, mockClient := setupTestServiceWithMockStripe(t)

	mockClient.CancelSubscriptionErr = errors.New("stripe cancellation failed")

	stripeSubID := "sub_test456"
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

	err := svc.CancelSubscription(context.Background(), 1)
	if err == nil {
		t.Error("expected error, got nil")
	}
	if err.Error() != "stripe cancellation failed" {
		t.Errorf("expected 'stripe cancellation failed', got %v", err)
	}

	if len(mockClient.CancelSubscriptionCalls) != 1 {
		t.Errorf("expected 1 CancelSubscription call, got %d", len(mockClient.CancelSubscriptionCalls))
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.Status != billing.SubscriptionStatusActive {
		t.Errorf("expected status to remain active, got %s", sub.Status)
	}
}

// TestCancelSubscription_NoStripeSubscription tests cancellation when no Stripe ID
func TestCancelSubscription_NoStripeSubscription(t *testing.T) {
	svc, db, mockClient := setupTestServiceWithMockStripe(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	err := svc.CancelSubscription(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mockClient.CancelSubscriptionCalls) != 0 {
		t.Errorf("expected 0 CancelSubscription calls, got %d", len(mockClient.CancelSubscriptionCalls))
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.Status != billing.SubscriptionStatusCanceled {
		t.Errorf("expected canceled status, got %s", sub.Status)
	}
}
