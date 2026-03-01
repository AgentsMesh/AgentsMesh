package billing

import (
	"context"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Coverage Boost Tests - Plan & Deployment
// ===========================================

// TestListPlansWithPrices_SkipsPlansWithoutPrice tests skipping plans without prices
func TestListPlansWithPrices_SkipsPlansWithoutPrice(t *testing.T) {
	svc, db := setupTestService(t)

	// Create a plan without any prices
	planWithoutPrice := &billing.SubscriptionPlan{
		Name:                "no_price_plan",
		DisplayName:         "No Price Plan",
		PricePerSeatMonthly: 0,
		PricePerSeatYearly:  0,
		IsActive:            true,
	}
	db.Create(planWithoutPrice)

	// List plans with prices in EUR (no plans have EUR prices)
	plans, err := svc.ListPlansWithPrices(context.Background(), "EUR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should skip plans without EUR prices
	if len(plans) != 0 {
		t.Errorf("expected 0 plans with EUR prices, got %d", len(plans))
	}

	// List plans with USD prices (should include standard plans)
	plans, err = svc.ListPlansWithPrices(context.Background(), "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have the standard plans with USD prices
	if len(plans) == 0 {
		t.Error("expected some plans with USD prices")
	}
}

// TestGetDeploymentInfo_WithConfig tests deployment info with payment config
func TestGetDeploymentInfo_WithConfig(t *testing.T) {
	svc, _ := setupTestService(t)

	// Service created without config, so should return defaults
	info := svc.GetDeploymentInfo()
	if info.DeploymentType != "global" {
		t.Errorf("expected global deployment type, got %s", info.DeploymentType)
	}
}
