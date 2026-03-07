package billing

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// TestIntegrationWebhookSubscriptionCanceled tests subscription cancellation webhook
func TestIntegrationWebhookSubscriptionCanceled(t *testing.T) {
	service, _, db := setupIntegrationTestService(t)
	ctx := context.Background()
	c, _ := createTestGinContext()

	// 1. Create subscription with Stripe IDs
	proPlan, _ := service.GetPlan(ctx, "pro")
	now := time.Now()
	stripeSubID := "sub_test_cancel"
	stripeCusID := "cus_test_cancel"

	sub := &billing.Subscription{
		OrganizationID:       1,
		PlanID:               proPlan.ID,
		Status:               billing.SubscriptionStatusActive,
		BillingCycle:         billing.BillingCycleMonthly,
		CurrentPeriodStart:   now,
		CurrentPeriodEnd:     now.AddDate(0, 1, 0),
		StripeSubscriptionID: &stripeSubID,
		StripeCustomerID:     &stripeCusID,
	}
	db.Create(sub)

	// 2. Simulate subscription canceled webhook
	event := &payment.WebhookEvent{
		EventID:        "evt_cancel_001",
		EventType:      "customer.subscription.deleted",
		Provider:       "mock",
		SubscriptionID: stripeSubID,
		CustomerID:     stripeCusID,
		Status:         billing.SubscriptionStatusCanceled,
	}

	// 3. Handle subscription canceled
	err := service.HandleSubscriptionCanceled(c, event)
	if err != nil {
		t.Fatalf("failed to handle subscription canceled: %v", err)
	}

	// 4. Verify subscription status updated
	updatedSub, _ := service.GetSubscription(ctx, 1)
	if updatedSub.Status != billing.SubscriptionStatusCanceled {
		t.Errorf("expected subscription status canceled, got %s", updatedSub.Status)
	}
}

// TestIntegrationWebhookSubscriptionUpdated tests subscription status update webhook
func TestIntegrationWebhookSubscriptionUpdated(t *testing.T) {
	service, _, db := setupIntegrationTestService(t)
	ctx := context.Background()
	c, _ := createTestGinContext()

	// 1. Create subscription with Stripe IDs
	proPlan, _ := service.GetPlan(ctx, "pro")
	now := time.Now()
	stripeSubID := "sub_test_update"
	stripeCusID := "cus_test_update"

	sub := &billing.Subscription{
		OrganizationID:       1,
		PlanID:               proPlan.ID,
		Status:               billing.SubscriptionStatusActive,
		BillingCycle:         billing.BillingCycleMonthly,
		CurrentPeriodStart:   now,
		CurrentPeriodEnd:     now.AddDate(0, 1, 0),
		StripeSubscriptionID: &stripeSubID,
		StripeCustomerID:     &stripeCusID,
		AutoRenew:            true,
	}
	db.Create(sub)

	// 2. Simulate subscription status change to past_due
	event := &payment.WebhookEvent{
		EventID:        "evt_update_001",
		EventType:      "customer.subscription.updated",
		Provider:       "mock",
		SubscriptionID: stripeSubID,
		CustomerID:     stripeCusID,
		Status:         "past_due", // Changed status
	}

	// 3. Handle subscription updated
	err := service.HandleSubscriptionUpdated(c, event)
	if err != nil {
		t.Fatalf("failed to handle subscription updated: %v", err)
	}

	// 4. Verify subscription status updated
	updatedSub, _ := service.GetSubscription(ctx, 1)
	if updatedSub.Status != billing.SubscriptionStatusPastDue {
		t.Errorf("expected status past_due, got %s", updatedSub.Status)
	}
}

