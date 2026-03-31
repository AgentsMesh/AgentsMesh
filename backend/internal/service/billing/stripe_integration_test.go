
package billing

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stripe/stripe-go/v76"
	"gorm.io/gorm"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/testutil"
)

// Stripe Integration Tests - Run with: go test -tags=integration -v ./internal/service/billing/...
// Required: STRIPE_TEST_SECRET_KEY environment variable (sk_test_...)

func getStripeTestKey(t *testing.T) string {
	key := os.Getenv("STRIPE_TEST_SECRET_KEY")
	if key == "" {
		t.Skip("STRIPE_TEST_SECRET_KEY not set")
	}
	if len(key) < 7 || key[:7] != "sk_test" {
		t.Fatal("STRIPE_TEST_SECRET_KEY must be a test key (sk_test_...)")
	}
	return key
}

func setupStripeIntegrationTestDB(t *testing.T) *gorm.DB {
	return testutil.SetupTestDB(t)
}

func seedStripeIntegrationTestPlan(t *testing.T, db *gorm.DB) {
	plan := &billing.SubscriptionPlan{
		Name: "pro", DisplayName: "Pro",
		PricePerSeatMonthly: 1999, PricePerSeatYearly: 19990,
		MaxUsers: 10, MaxRunners: 5, MaxConcurrentPods: 3,
		MaxRepositories: 20, IncludedPodMinutes: 1000, IsActive: true,
	}
	if err := db.Create(plan).Error; err != nil {
		t.Fatalf("failed to seed test plan: %v", err)
	}
}

func TestStripeIntegration_CreateCustomer(t *testing.T) {
	stripeKey := getStripeTestKey(t)
	db := setupStripeIntegrationTestDB(t)
	seedStripeIntegrationTestPlan(t, db)

	svc := NewService(newTestRepo(db), stripeKey)
	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID: 1, PlanID: 1, Status: billing.SubscriptionStatusActive,
		BillingCycle: billing.BillingCycleMonthly, CurrentPeriodStart: now,
		CurrentPeriodEnd: now.AddDate(0, 1, 0), SeatCount: 1,
	})

	ctx := context.Background()
	customerID, err := svc.CreateStripeCustomer(ctx, 1, "integration-test@example.com", "Test User")
	if err != nil {
		t.Fatalf("failed to create Stripe customer: %v", err)
	}
	if customerID == "" || len(customerID) < 4 || customerID[:4] != "cus_" {
		t.Errorf("invalid customer ID: %s", customerID)
	}

	updatedSub, _ := svc.GetSubscription(ctx, 1)
	if updatedSub.StripeCustomerID == nil || *updatedSub.StripeCustomerID != customerID {
		t.Error("expected subscription to be updated with customer ID")
	}
	t.Logf("Created Stripe test customer: %s", customerID)
}

func TestStripeIntegration_DefaultStripeClient(t *testing.T) {
	stripeKey := getStripeTestKey(t)
	stripe.Key = stripeKey

	client := NewDefaultStripeClient()
	params := &stripe.CustomerParams{
		Email:    stripe.String("direct-client-test@example.com"),
		Name:     stripe.String("Direct Client Test"),
		Metadata: map[string]string{"test": "true"},
	}

	customer, err := client.CreateCustomer(params)
	if err != nil {
		t.Fatalf("failed to create customer: %v", err)
	}
	if customer.ID == "" || customer.Email != "direct-client-test@example.com" {
		t.Errorf("invalid customer data: %+v", customer)
	}
	t.Logf("Created customer via direct client: %s", customer.ID)
}

func TestStripeIntegration_StripeClientInterface(t *testing.T) {
	_ = getStripeTestKey(t)
	var _ StripeClient = (*DefaultStripeClient)(nil)
	var _ StripeClient = (*MockStripeClient)(nil)
}

func TestStripeIntegration_ServiceWithNilStripeClient(t *testing.T) {
	db := setupStripeIntegrationTestDB(t)
	seedStripeIntegrationTestPlan(t, db)

	svc := NewService(newTestRepo(db), "")
	customerID, err := svc.CreateStripeCustomer(context.Background(), 1, "test@example.com", "Test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if customerID != "" {
		t.Errorf("expected empty customer ID when Stripe disabled, got %s", customerID)
	}
}
