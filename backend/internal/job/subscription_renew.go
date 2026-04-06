package job

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
	"gorm.io/gorm"
)

// SubscriptionRenewJob handles automatic subscription renewals
type SubscriptionRenewJob struct {
	db             *gorm.DB
	paymentFactory *payment.Factory
	logger         *slog.Logger
}

// NewSubscriptionRenewJob creates a new subscription renewal job
// cfg is the full application config, needed for URL derivation in payment providers
func NewSubscriptionRenewJob(db *gorm.DB, cfg *config.Config, logger *slog.Logger) *SubscriptionRenewJob {
	return &SubscriptionRenewJob{
		db:             db,
		paymentFactory: payment.NewFactoryWithLicenseRepo(cfg, nil),
		logger:         logger,
	}
}

// Run executes the subscription renewal job
// This should be called periodically (e.g., every hour) by a scheduler
func (j *SubscriptionRenewJob) Run(ctx context.Context) error {
	j.logger.Info("starting subscription renewal job")

	// Find subscriptions that need renewal
	subscriptions, err := j.findSubscriptionsForRenewal(ctx)
	if err != nil {
		return err
	}

	j.logger.Info("found subscriptions for renewal", "count", len(subscriptions))

	// Process each subscription
	for _, sub := range subscriptions {
		if err := j.processRenewal(ctx, &sub); err != nil {
			j.logger.Error("failed to process subscription renewal",
				"subscription_id", sub.ID,
				"organization_id", sub.OrganizationID,
				"error", err,
			)
			// Continue with other subscriptions
			continue
		}
	}

	j.logger.Info("subscription renewal job completed")
	return nil
}

// findSubscriptionsForRenewal finds subscriptions that need renewal
func (j *SubscriptionRenewJob) findSubscriptionsForRenewal(ctx context.Context) ([]billing.Subscription, error) {
	var subscriptions []billing.Subscription
	checkTime := time.Now().Add(24 * time.Hour)

	err := j.db.WithContext(ctx).
		Where("status = ?", billing.SubscriptionStatusActive).
		Where("auto_renew = ?", true).
		Where("current_period_end <= ?", checkTime).
		Where("current_period_end > ?", time.Now()). // Not yet expired
		Where("(alipay_agreement_no IS NOT NULL AND alipay_agreement_no != '') OR (wechat_contract_id IS NOT NULL AND wechat_contract_id != '')").
		Find(&subscriptions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find subscriptions for renewal: %w", err)
	}

	return subscriptions, nil
}

// processRenewal processes a single subscription renewal
func (j *SubscriptionRenewJob) processRenewal(ctx context.Context, sub *billing.Subscription) error {
	j.logger.Info("processing subscription renewal",
		"subscription_id", sub.ID,
		"organization_id", sub.OrganizationID,
		"provider", sub.PaymentProvider,
	)

	// Create payment order
	order, currency, err := j.createRenewalOrder(ctx, sub)
	if err != nil {
		return err
	}

	// Determine provider
	var provider string
	if sub.PaymentProvider != nil {
		provider = *sub.PaymentProvider
	}

	// Execute agreement payment based on provider
	var payErr error
	switch provider {
	case billing.PaymentProviderAlipay:
		payErr = j.executeAlipayAgreementPay(ctx, sub, order)
	case billing.PaymentProviderWeChat:
		payErr = j.executeWeChatAgreementPay(ctx, sub, order)
	default:
		// Stripe handles renewals automatically via their subscription system
		j.logger.Debug("skipping non-CN subscription renewal", "provider", provider)
		return nil
	}

	if payErr != nil {
		// Update order status to failed
		j.db.WithContext(ctx).Model(order).Updates(map[string]interface{}{
			"status":      billing.OrderStatusFailed,
			"fail_reason": payErr.Error(),
		})
		return payErr
	}

	// Currency is used for logging/debugging
	_ = currency
	return nil
}