// TestIntegrationSubscriptionCanceledNoSubscription tests cancellation when subscription not found
func TestIntegrationSubscriptionCanceledNoSubscription(t *testing.T) {
	service, _, _ := setupIntegrationTestService(t)
	c, _ := createTestGinContext()

	event := &payment.WebhookEvent{
		EventID:        "evt_cancel_no_sub",
		EventType:      "customer.subscription.deleted",
		Provider:       "mock",
		SubscriptionID: "sub_nonexistent",
		CustomerID:     "cus_nonexistent",
		Status:         billing.SubscriptionStatusCanceled,
	}

	// Should not error - just ignores missing subscription
	err := service.HandleSubscriptionCanceled(c, event)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// TestIntegrationSubscriptionCanceledEmptySubscriptionID tests cancellation with empty ID
func TestIntegrationSubscriptionCanceledEmptySubscriptionID(t *testing.T) {
	service, _, _ := setupIntegrationTestService(t)
	c, _ := createTestGinContext()

	event := &payment.WebhookEvent{
		EventID:        "evt_cancel_empty",
		EventType:      "customer.subscription.deleted",
		Provider:       "mock",
		SubscriptionID: "", // Empty
		CustomerID:     "cus_any",
		Status:         billing.SubscriptionStatusCanceled,
	}

	// Should not error - exits early
	err := service.HandleSubscriptionCanceled(c, event)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// TestIntegrationSubscriptionUpdatedNoSubscription tests update when subscription not found
func TestIntegrationSubscriptionUpdatedNoSubscription(t *testing.T) {
	service, _, _ := setupIntegrationTestService(t)
	c, _ := createTestGinContext()

	event := &payment.WebhookEvent{
		EventID:        "evt_update_no_sub",
		EventType:      "customer.subscription.updated",
		Provider:       "mock",
		SubscriptionID: "sub_nonexistent",
		CustomerID:     "cus_nonexistent",
		Status:         "active",
	}

	// Should not error - just ignores missing subscription
	err := service.HandleSubscriptionUpdated(c, event)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// TestIntegrationSubscriptionUpdatedEmptySubscriptionID tests update with empty ID
func TestIntegrationSubscriptionUpdatedEmptySubscriptionID(t *testing.T) {
	service, _, _ := setupIntegrationTestService(t)
	c, _ := createTestGinContext()

	event := &payment.WebhookEvent{
		EventID:        "evt_update_empty",
		EventType:      "customer.subscription.updated",
		Provider:       "mock",
		SubscriptionID: "", // Empty
		CustomerID:     "cus_any",
		Status:         "active",
	}

	// Should not error - exits early
	err := service.HandleSubscriptionUpdated(c, event)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// TestIntegrationSubscriptionUpdatedVariousStatuses tests different status updates
func TestIntegrationSubscriptionUpdatedVariousStatuses(t *testing.T) {
	service, _, db := setupIntegrationTestService(t)
	ctx := context.Background()
	c, _ := createTestGinContext()

	proPlan, _ := service.GetPlan(ctx, "pro")

	statuses := []struct {
		stripeStatus   string
		expectedStatus string
	}{
		{"active", billing.SubscriptionStatusActive},
		{"trialing", billing.SubscriptionStatusTrialing},
		{"canceled", billing.SubscriptionStatusCanceled},
	}

	for i, tc := range statuses {
		stripeSubID := "sub_status_test_" + tc.stripeStatus
		stripeCusID := "cus_status_test"
		now := time.Now()

		// Create subscription
		sub := &billing.Subscription{
			OrganizationID:       int64(100 + i),
			PlanID:               proPlan.ID,
			Status:               billing.SubscriptionStatusActive,
			BillingCycle:         billing.BillingCycleMonthly,
			CurrentPeriodStart:   now,
			CurrentPeriodEnd:     now.AddDate(0, 1, 0),
			StripeSubscriptionID: &stripeSubID,
			StripeCustomerID:     &stripeCusID,
		}
		db.Create(sub)

		// Send update event
		event := &payment.WebhookEvent{
			EventID:        "evt_status_" + tc.stripeStatus,
			EventType:      "customer.subscription.updated",
			Provider:       "mock",
			SubscriptionID: stripeSubID,
			CustomerID:     stripeCusID,
			Status:         tc.stripeStatus,
		}

		err := service.HandleSubscriptionUpdated(c, event)
		if err != nil {
			t.Errorf("failed to update status to %s: %v", tc.stripeStatus, err)
		}

		updatedSub, _ := service.GetSubscription(ctx, int64(100+i))
		if updatedSub.Status != tc.expectedStatus {
			t.Errorf("expected status %s, got %s", tc.expectedStatus, updatedSub.Status)
		}
	}
}
