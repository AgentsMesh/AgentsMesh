package billing

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// TestIntegrationRenewalFlow tests subscription renewal flow
func TestIntegrationRenewalFlow(t *testing.T) {
	service, _, db := setupIntegrationTestService(t)
	ctx := context.Background()
	c, _ := createTestGinContext()

	// 1. Create subscription that needs renewal
	proPlan, _ := service.GetPlan(ctx, "pro")
	now := time.Now()

	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             proPlan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now.AddDate(0, -1, 0),
		CurrentPeriodEnd:   now.Add(-time.Hour), // Expired
		SeatCount:          2,
	}
	db.Create(sub)

	originalPeriodEnd := sub.CurrentPeriodEnd

	// 2. Create renewal order
	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-RENEWAL-001",
		OrderType:       billing.OrderTypeRenewal,
		PlanID:          &proPlan.ID,
		BillingCycle:    billing.BillingCycleMonthly,
		Seats:           2,
		Amount:          proPlan.PricePerSeatMonthly * 2,
		ActualAmount:    proPlan.PricePerSeatMonthly * 2,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	// 3. Simulate payment succeeded
	event := &payment.WebhookEvent{
		EventID:         "evt_renewal_001",
		EventType:       "checkout.session.completed",
		Provider:        "mock",
		OrderNo:         "ORD-RENEWAL-001",
		ExternalOrderNo: "mock_cs_renewal",
		Amount:          proPlan.PricePerSeatMonthly * 2,
		Currency:        "USD",
		Status:          billing.OrderStatusSucceeded,
	}

	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("failed to handle renewal payment: %v", err)
	}

	// 4. Verify subscription was renewed
	updatedSub, _ := service.GetSubscription(ctx, 1)
	if !updatedSub.CurrentPeriodEnd.After(originalPeriodEnd) {
		t.Error("expected period end to be extended")
	}
	if updatedSub.Status != billing.SubscriptionStatusActive {
		t.Errorf("expected active status, got %s", updatedSub.Status)
	}
}

// TestIntegrationRenewalFlowYearly tests yearly subscription renewal
func TestIntegrationRenewalFlowYearly(t *testing.T) {
	service, _, db := setupIntegrationTestService(t)
	ctx := context.Background()
	c, _ := createTestGinContext()

	// 1. Create yearly subscription that needs renewal
	proPlan, _ := service.GetPlan(ctx, "pro")
	now := time.Now()

	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             proPlan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleYearly, // Yearly
		CurrentPeriodStart: now.AddDate(-1, 0, 0),
		CurrentPeriodEnd:   now.Add(-time.Hour), // Expired
		SeatCount:          2,
	}
	db.Create(sub)

	originalPeriodEnd := sub.CurrentPeriodEnd

	// 2. Create renewal order
	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-YEARLY-RENEWAL",
		OrderType:       billing.OrderTypeRenewal,
		PlanID:          &proPlan.ID,
		BillingCycle:    billing.BillingCycleYearly,
		Seats:           2,
		Amount:          proPlan.PricePerSeatYearly * 2,
		ActualAmount:    proPlan.PricePerSeatYearly * 2,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	// 3. Simulate payment succeeded
	event := &payment.WebhookEvent{
		EventID:         "evt_yearly_renewal",
		EventType:       "checkout.session.completed",
		Provider:        "mock",
		OrderNo:         "ORD-YEARLY-RENEWAL",
		ExternalOrderNo: "mock_cs_yearly_renewal",
		Amount:          proPlan.PricePerSeatYearly * 2,
		Currency:        "USD",
		Status:          billing.OrderStatusSucceeded,
	}

	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("failed to handle renewal payment: %v", err)
	}

	// 4. Verify subscription renewed for 1 year
	updatedSub, _ := service.GetSubscription(ctx, 1)
	expectedEnd := originalPeriodEnd.AddDate(1, 0, 0)
	if updatedSub.CurrentPeriodEnd.Before(expectedEnd.Add(-time.Hour)) {
		t.Error("expected period end to be extended by 1 year")
	}
}
