package billing

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// TestIntegrationWebhookPaymentSucceeded tests webhook handling for successful payment
func TestIntegrationWebhookPaymentSucceeded(t *testing.T) {
	service, factory, _ := setupIntegrationTestService(t)
	ctx := context.Background()
	c, _ := createTestGinContext()

	// 1. Create subscription
	service.CreateSubscription(ctx, 1, "based")

	// 2. Create a payment order
	proPlan, _ := service.GetPlan(ctx, "pro")
	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-WEBHOOK-001",
		OrderType:       billing.OrderTypeSubscription,
		PlanID:          &proPlan.ID,
		BillingCycle:    billing.BillingCycleMonthly,
		Seats:           1,
		Amount:          proPlan.PricePerSeatMonthly,
		ActualAmount:    proPlan.PricePerSeatMonthly,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	// 3. Simulate webhook event
	event := &payment.WebhookEvent{
		EventID:         "evt_test_001",
		EventType:       "checkout.session.completed",
		Provider:        "mock",
		OrderNo:         "ORD-WEBHOOK-001",
		ExternalOrderNo: "mock_cs_123",
		CustomerID:      "mock_cus_456",
		SubscriptionID:  "mock_sub_789",
		Amount:          proPlan.PricePerSeatMonthly,
		Currency:        "USD",
		Status:          billing.OrderStatusSucceeded,
	}

	// 4. Handle payment succeeded
	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("failed to handle payment succeeded: %v", err)
	}

	// 5. Verify order status updated
	updatedOrder, _ := service.GetPaymentOrderByNo(ctx, "ORD-WEBHOOK-001")
	if updatedOrder.Status != billing.OrderStatusSucceeded {
		t.Errorf("expected order status succeeded, got %s", updatedOrder.Status)
	}
	if updatedOrder.PaidAt == nil {
		t.Error("expected PaidAt to be set")
	}

	// 6. Verify subscription upgraded
	sub, _ := service.GetSubscription(ctx, 1)
	if sub.StripeCustomerID == nil || *sub.StripeCustomerID != "mock_cus_456" {
		t.Error("expected customer ID to be set")
	}
	if sub.StripeSubscriptionID == nil || *sub.StripeSubscriptionID != "mock_sub_789" {
		t.Error("expected subscription ID to be set")
	}

	// Verify factory mock status
	if !factory.IsMockEnabled() {
		t.Error("mock should be enabled")
	}
}

// TestIntegrationWebhookPaymentFailed tests webhook handling for failed payment
func TestIntegrationWebhookPaymentFailed(t *testing.T) {
	service, _, _ := setupIntegrationTestService(t)
	ctx := context.Background()
	c, _ := createTestGinContext()

	// 1. Create subscription
	service.CreateSubscription(ctx, 1, "based")

	// 2. Create a payment order
	proPlan, _ := service.GetPlan(ctx, "pro")
	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-FAILED-001",
		OrderType:       billing.OrderTypeSubscription,
		PlanID:          &proPlan.ID,
		BillingCycle:    billing.BillingCycleMonthly,
		Seats:           1,
		Amount:          proPlan.PricePerSeatMonthly,
		ActualAmount:    proPlan.PricePerSeatMonthly,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	// 3. Simulate failed payment webhook
	event := &payment.WebhookEvent{
		EventID:      "evt_failed_001",
		EventType:    "invoice.payment_failed",
		Provider:     "mock",
		OrderNo:      "ORD-FAILED-001",
		Amount:       proPlan.PricePerSeatMonthly,
		Currency:     "USD",
		Status:       billing.OrderStatusFailed,
		FailedReason: "Card declined",
	}

	// 4. Handle payment failed
	err := service.HandlePaymentFailed(c, event)
	if err != nil {
		t.Fatalf("failed to handle payment failed: %v", err)
	}

	// 5. Verify order status updated
	updatedOrder, _ := service.GetPaymentOrderByNo(ctx, "ORD-FAILED-001")
	if updatedOrder.Status != billing.OrderStatusFailed {
		t.Errorf("expected order status failed, got %s", updatedOrder.Status)
	}
	if updatedOrder.FailureReason == nil || *updatedOrder.FailureReason != "Card declined" {
		t.Error("expected failure reason to be set")
	}
}

// TestIntegrationPaymentSucceededByExternalOrderNo tests finding order by external order no
func TestIntegrationPaymentSucceededByExternalOrderNo(t *testing.T) {
	service, _, _ := setupIntegrationTestService(t)
	ctx := context.Background()
	c, _ := createTestGinContext()

	service.CreateSubscription(ctx, 1, "based")

	proPlan, _ := service.GetPlan(ctx, "pro")
	externalNo := "mock_cs_external_123"

	// Create order with external order no but no internal order no in event
	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-INTERNAL-001",
		ExternalOrderNo: &externalNo,
		OrderType:       billing.OrderTypeSubscription,
		PlanID:          &proPlan.ID,
		BillingCycle:    billing.BillingCycleMonthly,
		Seats:           1,
		Amount:          proPlan.PricePerSeatMonthly,
		ActualAmount:    proPlan.PricePerSeatMonthly,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	// Event only has external order no
	event := &payment.WebhookEvent{
		EventID:         "evt_external",
		EventType:       "checkout.session.completed",
		Provider:        "mock",
		ExternalOrderNo: externalNo,
		Amount:          proPlan.PricePerSeatMonthly,
		Currency:        "USD",
		Status:          billing.OrderStatusSucceeded,
	}

	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("failed to handle payment: %v", err)
	}

	// Verify order found and updated
	updatedOrder, _ := service.GetPaymentOrderByNo(ctx, "ORD-INTERNAL-001")
	if updatedOrder.Status != billing.OrderStatusSucceeded {
		t.Errorf("expected succeeded status, got %s", updatedOrder.Status)
	}
}

// TestIntegrationPaymentFailedByExternalOrderNo tests finding failed order by external order no
func TestIntegrationPaymentFailedByExternalOrderNo(t *testing.T) {
	service, _, _ := setupIntegrationTestService(t)
	ctx := context.Background()
	c, _ := createTestGinContext()

	service.CreateSubscription(ctx, 1, "based")

	proPlan, _ := service.GetPlan(ctx, "pro")
	externalNo := "mock_cs_external_fail"

	// Create order
	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-FAIL-EXTERNAL",
		ExternalOrderNo: &externalNo,
		OrderType:       billing.OrderTypeSubscription,
		PlanID:          &proPlan.ID,
		BillingCycle:    billing.BillingCycleMonthly,
		Seats:           1,
		Amount:          proPlan.PricePerSeatMonthly,
		ActualAmount:    proPlan.PricePerSeatMonthly,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	// Event only has external order no
	event := &payment.WebhookEvent{
		EventID:         "evt_fail_external",
		EventType:       "payment_intent.payment_failed",
		Provider:        "mock",
		ExternalOrderNo: externalNo,
		Amount:          proPlan.PricePerSeatMonthly,
		Currency:        "USD",
		Status:          billing.OrderStatusFailed,
		FailedReason:    "Card declined",
	}

	err := service.HandlePaymentFailed(c, event)
	if err != nil {
		t.Fatalf("failed to handle payment failed: %v", err)
	}

	// Verify order found and updated
	updatedOrder, _ := service.GetPaymentOrderByNo(ctx, "ORD-FAIL-EXTERNAL")
	if updatedOrder.Status != billing.OrderStatusFailed {
		t.Errorf("expected failed status, got %s", updatedOrder.Status)
	}
}
