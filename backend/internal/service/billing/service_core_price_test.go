package billing

import (
	"context"
	"testing"
)

// ===========================================
// Service Core Tests - Plan Price Operations
// ===========================================

func TestGetPlanPrice(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db) // Seeds based plan with USD and CNY prices

	// Test USD price
	price, err := service.GetPlanPrice(ctx, "based", "USD")
	if err != nil {
		t.Fatalf("failed to get plan price: %v", err)
	}
	if price.PriceMonthly != 9.9 {
		t.Errorf("expected USD monthly price 9.9, got %f", price.PriceMonthly)
	}
	if price.PriceYearly != 99 {
		t.Errorf("expected USD yearly price 99, got %f", price.PriceYearly)
	}
	if price.Plan == nil {
		t.Error("expected Plan to be populated")
	}

	// Test CNY price
	price, err = service.GetPlanPrice(ctx, "based", "CNY")
	if err != nil {
		t.Fatalf("failed to get plan price: %v", err)
	}
	if price.PriceMonthly != 69 {
		t.Errorf("expected CNY monthly price 69, got %f", price.PriceMonthly)
	}
}

func TestGetPlanPricePlanNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	_, err := service.GetPlanPrice(ctx, "nonexistent", "USD")
	if err != ErrPlanNotFound {
		t.Errorf("expected ErrPlanNotFound, got %v", err)
	}
}

func TestGetPlanPriceCurrencyNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)

	_, err := service.GetPlanPrice(ctx, "based", "EUR")
	if err != ErrPriceNotFound {
		t.Errorf("expected ErrPriceNotFound, got %v", err)
	}
}

func TestGetPlanPriceByID(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	plan := seedTestPlan(t, db)

	price, err := service.GetPlanPriceByID(ctx, plan.ID, "USD")
	if err != nil {
		t.Fatalf("failed to get plan price by ID: %v", err)
	}
	if price.PriceMonthly != 9.9 {
		t.Errorf("expected monthly price 9.9, got %f", price.PriceMonthly)
	}
	if price.Plan == nil {
		t.Error("expected Plan to be preloaded")
	}
}

func TestGetPlanPriceByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	_, err := service.GetPlanPriceByID(ctx, 9999, "USD")
	if err != ErrPriceNotFound {
		t.Errorf("expected ErrPriceNotFound, got %v", err)
	}
}

func TestGetPlanPrices(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db) // Seeds with USD and CNY prices

	prices, err := service.GetPlanPrices(ctx, "based")
	if err != nil {
		t.Fatalf("failed to get plan prices: %v", err)
	}
	if len(prices) != 2 {
		t.Errorf("expected 2 prices (USD and CNY), got %d", len(prices))
	}

	// Verify both currencies are present
	currencies := make(map[string]bool)
	for _, p := range prices {
		currencies[p.Currency] = true
		if p.Plan == nil {
			t.Error("expected Plan to be attached to each price")
		}
	}
	if !currencies["USD"] || !currencies["CNY"] {
		t.Error("expected both USD and CNY prices")
	}
}

func TestGetPlanPricesPlanNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	_, err := service.GetPlanPrices(ctx, "nonexistent")
	if err != ErrPlanNotFound {
		t.Errorf("expected ErrPlanNotFound, got %v", err)
	}
}

func TestListPlansWithPrices(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	seedProPlan(t, db)

	// List with USD prices
	plansWithPrices, err := service.ListPlansWithPrices(ctx, "USD")
	if err != nil {
		t.Fatalf("failed to list plans with prices: %v", err)
	}
	if len(plansWithPrices) != 2 {
		t.Errorf("expected 2 plans, got %d", len(plansWithPrices))
	}

	for _, pwp := range plansWithPrices {
		if pwp.Plan == nil {
			t.Error("expected Plan to be set")
		}
		if pwp.Price == nil {
			t.Error("expected Price to be set")
		}
		if pwp.Price.Currency != "USD" {
			t.Errorf("expected USD currency, got %s", pwp.Price.Currency)
		}
	}
}

func TestListPlansWithPricesCNY(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	seedProPlan(t, db)

	// List with CNY prices
	plansWithPrices, err := service.ListPlansWithPrices(ctx, "CNY")
	if err != nil {
		t.Fatalf("failed to list plans with prices: %v", err)
	}
	if len(plansWithPrices) != 2 {
		t.Errorf("expected 2 plans, got %d", len(plansWithPrices))
	}

	for _, pwp := range plansWithPrices {
		if pwp.Price.Currency != "CNY" {
			t.Errorf("expected CNY currency, got %s", pwp.Price.Currency)
		}
	}
}

func TestListPlansWithPricesNoCurrency(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)

	// List with non-existent currency - should return empty
	plansWithPrices, err := service.ListPlansWithPrices(ctx, "EUR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plansWithPrices) != 0 {
		t.Errorf("expected 0 plans for EUR currency, got %d", len(plansWithPrices))
	}
}
