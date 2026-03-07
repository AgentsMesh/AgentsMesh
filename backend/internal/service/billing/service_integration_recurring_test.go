package billing

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// TestIntegrationRecurringPaymentSuccess tests recurring payment handling
func TestIntegrationRecurringPaymentSuccess(t *testing.T) {
	service, _, db := setupIntegrationTestService(t)
	ctx := context.Background()
	c, _ := createTestGinContext()

	// 1. Create active subscription with Stripe IDs
	proPlan, _ := service.GetPlan(ctx, "pro")
	now := time.Now()
	stripeSubID := "sub_recurring"
	stripeCusID := "cus_recurring"

	sub := &billing.Subscription{
		OrganizationID:       1,
		PlanID:               proPlan.ID,
		Status:               billing.SubscriptionStatusActive,
		BillingCycle:         billing.BillingCycleMonthly,
		CurrentPeriodStart:   now.AddDate(0, -1, 0), // Started last month
		CurrentPeriodEnd:     now,                   // Ending now
		StripeSubscriptionID: &stripeSubID,
		StripeCustomerID:     &stripeCusID,
		AutoRenew:            true,
		SeatCount:            1,
	}
	db.Create(sub)

	originalPeriodEnd := sub.CurrentPeriodEnd

	// 2. Simulate invoice.paid webhook for recurring payment
	event := &payment.WebhookEvent{
		EventID:        "evt_recurring_001",
		EventType:      "invoice.paid",
		Provider:       "mock",
		SubscriptionID: stripeSubID,
		CustomerID:     stripeCusID,
		Amount:         proPlan.PricePerSeatMonthly,
		Currency:       "USD",
		Status:         billing.OrderStatusSucceeded,
	}

	// 3. Handle payment succeeded (recurring)
	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("failed to handle recurring payment: %v", err)
	}

	// 4. Verify subscription period was extended
	updatedSub, _ := service.GetSubscription(ctx, 1)
	if !updatedSub.CurrentPeriodStart.After(originalPeriodEnd.Add(-time.Hour)) {
		t.Error("expected period start to be updated")
	}
}

// TestIntegrationRecurringPaymentYearly tests recurring payment for yearly subscription
func TestIntegrationRecurringPaymentYearly(t *testing.T) {
	service, _, db := setupIntegrationTestService(t)
	ctx := context.Background()
	c, _ := createTestGinContext()

	// 1. Create yearly subscription
	proPlan, _ := service.GetPlan(ctx, "pro")
	now := time.Now()
	stripeSubID := "sub_yearly_recurring"
	stripeCusID := "cus_yearly_recurring"

	sub := &billing.Subscription{
		OrganizationID:       1,
		PlanID:               proPlan.ID,
		Status:               billing.SubscriptionStatusActive,
		BillingCycle:         billing.BillingCycleYearly, // Yearly
		CurrentPeriodStart:   now.AddDate(-1, 0, 0),      // Started last year
		CurrentPeriodEnd:     now,                        // Ending now
		StripeSubscriptionID: &stripeSubID,
		StripeCustomerID:     &stripeCusID,
		AutoRenew:            true,
		SeatCount:            1,
	}
	db.Create(sub)

	originalPeriodEnd := sub.CurrentPeriodEnd

	// 2. Simulate invoice.paid webhook
	event := &payment.WebhookEvent{
		EventID:        "evt_yearly_recurring",
		EventType:      "invoice.paid",
		Provider:       "mock",
		SubscriptionID: stripeSubID,
		CustomerID:     stripeCusID,
		Amount:         proPlan.PricePerSeatYearly,
		Currency:       "USD",
		Status:         billing.OrderStatusSucceeded,
	}

	// 3. Handle recurring payment
	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("failed to handle recurring payment: %v", err)
	}

	// 4. Verify period extended by 1 year
	updatedSub, _ := service.GetSubscription(ctx, 1)
	expectedEnd := originalPeriodEnd.AddDate(1, 0, 0)
	if updatedSub.CurrentPeriodEnd.Before(expectedEnd.Add(-time.Hour)) {
		t.Errorf("expected period end to be extended by 1 year, got %v", updatedSub.CurrentPeriodEnd)
	}
}

