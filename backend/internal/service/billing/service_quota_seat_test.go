package billing

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// ===========================================
// Seat Availability and Usage Tests
// ===========================================

func TestCheckSeatAvailability(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)

	plan, _ := service.GetPlan(ctx, "based")
	now := time.Now()
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             plan.ID,
		SeatCount:          5,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
	}
	db.Create(sub)

	// Add 2 members
	db.Exec("INSERT INTO organization_members (organization_id, user_id, role) VALUES (1, 1, 'owner')")
	db.Exec("INSERT INTO organization_members (organization_id, user_id, role) VALUES (1, 2, 'member')")

	// Should have 3 available seats
	err := service.CheckSeatAvailability(ctx, 1, 3)
	if err != nil {
		t.Errorf("expected 3 available seats, got error: %v", err)
	}

	// Should fail for 4 seats
	err = service.CheckSeatAvailability(ctx, 1, 4)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

func TestCheckSeatAvailabilityWithPendingInvitations(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)

	plan, _ := service.GetPlan(ctx, "based")
	now := time.Now()
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             plan.ID,
		SeatCount:          3,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
	}
	db.Create(sub)

	// 1 member
	db.Exec("INSERT INTO organization_members (organization_id, user_id, role) VALUES (1, 1, 'owner')")
	// 1 pending invitation
	db.Exec("INSERT INTO invitations (organization_id, email, expires_at) VALUES (1, 'test@example.com', ?)", now.Add(24*time.Hour))

	// 3 seats - 1 used - 1 pending = 1 available
	err := service.CheckSeatAvailability(ctx, 1, 1)
	if err != nil {
		t.Errorf("expected 1 available seat, got error: %v", err)
	}

	err = service.CheckSeatAvailability(ctx, 1, 2)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

func TestCheckSeatAvailabilityNoSubscription(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	// No subscription = default 1 seat
	// Add 1 member
	db.Exec("INSERT INTO organization_members (organization_id, user_id, role) VALUES (1, 1, 'owner')")

	// Should fail to add another (no more seats available)
	err := service.CheckSeatAvailability(ctx, 1, 1)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

func TestGetSeatUsage(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedProPlan(t, db) // Use Pro plan for CanAddSeats = true

	plan, _ := service.GetPlan(ctx, "pro")
	now := time.Now()
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             plan.ID,
		SeatCount:          5,
		Status:             billing.SubscriptionStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
	}
	db.Create(sub)

	// Add 2 members
	db.Exec("INSERT INTO organization_members (organization_id, user_id, role) VALUES (1, 1, 'owner')")
	db.Exec("INSERT INTO organization_members (organization_id, user_id, role) VALUES (1, 2, 'member')")

	usage, err := service.GetSeatUsage(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get seat usage: %v", err)
	}
	if usage.TotalSeats != 5 {
		t.Errorf("expected 5 total seats, got %d", usage.TotalSeats)
	}
	if usage.UsedSeats != 2 {
		t.Errorf("expected 2 used seats, got %d", usage.UsedSeats)
	}
	if usage.AvailableSeats != 3 {
		t.Errorf("expected 3 available seats, got %d", usage.AvailableSeats)
	}
	if !usage.CanAddSeats {
		t.Error("expected CanAddSeats to be true for non-based plan")
	}
}

func TestGetSeatUsageFreePlan(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	service.CreateSubscription(ctx, 1, "based")

	usage, err := service.GetSeatUsage(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get seat usage: %v", err)
	}
	if usage.CanAddSeats {
		t.Error("expected CanAddSeats to be false for based plan")
	}
}

func TestGetSeatUsageNoSubscription(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	_, err := service.GetSeatUsage(ctx, 999)
	if err != ErrSubscriptionNotFound {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

func TestGetSeatUsageWithNilPlan(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	plan := seedProPlan(t, db)

	// Create subscription without plan preloaded
	now := time.Now()
	sub := &billing.Subscription{
		OrganizationID:     1,
		PlanID:             plan.ID,
		SeatCount:          5,
		Status:             billing.SubscriptionStatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0),
	}
	db.Create(sub)

	// GetSeatUsage should fetch plan if nil
	usage, err := service.GetSeatUsage(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get seat usage: %v", err)
	}
	if usage.MaxSeats != 50 { // Pro plan max_users
		t.Errorf("expected max seats 50, got %d", usage.MaxSeats)
	}
}
