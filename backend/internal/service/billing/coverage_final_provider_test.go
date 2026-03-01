package billing

import (
	"context"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// ===========================================
// Final Coverage Tests - Provider IDs
// ===========================================

// TestCalculateSubscriptionPriceWithCurrency_ProviderIDs tests provider-specific IDs
func TestCalculateSubscriptionPriceWithCurrency_ProviderIDs(t *testing.T) {
	svc, db := setupTestService(t)

	// Add provider-specific IDs to plan prices
	var price billing.PlanPrice
	db.Where("plan_id = ? AND currency = ?", 2, "USD").First(&price)

	stripeMonthly := "price_stripe_monthly"
	stripeYearly := "price_stripe_yearly"
	lsMonthly := "var_ls_monthly"
	lsYearly := "var_ls_yearly"

	db.Model(&price).Updates(map[string]interface{}{
		"stripe_price_id_monthly":         &stripeMonthly,
		"stripe_price_id_yearly":          &stripeYearly,
		"lemonsqueezy_variant_id_monthly": &lsMonthly,
		"lemonsqueezy_variant_id_yearly":  &lsYearly,
	})

	// Test monthly cycle with provider IDs
	result, err := svc.CalculateSubscriptionPriceWithCurrency(context.Background(), "pro", "USD", billing.BillingCycleMonthly, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.StripePrice != stripeMonthly {
		t.Errorf("expected Stripe price %s, got %s", stripeMonthly, result.StripePrice)
	}
	if result.LemonSqueezyVariantID != lsMonthly {
		t.Errorf("expected LS variant %s, got %s", lsMonthly, result.LemonSqueezyVariantID)
	}

	// Test yearly cycle with provider IDs
	result, err = svc.CalculateSubscriptionPriceWithCurrency(context.Background(), "pro", "USD", billing.BillingCycleYearly, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.StripePrice != stripeYearly {
		t.Errorf("expected Stripe price %s, got %s", stripeYearly, result.StripePrice)
	}
	if result.LemonSqueezyVariantID != lsYearly {
		t.Errorf("expected LS variant %s, got %s", lsYearly, result.LemonSqueezyVariantID)
	}
}

// TestCalculateUpgradePrice_WithProviderIDs tests upgrade pricing with provider IDs
func TestCalculateUpgradePrice_WithProviderIDs(t *testing.T) {
	svc, db := setupTestService(t)

	// Add provider IDs to enterprise plan
	var price billing.PlanPrice
	db.Where("plan_id = ? AND currency = ?", 3, "USD").First(&price)

	stripeYearly := "price_enterprise_yearly"
	lsYearly := "var_enterprise_yearly"

	db.Model(&price).Updates(map[string]interface{}{
		"stripe_price_id_yearly":         &stripeYearly,
		"lemonsqueezy_variant_id_yearly": &lsYearly,
	})

	now := time.Now()
	db.Create(&billing.Subscription{
		OrganizationID:     1,
		PlanID:             2, // pro
		Status:             billing.SubscriptionStatusActive,
		BillingCycle:       billing.BillingCycleYearly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(1, 0, 0),
		SeatCount:          1,
	})

	result, err := svc.CalculateUpgradePrice(context.Background(), 1, "enterprise")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.StripePrice != stripeYearly {
		t.Errorf("expected Stripe price %s, got %s", stripeYearly, result.StripePrice)
	}
	if result.LemonSqueezyVariantID != lsYearly {
		t.Errorf("expected LS variant %s, got %s", lsYearly, result.LemonSqueezyVariantID)
	}
}

// TestCalculateUpgradePrice_MonthlyWithProviderIDs tests monthly upgrade with provider IDs
func TestCalculateUpgradePrice_MonthlyWithProviderIDs(t *testing.T) {
	svc, db := setupTestService(t)

	// Add provider IDs to enterprise plan
	var price billing.PlanPrice
	db.Where("plan_id = ? AND currency = ?", 3, "USD").First(&price)

	stripeMonthly := "price_enterprise_monthly"
	lsMonthly := "var_enterprise_monthly"

	db.Model(&price).Updates(map[string]interface{}{
		"stripe_price_id_monthly":         &stripeMonthly,
		"lemonsqueezy_variant_id_monthly": &lsMonthly,
	})

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

	result, err := svc.CalculateUpgradePrice(context.Background(), 1, "enterprise")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.StripePrice != stripeMonthly {
		t.Errorf("expected Stripe price %s, got %s", stripeMonthly, result.StripePrice)
	}
	if result.LemonSqueezyVariantID != lsMonthly {
		t.Errorf("expected LS variant %s, got %s", lsMonthly, result.LemonSqueezyVariantID)
	}
}
