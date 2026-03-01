package billing

import (
	"context"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// CalculateUpgradePrice calculates the prorated price for upgrading to a new plan
func (s *Service) CalculateUpgradePrice(ctx context.Context, orgID int64, newPlanName string) (*PriceCalculation, error) {
	// Get current subscription
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// Get new plan
	newPlan, err := s.GetPlan(ctx, newPlanName)
	if err != nil {
		return nil, err
	}

	// Get new plan price for provider-specific IDs
	newPlanPrice, err := s.GetPlanPrice(ctx, newPlanName, billing.CurrencyUSD)
	if err != nil {
		return nil, err
	}

	// Get current plan
	currentPlan, err := s.GetPlanByID(ctx, sub.PlanID)
	if err != nil {
		return nil, err
	}

	seats := sub.SeatCount
	if seats <= 0 {
		seats = 1
	}

	// Calculate full prices and get provider-specific IDs
	var currentPrice, newPrice float64
	var stripePrice, lsVariantID string
	if sub.BillingCycle == billing.BillingCycleYearly {
		currentPrice = currentPlan.PricePerSeatYearly * float64(seats)
		newPrice = newPlan.PricePerSeatYearly * float64(seats)
		if newPlanPrice.StripePriceIDYearly != nil {
			stripePrice = *newPlanPrice.StripePriceIDYearly
		}
		if newPlanPrice.LemonSqueezyVariantIDYearly != nil {
			lsVariantID = *newPlanPrice.LemonSqueezyVariantIDYearly
		}
	} else {
		currentPrice = currentPlan.PricePerSeatMonthly * float64(seats)
		newPrice = newPlan.PricePerSeatMonthly * float64(seats)
		if newPlanPrice.StripePriceIDMonthly != nil {
			stripePrice = *newPlanPrice.StripePriceIDMonthly
		}
		if newPlanPrice.LemonSqueezyVariantIDMonthly != nil {
			lsVariantID = *newPlanPrice.LemonSqueezyVariantIDMonthly
		}
	}

	// Calculate remaining period ratio
	ratio := calculateRemainingPeriodRatio(sub.CurrentPeriodStart, sub.CurrentPeriodEnd)

	// Calculate prorated difference
	actualAmount := (newPrice - currentPrice) * ratio
	if actualAmount < 0 {
		actualAmount = 0 // Downgrade should use different flow
	}

	return &PriceCalculation{
		Amount:                newPrice,
		ActualAmount:          actualAmount,
		Currency:              billing.CurrencyUSD,
		PlanID:                newPlan.ID,
		Seats:                 seats,
		BillingCycle:          sub.BillingCycle,
		Description:           "Upgrade to " + newPlan.DisplayName,
		StripePrice:           stripePrice,
		LemonSqueezyVariantID: lsVariantID,
	}, nil
}

// CalculateSeatPurchasePrice calculates the prorated price for purchasing additional seats
func (s *Service) CalculateSeatPurchasePrice(ctx context.Context, orgID int64, additionalSeats int) (*PriceCalculation, error) {
	if additionalSeats <= 0 {
		return nil, ErrInvalidPlan
	}

	// Get current subscription
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// Get plan
	plan, err := s.GetPlanByID(ctx, sub.PlanID)
	if err != nil {
		return nil, err
	}

	// Check if plan allows adding seats (Based plan has fixed 1 seat)
	if plan.Name == billing.PlanBased {
		return nil, ErrInvalidPlan
	}

	// Check max seats
	if plan.MaxUsers != -1 && sub.SeatCount+additionalSeats > plan.MaxUsers {
		return nil, ErrQuotaExceeded
	}

	// Get price per seat based on billing cycle
	var pricePerSeat float64
	if sub.BillingCycle == billing.BillingCycleYearly {
		pricePerSeat = plan.PricePerSeatYearly
	} else {
		pricePerSeat = plan.PricePerSeatMonthly
	}

	// Calculate full amount
	amount := pricePerSeat * float64(additionalSeats)

	// Calculate prorated amount for remaining period
	ratio := calculateRemainingPeriodRatio(sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
	actualAmount := amount * ratio

	return &PriceCalculation{
		Amount:       amount,
		ActualAmount: actualAmount,
		Currency:     billing.CurrencyUSD,
		PlanID:       sub.PlanID,
		Seats:        additionalSeats,
		BillingCycle: sub.BillingCycle,
		Description:  "Additional seats",
	}, nil
}
