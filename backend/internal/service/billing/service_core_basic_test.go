package billing

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
)

// ===========================================
// Service Core Tests - Basic
// ===========================================

func TestNewService(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")

	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if service.stripeEnabled {
		t.Error("expected stripe to be disabled without key")
	}
}

func TestNewServiceWithStripeKey(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "sk_test_fake_key")

	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if !service.stripeEnabled {
		t.Error("expected stripe to be enabled with key")
	}
}

// testConfigWithPayment creates a test config with payment settings
func testConfigWithPayment(payment *config.PaymentConfig) *config.Config {
	if payment == nil {
		return &config.Config{
			PrimaryDomain: "localhost:10000",
			UseHTTPS:      false,
		}
	}
	return &config.Config{
		PrimaryDomain: "localhost:10000",
		UseHTTPS:      false,
		Payment:       *payment,
	}
}

func TestNewServiceWithConfig(t *testing.T) {
	db := setupTestDB(t)

	// Test with nil config
	service := NewServiceWithConfig(newTestRepo(db), nil)
	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if service.stripeEnabled {
		t.Error("expected stripe to be disabled without config")
	}

	// Test with mock enabled config
	appCfg := testConfigWithPayment(&config.PaymentConfig{
		DeploymentType: config.DeploymentGlobal,
		MockEnabled:    true,
	})
	service = NewServiceWithConfig(newTestRepo(db), appCfg)
	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if service.GetPaymentFactory() == nil {
		t.Error("expected payment factory to be set")
	}
}

func TestGetPaymentFactory(t *testing.T) {
	db := setupTestDB(t)
	appCfg := testConfigWithPayment(&config.PaymentConfig{MockEnabled: true})
	service := NewServiceWithConfig(newTestRepo(db), appCfg)

	factory := service.GetPaymentFactory()
	if factory == nil {
		t.Error("expected non-nil payment factory")
	}
}

func TestCreateStripeCustomerDisabled(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	customerID, err := service.CreateStripeCustomer(ctx, 1, "test@example.com", "Test Org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if customerID != "" {
		t.Error("expected empty customer ID when stripe is disabled")
	}
}

func TestGetDeploymentInfo(t *testing.T) {
	db := setupTestDB(t)

	// Without config
	service := NewService(newTestRepo(db), "")
	info := service.GetDeploymentInfo()
	if info.DeploymentType != "global" {
		t.Errorf("expected 'global', got %s", info.DeploymentType)
	}

	// With config
	appCfg := testConfigWithPayment(&config.PaymentConfig{
		DeploymentType: config.DeploymentCN,
		MockEnabled:    true,
	})
	service = NewServiceWithConfig(newTestRepo(db), appCfg)
	info = service.GetDeploymentInfo()
	if info.DeploymentType != "cn" {
		t.Errorf("expected 'cn', got %s", info.DeploymentType)
	}
}

func TestErrorVariables(t *testing.T) {
	errors := map[error]string{
		ErrSubscriptionNotFound:  "subscription not found",
		ErrPlanNotFound:          "plan not found",
		ErrQuotaExceeded:         "quota exceeded",
		ErrInvalidPlan:           "invalid plan",
		ErrOrderNotFound:         "order not found",
		ErrOrderExpired:          "order expired",
		ErrInvalidOrderStatus:    "invalid order status",
		ErrSeatCountExceedsLimit: "current seat count exceeds target plan limit",
	}

	for err, msg := range errors {
		if err.Error() != msg {
			t.Errorf("unexpected error message for %v: %s", err, err.Error())
		}
	}
}

func TestNewServiceWithConfigStripe(t *testing.T) {
	db := setupTestDB(t)

	// Test with Stripe key
	appCfg := testConfigWithPayment(&config.PaymentConfig{
		DeploymentType: config.DeploymentGlobal,
		Stripe: config.StripeConfig{
			SecretKey: "sk_test_fake",
		},
	})
	service := NewServiceWithConfig(newTestRepo(db), appCfg)
	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if !service.stripeEnabled {
		t.Error("expected stripe to be enabled with stripe config")
	}
}
