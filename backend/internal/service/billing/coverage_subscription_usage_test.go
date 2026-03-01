package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Usage Tests
// ===========================================

// TestAdditionalGetUsage_NoUsageRecords tests getting usage when no records exist
func TestAdditionalGetUsage_NoUsageRecords(t *testing.T) {
	svc, db := setupTestService(t)

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2,
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleMonthly,
		CurrentPeriodStart: now.AddDate(0, 0, -15),
		CurrentPeriodEnd:   now.AddDate(0, 0, 15),
		SeatCount:          1,
	})

	usage, err := svc.GetUsage(context.Background(), 1, "pod_minutes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if usage != 0 {
		t.Errorf("expected 0 usage, got %f", usage)
	}
}

// ===========================================
// Invoice Tests
// ===========================================

// TestAdditionalGetInvoices_Empty tests getting invoices when none exist
func TestAdditionalGetInvoices_Empty(t *testing.T) {
	svc, _ := setupTestService(t)

	invoices, err := svc.GetInvoicesByOrg(context.Background(), 1, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(invoices) != 0 {
		t.Errorf("expected empty invoices, got %d", len(invoices))
	}
}

// TestAdditionalGetInvoices_WithData tests getting invoices with data
// Skipped: Invoice table schema has complex JSON fields that don't work well with SQLite test db
func TestAdditionalGetInvoices_WithData(t *testing.T) {
	t.Skip("Invoice table has complex JSON fields incompatible with SQLite test db")
}