// TestIntegrationRecurringPaymentWithDowngrade tests recurring payment with pending downgrade
func TestIntegrationRecurringPaymentWithDowngrade(t *testing.T) {
	service, _, db := setupIntegrationTestService(t)
	ctx := context.Background()
	c, _ := createTestGinContext()

	// 1. Create pro subscription with pending downgrade to based
	proPlan, _ := service.GetPlan(ctx, "pro")
	basedPlan, _ := service.GetPlan(ctx, "based")
	now := time.Now()
	stripeSubID := "sub_downgrade"
	stripeCusID := "cus_downgrade"
	downgradePlan := "based"

	sub := &billing.Subscription{
		OrganizationID:       1,
		PlanID:               proPlan.ID,
		Status:               billing.SubscriptionStatusActive,
		BillingCycle:         billing.BillingCycleMonthly,
		CurrentPeriodStart:   now.AddDate(0, -1, 0),
		CurrentPeriodEnd:     now,
		StripeSubscriptionID: &stripeSubID,
		StripeCustomerID:     &stripeCusID,
		DowngradeToPlan:      &downgradePlan, // Pending downgrade
		AutoRenew:            true,
		SeatCount:            1,
	}
	db.Create(sub)

	// 2. Simulate invoice.paid webhook
	event := &payment.WebhookEvent{
		EventID:        "evt_with_downgrade",
		EventType:      "invoice.paid",
		Provider:       "mock",
		SubscriptionID: stripeSubID,
		CustomerID:     stripeCusID,
		Amount:         basedPlan.PricePerSeatMonthly, // Paying based price
		Currency:       "USD",
		Status:         billing.OrderStatusSucceeded,
	}

	// 3. Handle recurring payment
	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("failed to handle payment: %v", err)
	}

	// 4. Verify downgrade applied
	updatedSub, _ := service.GetSubscription(ctx, 1)
	if updatedSub.PlanID != basedPlan.ID {
		t.Errorf("expected plan to be downgraded to based, got plan ID %d", updatedSub.PlanID)
	}
	if updatedSub.DowngradeToPlan != nil {
		t.Error("expected DowngradeToPlan to be cleared")
	}
}

// TestIntegrationRecurringPaymentWithBillingCycleChange tests recurring payment with billing cycle change
func TestIntegrationRecurringPaymentWithBillingCycleChange(t *testing.T) {
	service, _, db := setupIntegrationTestService(t)
	ctx := context.Background()
	c, _ := createTestGinContext()

	// 1. Create monthly subscription with pending change to yearly
	proPlan, _ := service.GetPlan(ctx, "pro")
	now := time.Now()
	stripeSubID := "sub_cycle_change"
	stripeCusID := "cus_cycle_change"
	nextCycle := billing.BillingCycleYearly

	sub := &billing.Subscription{
		OrganizationID:       1,
		PlanID:               proPlan.ID,
		Status:               billing.SubscriptionStatusActive,
		BillingCycle:         billing.BillingCycleMonthly,
		CurrentPeriodStart:   now.AddDate(0, -1, 0),
		CurrentPeriodEnd:     now,
		StripeSubscriptionID: &stripeSubID,
		StripeCustomerID:     &stripeCusID,
		NextBillingCycle:     &nextCycle, // Pending cycle change
		AutoRenew:            true,
		SeatCount:            1,
	}
	db.Create(sub)

	// 2. Simulate invoice.paid webhook
	event := &payment.WebhookEvent{
		EventID:        "evt_cycle_change",
		EventType:      "invoice.paid",
		Provider:       "mock",
		SubscriptionID: stripeSubID,
		CustomerID:     stripeCusID,
		Amount:         proPlan.PricePerSeatYearly,
		Currency:       "USD",
		Status:         billing.OrderStatusSucceeded,
	}

	// 3. Handle recurring payment
	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("failed to handle payment: %v", err)
	}

	// 4. Verify billing cycle changed
	updatedSub, _ := service.GetSubscription(ctx, 1)
	if updatedSub.BillingCycle != billing.BillingCycleYearly {
		t.Errorf("expected billing cycle to be yearly, got %s", updatedSub.BillingCycle)
	}
	if updatedSub.NextBillingCycle != nil {
		t.Error("expected NextBillingCycle to be cleared")
	}
}

