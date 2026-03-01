package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
)

// ===========================================
// Subscription Tests from Coverage 95
// ===========================================

// TestUpdateSubscription_UpgradeStripeEnabled_95 tests update with Stripe enabled (code path)
func TestUpdateSubscription_UpgradeStripeEnabled_95(t *testing.T) {
	svc, db := setupTestService(t)

	// Seed free plan
	freePlan := seedFreePlan(t, db)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             freePlan.ID,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	// Upgrade from free to pro - should apply immediately (free has price 0)
	sub, err := svc.UpdateSubscription(context.Background(), 1, "pro")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have the new plan reference
	if sub.Plan == nil || sub.Plan.Name != "pro" {
		t.Error("expected plan to be updated to pro")
	}
}

// TestHandleSubscriptionCreated_CustomerIDNotFound_95 tests when customer ID lookup fails
func TestHandleSubscriptionCreated_CustomerIDNotFound_95(t *testing.T) {
	svc, _ := setupTestService(t)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_cust_not_found_95",
		EventType:      billing.WebhookEventLSSubscriptionCreated,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "ls_sub_new_95",
		CustomerID:     "nonexistent_customer_95",
		// No OrderNo fallback either
	}

	// Should complete without error (nothing found, nothing to update)
	err := svc.HandleSubscriptionCreated(c, event)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

// TestHandleSubscriptionCreated_NonLemonSqueezy_95 tests with non-LemonSqueezy provider
func TestHandleSubscriptionCreated_NonLemonSqueezy_95(t *testing.T) {
	svc, _ := setupTestService(t)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:        "evt_stripe_created_95",
		EventType:      "customer.subscription.created",
		Provider:       billing.PaymentProviderStripe, // Not LemonSqueezy
		SubscriptionID: "sub_stripe_new_95",
	}

	// Should complete without error (only LemonSqueezy is handled)
	err := svc.HandleSubscriptionCreated(c, event)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

// TestGetSeatUsage_WithMembers_95 tests seat usage with actual members
func TestGetSeatUsage_WithMembers_95(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro plan
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          10,
	})

	// Add some members
	db.Exec("INSERT INTO organization_members (organization_id, user_id, role) VALUES (1, 1, 'owner')")
	db.Exec("INSERT INTO organization_members (organization_id, user_id, role) VALUES (1, 2, 'member')")

	usage, err := svc.GetSeatUsage(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if usage.TotalSeats != 10 {
		t.Errorf("expected 10 total seats, got %d", usage.TotalSeats)
	}
	if usage.UsedSeats != 2 {
		t.Errorf("expected 2 used seats, got %d", usage.UsedSeats)
	}
	if usage.AvailableSeats != 8 {
		t.Errorf("expected 8 available seats, got %d", usage.AvailableSeats)
	}
	if !usage.CanAddSeats {
		t.Error("expected CanAddSeats to be true for pro plan")
	}
}

// TestCheckQuota_PodMinutes_95 tests pod minutes quota check
func TestCheckQuota_PodMinutes_95(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro plan: 1000 pod minutes
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	// Should pass with low pod minutes request
	err := svc.CheckQuota(context.Background(), 1, "pod_minutes", 100)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

// TestCheckQuota_WithPreloadedPlan_95 tests quota when plan reference is nil
func TestCheckQuota_WithPreloadedPlan_95(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          5,
	}
	db.Create(sub)

	// The plan will be loaded from DB if not preloaded
	err := svc.CheckQuota(context.Background(), 1, "users", 1)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

// TestHandleSubscriptionCreated_UpdateCustomerIDToo_95 tests updating subscription IDs
func TestHandleSubscriptionCreated_UpdateCustomerIDToo_95(t *testing.T) {
	svc, db := setupTestService(t)

	lsCustID := "ls_cust_partial_95"
	now := time.Now()
	// Create subscription with only customer ID (no subscription ID)
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
		EventID:        "evt_sub_update_ids_95",
		EventType:      billing.WebhookEventLSSubscriptionCreated,
		Provider:       billing.PaymentProviderLemonSqueezy,
		SubscriptionID: "ls_sub_new_id_95",
		CustomerID:     lsCustID,
	}

	err := svc.HandleSubscriptionCreated(c, event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.LemonSqueezySubscriptionID == nil || *sub.LemonSqueezySubscriptionID != "ls_sub_new_id_95" {
		t.Error("expected subscription ID to be set")
	}
}
