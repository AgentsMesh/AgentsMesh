package billing

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
	"gorm.io/gorm"
)

// ===========================================
// Integration Tests with Mock Provider
// These tests verify the complete payment flow
// ===========================================

func setupIntegrationTestService(t *testing.T) (*Service, *payment.Factory, *gorm.DB) {
	db := setupTestDB(t)

	// Seed plans
	seedTestPlan(t, db)
	seedProPlan(t, db)
	seedEnterprisePlan(t, db)

	// Create service with mock provider enabled
	appCfg := &config.Config{
		PrimaryDomain: "localhost:3000",
		UseHTTPS:      false,
		Payment: config.PaymentConfig{
			DeploymentType: config.DeploymentGlobal,
			MockEnabled:    true,
			MockBaseURL:    "http://localhost:3000",
		},
	}

	service := NewServiceWithConfig(newTestRepo(db), appCfg)
	factory := service.GetPaymentFactory()

	return service, factory, db
}

// TestIntegrationCreateSubscriptionFlow tests the complete subscription creation flow
func TestIntegrationCreateSubscriptionFlow(t *testing.T) {
	service, factory, _ := setupIntegrationTestService(t)
	ctx := context.Background()

	// 1. Create a based subscription first
	sub, err := service.CreateSubscription(ctx, 1, "based")
	if err != nil {
		t.Fatalf("failed to create based subscription: %v", err)
	}
	if sub.Status != billing.SubscriptionStatusActive {
		t.Errorf("expected active status, got %s", sub.Status)
	}

	// 2. Verify factory is available
	if factory == nil {
		t.Fatal("expected payment factory to be available")
	}
	if !factory.IsMockEnabled() {
		t.Error("expected mock to be enabled")
	}

	// 3. Get default provider (should be mock)
	provider, err := factory.GetDefaultProvider()
	if err != nil {
		t.Fatalf("failed to get default provider: %v", err)
	}
	if provider.GetProviderName() != "mock" {
		t.Errorf("expected mock provider, got %s", provider.GetProviderName())
	}
}
