package billing

import (
	"context"
	"testing"
)

// ===========================================
// Service Core Tests - Plan Operations
// ===========================================

func TestGetPlan(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)

	plan, err := service.GetPlan(ctx, "based")
	if err != nil {
		t.Fatalf("failed to get plan: %v", err)
	}
	if plan.Name != "based" {
		t.Errorf("expected plan name 'free', got %s", plan.Name)
	}
}

func TestGetPlanNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	_, err := service.GetPlan(ctx, "nonexistent")
	if err != ErrPlanNotFound {
		t.Errorf("expected ErrPlanNotFound, got %v", err)
	}
}

func TestListPlans(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	seedProPlan(t, db)

	plans, err := service.ListPlans(ctx)
	if err != nil {
		t.Fatalf("failed to list plans: %v", err)
	}
	if len(plans) != 2 {
		t.Errorf("expected 2 plans, got %d", len(plans))
	}
}

func TestGetPlanByID(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	plan := seedTestPlan(t, db)

	result, err := service.GetPlanByID(ctx, plan.ID)
	if err != nil {
		t.Fatalf("failed to get plan by ID: %v", err)
	}
	if result.Name != "based" {
		t.Errorf("expected plan name 'free', got %s", result.Name)
	}
}

func TestGetPlanByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	_, err := service.GetPlanByID(ctx, 9999)
	if err != ErrPlanNotFound {
		t.Errorf("expected ErrPlanNotFound, got %v", err)
	}
}

func TestListPlansEmpty(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	plans, err := service.ListPlans(ctx)
	if err != nil {
		t.Fatalf("failed to list plans: %v", err)
	}
	if len(plans) != 0 {
		t.Errorf("expected 0 plans, got %d", len(plans))
	}
}
