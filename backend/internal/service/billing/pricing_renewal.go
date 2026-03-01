package billing

import (
	"context"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// CalculateRenewalPrice calculates the price for renewing a subscription
func (s *Service) CalculateRenewalPrice(ctx context.Context, orgID int64, newBillingCycle string) (*PriceCalculation, error) {
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

	seats := sub.SeatCount
	if seats <= 0 {
		seats = 1
	}

	// Use new billing cycle if provided, otherwise keep current
	billingCycle := sub.BillingCycle
	if newBillingCycle != "" {
		billingCycle = newBillingCycle
	}

	var amount float64
	if billingCycle == billing.BillingCycleYearly {
		amount = plan.PricePerSeatYearly * float64(seats)
	} else {
		amount = plan.PricePerSeatMonthly * float64(seats)
	}

	return &PriceCalculation{
		Amount:       amount,
		ActualAmount: amount,
		Currency:     billing.CurrencyUSD,
		PlanID:       plan.ID,
		Seats:        seats,
		BillingCycle: billingCycle,
		Description:  "Renewal - " + plan.DisplayName,
	}, nil
}

// CalculateBillingCycleChangePrice calculates the price difference when changing billing cycle
func (s *Service) CalculateBillingCycleChangePrice(ctx context.Context, orgID int64, newBillingCycle string) (*PriceCalculation, error) {
	// Get current subscription
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// If same cycle, no change needed
	if sub.BillingCycle == newBillingCycle {
		return nil, nil
	}

	// Get plan
	plan, err := s.GetPlanByID(ctx, sub.PlanID)
	if err != nil {
		return nil, err
	}

	seats := sub.SeatCount
	if seats <= 0 {
		seats = 1
	}

	// Calculate prices for both cycles
	var currentPrice, newPrice float64
	if sub.BillingCycle == billing.BillingCycleYearly {
		currentPrice = plan.PricePerSeatYearly * float64(seats)
		newPrice = plan.PricePerSeatMonthly * float64(seats)
	} else {
		currentPrice = plan.PricePerSeatMonthly * float64(seats)
		newPrice = plan.PricePerSeatYearly * float64(seats)
	}

	// Calculate remaining period ratio
	ratio := calculateRemainingPeriodRatio(sub.CurrentPeriodStart, sub.CurrentPeriodEnd)

	// Calculate prorated difference
	// Positive = need to pay more (monthly to yearly)
	// Negative = credit (yearly to monthly, handled at renewal)
	actualAmount := (newPrice - currentPrice) * ratio
	if actualAmount < 0 {
		actualAmount = 0 // Credit will be applied at renewal
	}

	return &PriceCalculation{
		Amount:       newPrice,
		ActualAmount: actualAmount,
		Currency:     billing.CurrencyUSD,
		PlanID:       plan.ID,
		Seats:        seats,
		BillingCycle: newBillingCycle,
		Description:  "Billing cycle change",
	}, nil
}
