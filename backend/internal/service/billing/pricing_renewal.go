package billing

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

func (s *Service) CalculateRenewalPrice(ctx context.Context, orgID int64, newBillingCycle string) (*PriceCalculation, error) {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return nil, err
	}

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

func (s *Service) CalculateBillingCycleChangePrice(ctx context.Context, orgID int64, newBillingCycle string) (*PriceCalculation, error) {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return nil, err
	}

	if sub.BillingCycle == newBillingCycle {
		return nil, nil
	}

	plan, err := s.GetPlanByID(ctx, sub.PlanID)
	if err != nil {
		return nil, err
	}

	seats := sub.SeatCount
	if seats <= 0 {
		seats = 1
	}

	var currentPrice, newPrice float64
	if sub.BillingCycle == billing.BillingCycleYearly {
		currentPrice = plan.PricePerSeatYearly * float64(seats)
		newPrice = plan.PricePerSeatMonthly * float64(seats)
	} else {
		currentPrice = plan.PricePerSeatMonthly * float64(seats)
		newPrice = plan.PricePerSeatYearly * float64(seats)
	}

	ratio := calculateRemainingPeriodRatio(sub.CurrentPeriodStart, sub.CurrentPeriodEnd)

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
