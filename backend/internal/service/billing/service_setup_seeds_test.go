package billing

import (
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"gorm.io/gorm"
)

func seedTestPlan(t *testing.T, db *gorm.DB) *billing.SubscriptionPlan {
	plan := &billing.SubscriptionPlan{
		Name:                "based",
		DisplayName:         "Based Plan",
		PricePerSeatMonthly: 9.9,
		PricePerSeatYearly:  99,
		IncludedPodMinutes:  100,
		PricePerExtraMinute: 0,
		MaxUsers:            1, // Based plan has fixed 1 seat
		MaxRunners:          1,
		MaxConcurrentPods:   5,
		MaxRepositories:     5,
		IsActive:            true,
	}
	if err := db.Create(plan).Error; err != nil {
		t.Fatalf("failed to seed plan: %v", err)
	}

	// Seed plan prices (Single Source of Truth)
	prices := []billing.PlanPrice{
		{PlanID: plan.ID, Currency: billing.CurrencyUSD, PriceMonthly: 9.9, PriceYearly: 99},
		{PlanID: plan.ID, Currency: billing.CurrencyCNY, PriceMonthly: 69, PriceYearly: 690},
	}
	for _, price := range prices {
		if err := db.Create(&price).Error; err != nil {
			t.Fatalf("failed to seed plan price: %v", err)
		}
	}

	return plan
}

func seedProPlan(t *testing.T, db *gorm.DB) *billing.SubscriptionPlan {
	plan := &billing.SubscriptionPlan{
		Name:                "pro",
		DisplayName:         "Pro Plan",
		PricePerSeatMonthly: 19.99,
		PricePerSeatYearly:  199.90,
		IncludedPodMinutes:  1000,
		PricePerExtraMinute: 0.05,
		MaxUsers:            50,
		MaxRunners:          10,
		MaxConcurrentPods:   5,
		MaxRepositories:     100,
		IsActive:            true,
	}
	if err := db.Create(plan).Error; err != nil {
		t.Fatalf("failed to seed pro plan: %v", err)
	}

	// Seed plan prices (Single Source of Truth)
	prices := []billing.PlanPrice{
		{PlanID: plan.ID, Currency: billing.CurrencyUSD, PriceMonthly: 19.99, PriceYearly: 199.90},
		{PlanID: plan.ID, Currency: billing.CurrencyCNY, PriceMonthly: 139, PriceYearly: 1390},
	}
	for _, price := range prices {
		if err := db.Create(&price).Error; err != nil {
			t.Fatalf("failed to seed pro plan price: %v", err)
		}
	}

	return plan
}

func seedEnterprisePlan(t *testing.T, db *gorm.DB) *billing.SubscriptionPlan {
	plan := &billing.SubscriptionPlan{
		Name:                "enterprise",
		DisplayName:         "Enterprise Plan",
		PricePerSeatMonthly: 99.99,
		PricePerSeatYearly:  999.90,
		IncludedPodMinutes:  -1, // unlimited
		PricePerExtraMinute: 0,
		MaxUsers:            -1, // unlimited
		MaxRunners:          -1,
		MaxConcurrentPods:   -1,
		MaxRepositories:     -1,
		IsActive:            true,
	}
	if err := db.Create(plan).Error; err != nil {
		t.Fatalf("failed to seed enterprise plan: %v", err)
	}

	// Seed plan prices (Single Source of Truth)
	prices := []billing.PlanPrice{
		{PlanID: plan.ID, Currency: billing.CurrencyUSD, PriceMonthly: 99.99, PriceYearly: 999.90},
		{PlanID: plan.ID, Currency: billing.CurrencyCNY, PriceMonthly: 690, PriceYearly: 6900},
	}
	for _, price := range prices {
		if err := db.Create(&price).Error; err != nil {
			t.Fatalf("failed to seed enterprise plan price: %v", err)
		}
	}

	return plan
}

// seedFreePlan creates a free plan with 0 price for testing upgrade from free plan
func seedFreePlan(t *testing.T, db *gorm.DB) *billing.SubscriptionPlan {
	plan := &billing.SubscriptionPlan{
		Name:                "free",
		DisplayName:         "Free Plan",
		PricePerSeatMonthly: 0, // Free
		PricePerSeatYearly:  0,
		IncludedPodMinutes:  10,
		PricePerExtraMinute: 0,
		MaxUsers:            1,
		MaxRunners:          1,
		MaxConcurrentPods:   1,
		MaxRepositories:     1,
		IsActive:            true,
	}
	if err := db.Create(plan).Error; err != nil {
		t.Fatalf("failed to seed free plan: %v", err)
	}

	// Seed plan prices (Single Source of Truth)
	prices := []billing.PlanPrice{
		{PlanID: plan.ID, Currency: billing.CurrencyUSD, PriceMonthly: 0, PriceYearly: 0},
		{PlanID: plan.ID, Currency: billing.CurrencyCNY, PriceMonthly: 0, PriceYearly: 0},
	}
	for _, price := range prices {
		if err := db.Create(&price).Error; err != nil {
			t.Fatalf("failed to seed free plan price: %v", err)
		}
	}

	return plan
}
