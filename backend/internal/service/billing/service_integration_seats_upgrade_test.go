package billing

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// TestIntegrationAddSeatsFlow tests the seat addition flow
func TestIntegrationAddSeatsFlow(t *testing.T) {
	service, _, db := setupIntegrationTestService(t)
	ctx := context.Background()
	c, _ := createTestGinContext()

	// 1. Create pro subscription
	proPlan, _ := service.GetPlan(ctx, "pro")
	now := time.Now()
	stripeSubID := "sub_seats"
	stripeCusID := "cus_seats"

	sub := &billing.Subscription{
		OrganizationID:       1,
		PlanID:               proPlan.ID,
		Status:               billing.SubscriptionStatusActive,
		BillingCycle:         billing.BillingCycleMonthly,
		CurrentPeriodStart:   now,
		CurrentPeriodEnd:     now.AddDate(0, 1, 0),
		StripeSubscriptionID: &stripeSubID,
		StripeCustomerID:     &stripeCusID,
		SeatCount:            1,
	}
	db.Create(sub)

	// 2. Create order for adding seats
	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-SEATS-001",
		OrderType:       billing.OrderTypeSeatPurchase,
		PlanID:          &proPlan.ID,
		Seats:           3, // Add 3 seats
		Amount:          proPlan.PricePerSeatMonthly * 3,
		ActualAmount:    proPlan.PricePerSeatMonthly * 3,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	// 3. Simulate payment succeeded
	event := &payment.WebhookEvent{
		EventID:         "evt_seats_001",
		EventType:       "checkout.session.completed",
		Provider:        "mock",
		OrderNo:         "ORD-SEATS-001",
		ExternalOrderNo: "mock_cs_seats",
		CustomerID:      stripeCusID,
		SubscriptionID:  stripeSubID,
		Amount:          proPlan.PricePerSeatMonthly * 3,
		Currency:        "USD",
		Status:          billing.OrderStatusSucceeded,
	}

	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("failed to handle add seats payment: %v", err)
	}

	// 4. Verify seats were added
	updatedSub, _ := service.GetSubscription(ctx, 1)
	if updatedSub.SeatCount != 4 { // 1 + 3 = 4
		t.Errorf("expected 4 seats, got %d", updatedSub.SeatCount)
	}
}

// TestIntegrationPlanUpgradeFlow tests plan upgrade via payment
func TestIntegrationPlanUpgradeFlow(t *testing.T) {
	service, _, db := setupIntegrationTestService(t)
	ctx := context.Background()
	c, _ := createTestGinContext()

	// 1. Create pro subscription
	proPlan, _ := service.GetPlan(ctx, "pro")
	entPlan, _ := service.GetPlan(ctx, "enterprise")
	now := time.Now()
	stripeSubID := "sub_upgrade"
	stripeCusID := "cus_upgrade"

	sub := &billing.Subscription{
		OrganizationID:       1,
		PlanID:               proPlan.ID,
		Status:               billing.SubscriptionStatusActive,
		BillingCycle:         billing.BillingCycleMonthly,
		CurrentPeriodStart:   now,
		CurrentPeriodEnd:     now.AddDate(0, 1, 0),
		StripeSubscriptionID: &stripeSubID,
		StripeCustomerID:     &stripeCusID,
		SeatCount:            1,
	}
	db.Create(sub)

	// 2. Create upgrade order
	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-UPGRADE-001",
		OrderType:       billing.OrderTypePlanUpgrade,
		PlanID:          &entPlan.ID,
		BillingCycle:    billing.BillingCycleMonthly,
		Seats:           1,
		Amount:          entPlan.PricePerSeatMonthly - proPlan.PricePerSeatMonthly, // Prorated
		ActualAmount:    entPlan.PricePerSeatMonthly - proPlan.PricePerSeatMonthly,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	// 3. Simulate payment succeeded
	event := &payment.WebhookEvent{
		EventID:         "evt_upgrade_001",
		EventType:       "checkout.session.completed",
		Provider:        "mock",
		OrderNo:         "ORD-UPGRADE-001",
		ExternalOrderNo: "mock_cs_upgrade",
		CustomerID:      stripeCusID,
		SubscriptionID:  stripeSubID,
		Amount:          entPlan.PricePerSeatMonthly - proPlan.PricePerSeatMonthly,
		Currency:        "USD",
		Status:          billing.OrderStatusSucceeded,
	}

	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("failed to handle upgrade payment: %v", err)
	}

	// 4. Verify plan was upgraded
	updatedSub, _ := service.GetSubscription(ctx, 1)
	if updatedSub.PlanID != entPlan.ID {
		t.Errorf("expected plan ID %d, got %d", entPlan.ID, updatedSub.PlanID)
	}
}

// TestIntegrationUpgradePlanWithNilPlanID tests upgrade with nil plan ID
func TestIntegrationUpgradePlanWithNilPlanID(t *testing.T) {
	service, _, _ := setupIntegrationTestService(t)
	ctx := context.Background()
	c, _ := createTestGinContext()

	service.CreateSubscription(ctx, 1, "based")

	// Create invalid order with nil plan ID
	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-INVALID-UPGRADE",
		OrderType:       billing.OrderTypePlanUpgrade,
		PlanID:          nil, // Invalid
		BillingCycle:    billing.BillingCycleMonthly,
		Seats:           1,
		Amount:          19.99,
		ActualAmount:    19.99,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	event := &payment.WebhookEvent{
		EventID:         "evt_invalid_upgrade",
		EventType:       "checkout.session.completed",
		Provider:        "mock",
		OrderNo:         "ORD-INVALID-UPGRADE",
		ExternalOrderNo: "mock_cs_invalid",
		Amount:          19.99,
		Currency:        "USD",
		Status:          billing.OrderStatusSucceeded,
	}

	// Should return error for invalid plan
	err := service.HandlePaymentSucceeded(c, event)
	if err != ErrInvalidPlan {
		t.Errorf("expected ErrInvalidPlan, got %v", err)
	}
}
