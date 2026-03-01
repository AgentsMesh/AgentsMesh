package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
)

// ===========================================
// Push 95% Coverage - Payment Tests
// ===========================================

// TestHandlePaymentSucceeded_FullFlow tests transaction creation
func TestHandlePaymentSucceeded_FullFlow(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	planID := int64(2)
	expiresAt := now.Add(time.Hour)

	db.Create(&billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-FULL-FLOW",
		OrderType:       billing.OrderTypeSubscription,
		PlanID:          &planID,
		Seats:           2,
		BillingCycle:    billing.BillingCycleMonthly,
		Amount:          39.98,
		Currency:        "USD",
		Status:          billing.OrderStatusPending,
		PaymentProvider: billing.PaymentProviderStripe,
		ExpiresAt:       &expiresAt,
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:         "evt_full_flow",
		EventType:       "checkout.session.completed",
		Provider:        billing.PaymentProviderStripe,
		OrderNo:         "ORD-FULL-FLOW",
		ExternalOrderNo: "cs_ext_123",
		Amount:          39.98,
		Currency:        "USD",
		RawPayload:      map[string]interface{}{"test": true},
	}

	err := svc.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify transaction was created
	var tx billing.PaymentTransaction
	db.Where("webhook_event_id = ?", "evt_full_flow").First(&tx)
	if tx.ID == 0 {
		t.Error("expected transaction to be created")
	}
}

// TestCreateStripeCustomer_Disabled tests when stripe is disabled
func TestCreateStripeCustomer_Disabled(t *testing.T) {
	svc, _ := setupTestService(t)

	// Service created without Stripe enabled
	customerID, err := svc.CreateStripeCustomer(context.Background(), 1, "test@example.com", "Test User")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if customerID != "" {
		t.Errorf("expected empty customer ID when Stripe disabled, got %s", customerID)
	}
}

// TestHandleSubscriptionCreated_SetCustomerID tests setting customer ID when nil
func TestHandleSubscriptionCreated_SetCustomerID(t *testing.T) {
	svc, db := setupTestService(t)

	lsCustID := "ls_cust_set"
	now := time.Now()
	// Subscription with customer ID but no subscription ID
	db.Create(&billing.Subscription{
		OrganizationID:         1,
		PlanID:                 2,
		Status:                 billing.SubscriptionStatusActive,
		BillingCycle:           billing.BillingCycleMonthly,
		CurrentPeriodStart:     now,
		CurrentPeriodEnd:       now.AddDate(0, 1, 0),
		SeatCount:              1,
		LemonSqueezyCustomerID: &lsCustID,
		// LemonSqueezySubscriptionID is nil
	})

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_set_cust_id",
		EventType:      billing.WebhookEventLSSubscriptionCreated,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "ls_sub_set_cust",
		CustomerID:     lsCustID,
	}

	err := svc.HandleSubscriptionCreated(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.LemonSqueezySubscriptionID == nil || *sub.LemonSqueezySubscriptionID != "ls_sub_set_cust" {
		t.Error("expected subscription ID to be set")
	}
}