// createRenewalOrder creates a payment order for subscription renewal
func (j *SubscriptionRenewJob) createRenewalOrder(ctx context.Context, sub *billing.Subscription) (*billing.PaymentOrder, string, error) {
	// Get the plan to calculate renewal amount
	var plan billing.SubscriptionPlan
	if err := j.db.WithContext(ctx).First(&plan, sub.PlanID).Error; err != nil {
		return nil, "", fmt.Errorf("failed to get plan: %w", err)
	}

	// Determine currency based on payment provider
	var provider string
	if sub.PaymentProvider != nil {
		provider = *sub.PaymentProvider
	}
	currency := billing.CurrencyUSD
	if provider == billing.PaymentProviderAlipay || provider == billing.PaymentProviderWeChat {
		currency = billing.CurrencyCNY
	}

	// Get price from plan_prices table (Single Source of Truth)
	amount, currency := j.calculateRenewalAmount(ctx, sub, &plan, currency)

	// Generate order number
	orderNo := fmt.Sprintf("RENEW-%d-%d", sub.OrganizationID, time.Now().Unix())

	// Create payment order
	expiresAt := time.Now().Add(24 * time.Hour)
	order := &billing.PaymentOrder{
		OrganizationID:  sub.OrganizationID,
		OrderNo:         orderNo,
		OrderType:       billing.OrderTypeRenewal,
		PaymentProvider: provider,
		PaymentMethod:   sub.PaymentMethod,
		Currency:        currency,
		Amount:          amount,
		ActualAmount:    amount,
		Status:          billing.OrderStatusPending,
		Metadata: map[string]interface{}{
			"subscription_id": sub.ID,
			"plan_id":         sub.PlanID,
			"seat_count":      sub.SeatCount,
			"billing_cycle":   sub.BillingCycle,
		},
		ExpiresAt: &expiresAt,
	}

	if err := j.db.WithContext(ctx).Create(order).Error; err != nil {
		return nil, "", fmt.Errorf("failed to create payment order: %w", err)
	}

	return order, currency, nil
}

// calculateRenewalAmount calculates the renewal amount based on plan prices
func (j *SubscriptionRenewJob) calculateRenewalAmount(ctx context.Context, sub *billing.Subscription, plan *billing.SubscriptionPlan, currency string) (float64, string) {
	var planPrice billing.PlanPrice
	if err := j.db.WithContext(ctx).
		Where("plan_id = ? AND currency = ?", sub.PlanID, currency).
		First(&planPrice).Error; err != nil {
		// Fallback to plan's legacy price fields if plan_prices not found
		j.logger.Warn("plan price not found, using legacy fields",
			"plan_id", sub.PlanID,
			"currency", currency,
		)
		var amount float64
		if sub.BillingCycle == billing.BillingCycleYearly {
			amount = plan.PricePerSeatYearly * float64(sub.SeatCount)
		} else {
			amount = plan.PricePerSeatMonthly * float64(sub.SeatCount)
		}
		return amount, billing.CurrencyUSD // Legacy prices are in USD
	}

	// Calculate renewal amount from plan_prices
	var amount float64
	if sub.BillingCycle == billing.BillingCycleYearly {
		amount = planPrice.PriceYearly * float64(sub.SeatCount)
	} else {
		amount = planPrice.PriceMonthly * float64(sub.SeatCount)
	}

	return amount, currency
}

// extendSubscription extends the subscription period after successful payment
func (j *SubscriptionRenewJob) extendSubscription(ctx context.Context, sub *billing.Subscription) error {
	var newPeriodEnd time.Time
	if sub.BillingCycle == billing.BillingCycleYearly {
		newPeriodEnd = sub.CurrentPeriodEnd.AddDate(1, 0, 0)
	} else {
		newPeriodEnd = sub.CurrentPeriodEnd.AddDate(0, 1, 0)
	}

	updates := map[string]interface{}{
		"current_period_start": sub.CurrentPeriodEnd,
		"current_period_end":   newPeriodEnd,
	}

	if err := j.db.WithContext(ctx).Model(sub).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to extend subscription: %w", err)
	}

	j.logger.Info("subscription extended",
		"subscription_id", sub.ID,
		"new_period_end", newPeriodEnd,
	)

	return nil
}
