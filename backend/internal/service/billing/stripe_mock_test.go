package billing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stripe/stripe-go/v76"
	"gorm.io/gorm"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// ===========================================
// Tests Using Mock Stripe Client - Customer
// ===========================================

// setupTestServiceWithMockStripe creates a test service with mock Stripe client
func setupTestServiceWithMockStripe(t *testing.T) (*Service, *gorm.DB, *MockStripeClient) {
	db := setupTestDB(t)
	svc := NewService(newTestRepo(db), "")

	mockClient := NewMockStripeClient()
	svc.SetStripeEnabled(true)
	svc.SetStripeClient(mockClient)

	seedTestPlan(t, db)
	seedProPlan(t, db)
	seedEnterprisePlan(t, db)

	return svc, db, mockClient
}

// TestCreateStripeCustomer_WithMock tests customer creation using mock
func TestCreateStripeCustomer_WithMock(t *testing.T) {
	svc, db, mockClient := setupTestServiceWithMockStripe(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	customerID, err := svc.CreateStripeCustomer(context.Background(), 1, "test@example.com", "Test User")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if customerID == "" {
		t.Error("expected non-empty customer ID")
	}
	if len(mockClient.CreateCustomerCalls) != 1 {
		t.Errorf("expected 1 CreateCustomer call, got %d", len(mockClient.CreateCustomerCalls))
	}

	call := mockClient.CreateCustomerCalls[0]
	if stripe.StringValue(call.Params.Email) != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %s", stripe.StringValue(call.Params.Email))
	}

	sub, _ := svc.GetSubscription(context.Background(), 1)
	if sub.StripeCustomerID == nil || *sub.StripeCustomerID != customerID {
		t.Error("expected subscription to be updated with customer ID")
	}
}

// TestCreateStripeCustomer_Error tests customer creation error handling
func TestCreateStripeCustomer_Error(t *testing.T) {
	svc, _, mockClient := setupTestServiceWithMockStripe(t)

	mockClient.CreateCustomerErr = errors.New("stripe API error")

	_, err := svc.CreateStripeCustomer(context.Background(), 1, "test@example.com", "Test User")
	if err == nil {
		t.Error("expected error, got nil")
	}
	if err.Error() != "stripe API error" {
		t.Errorf("expected 'stripe API error', got %v", err)
	}
	if len(mockClient.CreateCustomerCalls) != 1 {
		t.Errorf("expected 1 CreateCustomer call, got %d", len(mockClient.CreateCustomerCalls))
	}
}
