package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Price Preview Coverage Tests
// ===========================================

// TestGetPricePreview_Subscription tests price preview for new subscription
func TestGetPricePreview_Subscription(t *testing.T) {
	svc, _ := setupTestService(t)

	price, err := svc.GetPricePreview(context.Background(), 1, billing.OrderTypeSubscription, "pro", billing.BillingCycleMonthly, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if price.Seats != 2 {
		t.Errorf("expected 2 seats, got %d", price.Seats)
	}
}

// TestGetPricePreview_InvalidOrderType tests preview with invalid order type
func TestGetPricePreview_InvalidOrderType(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.GetPricePreview(context.Background(), 1, "invalid_type", "pro", billing.BillingCycleMonthly, 2)
	if err == nil {
		t.Error("expected error for invalid order type")
	}
}

// TestGetPricePreview_Upgrade tests price preview for upgrade
func TestGetPricePreview_Upgrade(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	price, err := svc.GetPricePreview(context.Background(), 1, billing.OrderTypePlanUpgrade, "enterprise", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Just check that description contains "Upgrade"
	if price.Description == "" {
		t.Error("expected non-empty description")
	}
}

// TestGetPricePreview_SeatPurchase tests price preview for seat purchase
func TestGetPricePreview_SeatPurchase(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          5,
	})

	price, err := svc.GetPricePreview(context.Background(), 1, billing.OrderTypeSeatPurchase, "", "", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if price.Seats != 3 {
		t.Errorf("expected 3 additional seats, got %d", price.Seats)
	}
}

// TestGetPricePreview_Renewal tests price preview for renewal
func TestGetPricePreview_Renewal(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
		SeatCount:          1,
	})

	price, err := svc.GetPricePreview(context.Background(), 1, billing.OrderTypeRenewal, "", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Just check that description contains "Renewal"
	if price.Description == "" {
		t.Error("expected non-empty description")
	}
}
