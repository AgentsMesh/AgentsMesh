package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Usage and Invoice Tests from Coverage 95
// ===========================================

// TestGetUsage_WithActualUsageRecords_95 tests getting usage with actual records
func TestGetUsage_WithActualUsageRecords_95(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	periodStart := now.AddDate(0, 0, -15)
	periodEnd := now.AddDate(0, 0, 15)

	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   periodEnd,
		SeatCount:          1,
	})

	// Create usage record within period
	db.Create(&billing.UsageRecord{
		OrganizationID: 1,
		UsageType:      billing.UsageTypePodMinutes,
		Quantity:       150.5,
		PeriodStart:    periodStart,
		PeriodEnd:      periodEnd,
	})

	usage, err := svc.GetUsage(context.Background(), 1, billing.UsageTypePodMinutes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if usage != 150.5 {
		t.Errorf("expected usage 150.5, got %f", usage)
	}
}

// TestGetInvoices_WithLimit_95 tests getting invoices with limit
func TestGetInvoices_WithLimit_95(t *testing.T) {
	svc, _ := setupTestService(t)

	invoices, err := svc.GetInvoicesByOrg(context.Background(), 1, 5, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return empty list (no invoices)
	if len(invoices) != 0 {
		t.Errorf("expected 0 invoices, got %d", len(invoices))
	}
}

// TestGetInvoices_WithoutLimit_95 tests getting invoices without limit
func TestGetInvoices_WithoutLimit_95(t *testing.T) {
	svc, _ := setupTestService(t)

	invoices, err := svc.GetInvoicesByOrg(context.Background(), 1, 0, 0) // No limit
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return empty list (no invoices)
	if len(invoices) != 0 {
		t.Errorf("expected 0 invoices, got %d", len(invoices))
	}
}

// TestCheckAndMarkWebhookProcessed_Success_95 tests idempotency with non-duplicate error
func TestCheckAndMarkWebhookProcessed_Success_95(t *testing.T) {
	svc, _ := setupTestService(t)

	ctx := context.Background()
	// First call should succeed
	err := svc.CheckAndMarkWebhookProcessed(ctx, "unique_event_id_95", billing.PaymentProviderLemonSqueezy, "test_event")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second call should return duplicate error
	err = svc.CheckAndMarkWebhookProcessed(ctx, "unique_event_id_95", billing.PaymentProviderLemonSqueezy, "test_event")
	if err != ErrWebhookAlreadyProcessed {
		t.Errorf("expected ErrWebhookAlreadyProcessed, got %v", err)
	}
}
