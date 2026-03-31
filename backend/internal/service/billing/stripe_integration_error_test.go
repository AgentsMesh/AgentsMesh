
package billing

import (
	"context"
	"testing"
	"time"

	"github.com/stripe/stripe-go/v76"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// ===========================================
// Stripe Integration Tests - Error Handling & Comparison
// ===========================================

// TestStripeIntegration_MockVsReal compares mock and real client behavior
func TestStripeIntegration_MockVsReal(t *testing.T) {
	stripeKey := getStripeTestKey(t)
	db := setupStripeIntegrationTestDB(t)
	seedStripeIntegrationTestPlan(t, db)

	// Test with mock client
	mockClient := NewMockStripeClient()
	mockParams := &stripe.CustomerParams{
		Email: stripe.String("mock@example.com"),
		Name:  stripe.String("Mock User"),
	}
	mockCustomer, err := mockClient.CreateCustomer(mockParams)
	if err != nil {
		t.Fatalf("mock client error: %v", err)
	}

	// Test with real client
	stripe.Key = stripeKey
	realClient := NewDefaultStripeClient()
	realParams := &stripe.CustomerParams{
		Email: stripe.String("real@example.com"),
		Name:  stripe.String("Real User"),
	}
	realCustomer, err := realClient.CreateCustomer(realParams)
	if err != nil {
		t.Fatalf("real client error: %v", err)
	}

	// Both should return valid customer IDs
	if mockCustomer.ID == "" {
		t.Error("mock customer ID should not be empty")
	}
	if realCustomer.ID == "" {
		t.Error("real customer ID should not be empty")
	}

	// Real ID starts with cus_
	if len(realCustomer.ID) < 4 || realCustomer.ID[:4] != "cus_" {
		t.Errorf("real customer ID should start with 'cus_', got %s", realCustomer.ID)
	}

	// Both should preserve email
	if mockCustomer.Email != "mock@example.com" {
		t.Errorf("mock email mismatch: %s", mockCustomer.Email)
	}
	if realCustomer.Email != "real@example.com" {
		t.Errorf("real email mismatch: %s", realCustomer.Email)
	}

	t.Logf("Mock customer: %s, Real customer: %s", mockCustomer.ID, realCustomer.ID)
}

// TestStripeIntegration_CancelSubscription_InvalidID tests canceling non-existent subscription
func TestStripeIntegration_CancelSubscription_InvalidID(t *testing.T) {
	stripeKey := getStripeTestKey(t)

	stripe.Key = stripeKey
	client := NewDefaultStripeClient()

	// Try to cancel a non-existent subscription
	_, err := client.CancelSubscription("sub_nonexistent_12345", nil)
	if err == nil {
		t.Error("expected error when canceling non-existent subscription")
	}

	t.Logf("Expected error received: %v", err)
}

// TestStripeIntegration_ErrorHandling tests error handling for API errors
func TestStripeIntegration_ErrorHandling(t *testing.T) {
	stripeKey := getStripeTestKey(t)
	db := setupStripeIntegrationTestDB(t)
	seedStripeIntegrationTestPlan(t, db)

	svc := NewService(newTestRepo(db), stripeKey)
	ctx := context.Background()

	// Create subscription first
	now := time.Now()
	sub := &billing.Subscription{
		OrganizationID:     3,
		PlanID:             1,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	}
	db.Create(sub)

	// This should work - Stripe accepts most email formats
	customerID, err := svc.CreateStripeCustomer(ctx, 3, "valid@example.com", "Error Test User")
	if err != nil {
		t.Logf("CreateStripeCustomer returned error (may be expected): %v", err)
	} else {
		t.Logf("Created customer successfully: %s", customerID)
	}
}
