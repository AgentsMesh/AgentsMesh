package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Push 95% Coverage - Renewal Tests
// ===========================================

// TestCalculateRenewalPrice_YearlyCycleWithZeroSeats tests renewal with yearly and zero seats
func TestCalculateRenewalPrice_YearlyCycleWithZeroSeats(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleYearly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(1, 0, 0),
		SeatCount:          0, // Zero seats edge case
	})

	result, err := svc.CalculateRenewalPrice(context.Background(), 1, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Seats != 1 {
		t.Errorf("expected 1 seat (default), got %d", result.Seats)
	}
	if result.BillingCycle != billing.BillingCycleYearly {
		t.Errorf("expected yearly cycle, got %s", result.BillingCycle)
	}
}

// TestGetUsage_SubscriptionNotFound tests getting usage without subscription
func TestGetUsage_SubscriptionNotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.GetUsage(context.Background(), 999, billing.UsageTypePodMinutes)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}
