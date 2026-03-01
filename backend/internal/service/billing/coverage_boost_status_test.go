package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
)

// ===========================================
// Coverage Boost Tests - Status Mappings
// ===========================================

// TestHandleSubscriptionUpdated_VariousStatuses tests all status mappings
func TestHandleSubscriptionUpdated_VariousStatuses(t *testing.T) {
	tests := []struct {
		name           string
		status         string
		expectedStatus string
	}{
		{"active", "active", billing.SubscriptionStatusActive},
		{"past_due", "past_due", billing.SubscriptionStatusPastDue},
		{"canceled", "canceled", billing.SubscriptionStatusCanceled},
		{"cancelled_uk", "cancelled", billing.SubscriptionStatusCanceled},
		{"trialing", "trialing", billing.SubscriptionStatusTrialing},
		{"paused", "paused", billing.SubscriptionStatusPaused},
		{"expired", "expired", billing.SubscriptionStatusExpired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, db := setupTestService(t)

			stripeSubID := "sub_stripe_status_" + tt.name
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
				FrozenAt:             &now, // Set frozen to test clearing
			})

			c, _ := createTestGinContext()
			event := &payment.WebhookEvent{
				EventID:        "evt_status_" + tt.name,
				EventType:      "customer.subscription.updated",
				Provider:       billing.PaymentProviderStripe,
				SubscriptionID: stripeSubID,
				Status:         tt.status,
			}

			err := svc.HandleSubscriptionUpdated(c, event)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			sub, _ := svc.GetSubscription(context.Background(), 1)
			if sub.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, sub.Status)
			}

			// For active status, FrozenAt should be cleared
			if tt.status == "active" && sub.FrozenAt != nil {
				t.Error("expected FrozenAt to be cleared for active status")
			}
		})
	}
}

// TestHandleSubscriptionUpdated_LemonSqueezyStatus tests LemonSqueezy status mapping
func TestHandleSubscriptionUpdated_LemonSqueezyStatus(t *testing.T) {
	svc, db := setupTestService(t)

	lsSubID := "ls_sub_updated"
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
		EventID:        "evt_ls_updated",
		EventType:      billing.WebhookEventLSSubscriptionUpdated,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: lsSubID,
		Status:         "active", // LemonSqueezy status
	}

	err := svc.HandleSubscriptionUpdated(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.Status != billing.SubscriptionStatusActive {
		t.Errorf("expected active status, got %s", sub.Status)
	}
}

// TestHandleSubscriptionUpdated_EmptySubID tests with empty subscription ID
func TestHandleSubscriptionUpdated_EmptySubID(t *testing.T) {
	svc, _ := setupTestService(t)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_empty_sub_updated",
		EventType:      "customer.subscription.updated",
		Provider:       billing.PaymentProviderStripe,
		SubscriptionID: "",
	}

	err := svc.HandleSubscriptionUpdated(c, event)
	if err != nil {
		t.Errorf("expected nil for empty subscription ID, got %v", err)
	}
}

// TestHandleSubscriptionUpdated_NotFound tests with non-existent subscription
func TestHandleSubscriptionUpdated_NotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_not_found_updated",
		EventType:      "customer.subscription.updated",
		Provider:       billing.PaymentProviderStripe,
		SubscriptionID: "nonexistent",
	}

	err := svc.HandleSubscriptionUpdated(c, event)
	if err != nil {
		t.Errorf("expected nil for non-existent subscription, got %v", err)
	}
}

// TestHandleSubscriptionCanceled_NotFound_Stripe tests cancellation with non-existent subscription via Stripe
func TestHandleSubscriptionCanceled_NotFound_Stripe(t *testing.T) {
	svc, _ := setupTestService(t)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_cancel_not_found_stripe",
		EventType:      "customer.subscription.deleted",
		Provider:       billing.PaymentProviderStripe,
		SubscriptionID: "nonexistent_stripe",
	}

	err := svc.HandleSubscriptionCanceled(c, event)
	if err != nil {
		t.Errorf("expected nil for non-existent subscription, got %v", err)
	}
}
