package billing

import (
	"context"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
)

// PriceCalculation represents the result of a price calculation
type PriceCalculation struct {
	Amount       float64 `json:"amount"`        // Full price before proration
	ActualAmount float64 `json:"actual_amount"` // Prorated/discounted price to charge
	Currency     string  `json:"currency"`
	PlanID       int64   `json:"plan_id,omitempty"`
	Seats        int     `json:"seats"`
	BillingCycle string  `json:"billing_cycle"`
	Description  string  `json:"description,omitempty"`
	// Payment provider specific fields
	StripePrice           string `json:"stripe_price_id,omitempty"`         // Stripe Price ID
	LemonSqueezyVariantID string `json:"lemonsqueezy_variant_id,omitempty"` // LemonSqueezy Variant ID
}

// CalculateSubscriptionPriceWithCurrency calculates the price for a subscription in specific currency
func (s *Service) CalculateSubscriptionPriceWithCurrency(ctx context.Context, planName string, currency string, billingCycle string, seats int) (*PriceCalculation, error) {
	price, err := s.GetPlanPrice(ctx, planName, currency)
	if err != nil {
		return nil, err
	}

	if seats <= 0 {
		seats = 1
	}

	var amount float64
	var stripePrice, lsVariantID string
	if billingCycle == billing.BillingCycleYearly {
		amount = price.PriceYearly * float64(seats)
		if price.StripePriceIDYearly != nil {
			stripePrice = *price.StripePriceIDYearly
		}
		if price.LemonSqueezyVariantIDYearly != nil {
			lsVariantID = *price.LemonSqueezyVariantIDYearly
		}
	} else {
		amount = price.PriceMonthly * float64(seats)
		billingCycle = billing.BillingCycleMonthly
		if price.StripePriceIDMonthly != nil {
			stripePrice = *price.StripePriceIDMonthly
		}
		if price.LemonSqueezyVariantIDMonthly != nil {
			lsVariantID = *price.LemonSqueezyVariantIDMonthly
		}
	}

	return &PriceCalculation{
		Amount:                amount,
		ActualAmount:          amount,
		Currency:              currency,
		PlanID:                price.PlanID,
		Seats:                 seats,
		BillingCycle:          billingCycle,
		Description:           price.Plan.DisplayName + " subscription",
		StripePrice:           stripePrice,
		LemonSqueezyVariantID: lsVariantID,
	}, nil
}

// CalculateSubscriptionPrice calculates the price for a new subscription
// Uses USD by default. For multi-currency support, use CalculateSubscriptionPriceWithCurrency.
func (s *Service) CalculateSubscriptionPrice(ctx context.Context, planName string, billingCycle string, seats int) (*PriceCalculation, error) {
	return s.CalculateSubscriptionPriceWithCurrency(ctx, planName, billing.CurrencyUSD, billingCycle, seats)
}

// GetPricePreview returns a price preview without creating an order
func (s *Service) GetPricePreview(ctx context.Context, orgID int64, orderType string, planName string, billingCycle string, seats int) (*PriceCalculation, error) {
	switch orderType {
	case billing.OrderTypeSubscription:
		return s.CalculateSubscriptionPrice(ctx, planName, billingCycle, seats)
	case billing.OrderTypePlanUpgrade:
		return s.CalculateUpgradePrice(ctx, orgID, planName)
	case billing.OrderTypeSeatPurchase:
		return s.CalculateSeatPurchasePrice(ctx, orgID, seats)
	case billing.OrderTypeRenewal:
		return s.CalculateRenewalPrice(ctx, orgID, billingCycle)
	default:
		return nil, ErrInvalidOrderStatus
	}
}

// calculateRemainingPeriodRatio calculates the ratio of remaining time in the billing period
func calculateRemainingPeriodRatio(periodStart, periodEnd time.Time) float64 {
	now := time.Now()
	totalPeriod := periodEnd.Sub(periodStart).Hours()
	remainingPeriod := periodEnd.Sub(now).Hours()

	if remainingPeriod < 0 {
		remainingPeriod = 0
	}
	if totalPeriod <= 0 {
		return 0
	}

	return remainingPeriod / totalPeriod
}
