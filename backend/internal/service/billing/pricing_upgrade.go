package billing

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

func (s *Service) CalculateUpgradePrice(ctx context.Context, orgID int64, newPlanName string) (*PriceCalculation, error) {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return nil, err
	}

	newPlan, err := s.GetPlan(ctx, newPlanName)
	if err != nil {
		return nil, err
	}

	newPlanPrice, err := s.GetPlanPrice(ctx, newPlanName, billing.CurrencyUSD)
	if err != nil {
		return nil, err
	}

	currentPlan, err := s.GetPlanByID(ctx, sub.PlanID)
	if err != nil {
		return nil, err
	}

	seats := sub.SeatCount
	if seats <= 0 {
		seats = 1
	}

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

	ratio := calculateRemainingPeriodRatio(sub.CurrentPeriodStart, sub.CurrentPeriodEnd)

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

func (s *Service) CalculateSeatPurchasePrice(ctx context.Context, orgID int64, additionalSeats int) (*PriceCalculation, error) {
	if additionalSeats <= 0 {
		return nil, ErrInvalidPlan
	}

	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return nil, err
	}

	plan, err := s.GetPlanByID(ctx, sub.PlanID)
	if err != nil {
		return nil, err
	}

	if plan.Name == billing.PlanBased {
		return nil, ErrInvalidPlan
	}

	if plan.MaxUsers > 0 && sub.SeatCount+additionalSeats > plan.MaxUsers {
		return nil, ErrQuotaExceeded
	}

	var pricePerSeat float64
	if sub.BillingCycle == billing.BillingCycleYearly {
		pricePerSeat = plan.PricePerSeatYearly
	} else {
		pricePerSeat = plan.PricePerSeatMonthly
	}

	amount := pricePerSeat * float64(additionalSeats)

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
