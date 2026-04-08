package billing

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// ===========================================
// Usage and Quota Tests
// ===========================================

func TestRecordUsage(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")

	err := service.RecordUsage(ctx, 1, "pod_minutes", 5.0, billing.UsageMetadata{})
	if err != nil {
		t.Fatalf("failed to record usage: %v", err)
	}
}

func TestGetUsage(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")

	service.RecordUsage(ctx, 1, "pod_minutes", 5.0, billing.UsageMetadata{})
	service.RecordUsage(ctx, 1, "pod_minutes", 3.0, billing.UsageMetadata{})

	usage, err := service.GetUsage(ctx, 1, "pod_minutes")
	if err != nil {
		t.Fatalf("failed to get usage: %v", err)
	}
	if usage != 8.0 {
		t.Errorf("expected usage 8.0, got %f", usage)
	}
}

func TestGetUsageHistory(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")

	service.RecordUsage(ctx, 1, "pod_minutes", 5.0, billing.UsageMetadata{})

	records, err := service.GetUsageHistory(ctx, 1, "pod_minutes", 10)
	if err != nil {
		t.Fatalf("failed to get usage history: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}
}

func TestGetUsageHistoryAllTypes(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")

	service.RecordUsage(ctx, 1, "pod_minutes", 5.0, billing.UsageMetadata{})
	service.RecordUsage(ctx, 1, "storage_gb", 1.0, billing.UsageMetadata{})

	records, err := service.GetUsageHistory(ctx, 1, "", 10) // Empty type = all types
	if err != nil {
		t.Fatalf("failed to get usage history: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}
}

func TestRecordUsageNoSubscription(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	// No subscription exists
	err := service.RecordUsage(ctx, 999, "pod_minutes", 5.0, billing.UsageMetadata{})
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

func TestGetUsageNoSubscription(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	_, err := service.GetUsage(ctx, 999, "pod_minutes")
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

func TestGetCurrentConcurrentPods(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	db.Exec("INSERT INTO pods (organization_id, pod_key, status) VALUES (1, 'pod1', 'running')")
	db.Exec("INSERT INTO pods (organization_id, pod_key, status) VALUES (1, 'pod2', 'initializing')")
	db.Exec("INSERT INTO pods (organization_id, pod_key, status) VALUES (1, 'pod3', 'stopped')")

	count, err := service.GetCurrentConcurrentPods(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get concurrent pods: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 concurrent pods, got %d", count)
	}
}
