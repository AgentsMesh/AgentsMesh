package billing

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// TestIntegrationUpgradeWithCheckout tests the upgrade flow with checkout
func TestIntegrationUpgradeWithCheckout(t *testing.T) {
	service, factory, _ := setupIntegrationTestService(t)
	ctx := context.Background()

	// 1. Create based subscription
	service.CreateSubscription(ctx, 1, "based")

	// 2. Get pro plan for upgrade
	proPlan, _ := service.GetPlan(ctx, "pro")

	// 3. Create checkout session for upgrade
	provider, _ := factory.GetDefaultProvider()
	checkoutReq := &payment.CheckoutRequest{
		OrganizationID: 1,
		OrderType:      billing.OrderTypeSubscription,
		PlanID:         proPlan.ID,
		BillingCycle:   billing.BillingCycleMonthly,
		Seats:          1,
		Currency:       "USD",
		Amount:         proPlan.PricePerSeatMonthly,
		ActualAmount:   proPlan.PricePerSeatMonthly,
		SuccessURL:     "http://localhost:3000/success",
		CancelURL:      "http://localhost:3000/cancel",
		IdempotencyKey: "ORD-TEST-001",
	}

	resp, err := provider.CreateCheckoutSession(ctx, checkoutReq)
	if err != nil {
		t.Fatalf("failed to create checkout session: %v", err)
	}
	if resp.SessionID == "" {
		t.Error("expected session ID")
	}
	if resp.SessionURL == "" {
		t.Error("expected session URL")
	}

	// 4. Verify session status is pending
	status, err := provider.GetCheckoutStatus(ctx, resp.SessionID)
	if err != nil {
		t.Fatalf("failed to get checkout status: %v", err)
	}
	if status != billing.OrderStatusPending {
		t.Errorf("expected pending status, got %s", status)
	}

	// 5. Complete the checkout (simulate user payment)
	mockProvider := factory.GetMockProvider()
	session, err := mockProvider.CompleteSession(resp.SessionID)
	if err != nil {
		t.Fatalf("failed to complete session: %v", err)
	}
	if session.Status != billing.OrderStatusSucceeded {
		t.Errorf("expected succeeded status, got %s", session.Status)
	}

	// 6. Verify status is now succeeded
	status, _ = provider.GetCheckoutStatus(ctx, resp.SessionID)
	if status != billing.OrderStatusSucceeded {
		t.Errorf("expected succeeded status, got %s", status)
	}
}

// TestIntegrationMockProviderWebhook tests mock provider webhook handling
func TestIntegrationMockProviderWebhook(t *testing.T) {
	service, factory, _ := setupIntegrationTestService(t)
	ctx := context.Background()

	// 1. Create checkout session
	provider, _ := factory.GetDefaultProvider()
	proPlan, _ := service.GetPlan(ctx, "pro")

	checkoutReq := &payment.CheckoutRequest{
		OrganizationID: 1,
		OrderType:      billing.OrderTypeSubscription,
		PlanID:         proPlan.ID,
		BillingCycle:   billing.BillingCycleMonthly,
		Seats:          1,
		Currency:       "USD",
		Amount:         proPlan.PricePerSeatMonthly,
		ActualAmount:   proPlan.PricePerSeatMonthly,
		SuccessURL:     "http://localhost:3000/success",
		CancelURL:      "http://localhost:3000/cancel",
		IdempotencyKey: "ORD-MOCK-001",
	}

	resp, err := provider.CreateCheckoutSession(ctx, checkoutReq)
	if err != nil {
		t.Fatalf("failed to create checkout: %v", err)
	}

	// 2. Complete the session
	mockProvider := factory.GetMockProvider()
	_, err = mockProvider.CompleteSession(resp.SessionID)
	if err != nil {
		t.Fatalf("failed to complete session: %v", err)
	}

	// 3. Simulate webhook
	webhookPayload := []byte(`{"event_type": "checkout.session.completed", "session_id": "` + resp.SessionID + `", "order_no": "ORD-MOCK-001"}`)
	event, err := provider.HandleWebhook(ctx, webhookPayload, "")
	if err != nil {
		t.Fatalf("failed to handle webhook: %v", err)
	}

	if event.Status != billing.OrderStatusSucceeded {
		t.Errorf("expected succeeded status, got %s", event.Status)
	}
	if event.EventType != "checkout.session.completed" {
		t.Errorf("expected checkout.session.completed, got %s", event.EventType)
	}
}

// TestIntegrationRefundPayment tests payment refund
func TestIntegrationRefundPayment(t *testing.T) {
	_, factory, _ := setupIntegrationTestService(t)
	ctx := context.Background()

	provider, _ := factory.GetDefaultProvider()

	refundReq := &payment.RefundRequest{
		OrderNo: "ORD-REFUND-001",
		Amount:  19.99,
		Reason:  "Customer request",
	}

	resp, err := provider.RefundPayment(ctx, refundReq)
	if err != nil {
		t.Fatalf("failed to refund: %v", err)
	}

	if resp.Status != "succeeded" {
		t.Errorf("expected succeeded, got %s", resp.Status)
	}
	if resp.Amount != 19.99 {
		t.Errorf("expected amount 19.99, got %f", resp.Amount)
	}
}

// TestIntegrationCancelSubscription tests subscription cancellation via provider
func TestIntegrationCancelSubscription(t *testing.T) {
	_, factory, _ := setupIntegrationTestService(t)
	ctx := context.Background()

	provider, _ := factory.GetDefaultProvider()

	// Mock provider always succeeds
	err := provider.CancelSubscription(ctx, "sub_test_123", false)
	if err != nil {
		t.Fatalf("failed to cancel subscription: %v", err)
	}
}