// TestIntegrationRecurringPaymentFailure tests recurring payment failure and subscription freeze
func TestIntegrationRecurringPaymentFailure(t *testing.T) {
	service, _, db := setupIntegrationTestService(t)
	ctx := context.Background()
	c, _ := createTestGinContext()

	// 1. Create active subscription
	proPlan, _ := service.GetPlan(ctx, "pro")
	now := time.Now()
	stripeSubID := "sub_fail_recurring"
	stripeCusID := "cus_fail_recurring"

	sub := &billing.Subscription{
		OrganizationID:       1,
		PlanID:               proPlan.ID,
		Status:               billing.SubscriptionStatusActive,
		BillingCycle:         billing.BillingCycleMonthly,
		CurrentPeriodStart:   now.AddDate(0, -1, 0),
		CurrentPeriodEnd:     now.Add(-time.Hour),
		StripeSubscriptionID: &stripeSubID,
		StripeCustomerID:     &stripeCusID,
		AutoRenew:            true,
		SeatCount:            1,
	}
	db.Create(sub)

	// 2. Simulate invoice.payment_failed webhook
	event := &payment.WebhookEvent{
		EventID:        "evt_fail_recurring",
		EventType:      "invoice.payment_failed",
		Provider:       "mock",
		SubscriptionID: stripeSubID,
		CustomerID:     stripeCusID,
		Amount:         proPlan.PricePerSeatMonthly,
		Currency:       "USD",
		Status:         billing.OrderStatusFailed,
		FailedReason:   "Card declined",
	}

	// 3. Handle payment failure
	err := service.HandlePaymentFailed(c, event)
	if err != nil {
		t.Fatalf("failed to handle payment failure: %v", err)
	}

	// 4. Verify subscription is frozen
	updatedSub, _ := service.GetSubscription(ctx, 1)
	if updatedSub.Status != billing.SubscriptionStatusFrozen {
		t.Errorf("expected status frozen, got %s", updatedSub.Status)
	}
	if updatedSub.FrozenAt == nil {
		t.Error("expected FrozenAt to be set")
	}
}

// TestIntegrationRecurringPaymentSuccessNoSubscription tests success when subscription not found
func TestIntegrationRecurringPaymentSuccessNoSubscription(t *testing.T) {
	service, _, _ := setupIntegrationTestService(t)
	c, _ := createTestGinContext()

	// Subscription ID that doesn't exist (recurring payment with no matching subscription)
	event := &payment.WebhookEvent{
		EventID:        "evt_no_sub_success",
		EventType:      "invoice.paid",
		Provider:       "mock",
		SubscriptionID: "sub_nonexistent",
		CustomerID:     "cus_nonexistent",
		Amount:         19.99,
		Currency:       "USD",
		Status:         billing.OrderStatusSucceeded,
	}

	// Should not error - just ignores missing subscription
	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// TestIntegrationRecurringPaymentFailureNoSubscription tests failure when subscription not found
func TestIntegrationRecurringPaymentFailureNoSubscription(t *testing.T) {
	service, _, _ := setupIntegrationTestService(t)
	c, _ := createTestGinContext()

	// Subscription ID that doesn't exist
	event := &payment.WebhookEvent{
		EventID:        "evt_no_sub_failure",
		EventType:      "invoice.payment_failed",
		Provider:       "mock",
		SubscriptionID: "sub_nonexistent",
		CustomerID:     "cus_nonexistent",
		Amount:         19.99,
		Currency:       "USD",
		Status:         billing.OrderStatusFailed,
		FailedReason:   "Card declined",
	}

	// Should not error - just ignores missing subscription
	err := service.HandlePaymentFailed(c, event)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
