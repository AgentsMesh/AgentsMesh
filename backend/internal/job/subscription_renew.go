package job

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment/types"
	"gorm.io/gorm"
)

// SubscriptionRenewJob handles automatic subscription renewals
type SubscriptionRenewJob struct {
	db             *gorm.DB
	paymentFactory *payment.Factory
	logger         *slog.Logger
}

// NewSubscriptionRenewJob creates a new subscription renewal job
func NewSubscriptionRenewJob(db *gorm.DB, cfg *config.PaymentConfig, logger *slog.Logger) *SubscriptionRenewJob {
	return &SubscriptionRenewJob{
		db:             db,
		paymentFactory: payment.NewFactoryWithDB(cfg, db),
		logger:         logger,
	}
}

// Run executes the subscription renewal job
// This should be called periodically (e.g., every hour) by a scheduler
func (j *SubscriptionRenewJob) Run(ctx context.Context) error {
	j.logger.Info("starting subscription renewal job")

	// Find subscriptions that need renewal
	// - status is active
	// - auto_renew is true
	// - current_period_end is within the next 24 hours
	// - has a valid agreement (for CN payments)
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
		return fmt.Errorf("failed to find subscriptions for renewal: %w", err)
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

// processRenewal processes a single subscription renewal
func (j *SubscriptionRenewJob) processRenewal(ctx context.Context, sub *billing.Subscription) error {
	j.logger.Info("processing subscription renewal",
		"subscription_id", sub.ID,
		"organization_id", sub.OrganizationID,
		"provider", sub.PaymentProvider,
	)

	// Get the plan to calculate renewal amount
	var plan billing.SubscriptionPlan
	if err := j.db.WithContext(ctx).First(&plan, sub.PlanID).Error; err != nil {
		return fmt.Errorf("failed to get plan: %w", err)
	}

	// Calculate renewal amount
	var amount float64
	if sub.BillingCycle == billing.BillingCycleYearly {
		amount = plan.PricePerSeatYearly * float64(sub.SeatCount)
	} else {
		amount = plan.PricePerSeatMonthly * float64(sub.SeatCount)
	}

	// Generate order number
	orderNo := fmt.Sprintf("RENEW-%d-%d", sub.OrganizationID, time.Now().Unix())

	// Create payment order
	expiresAt := time.Now().Add(24 * time.Hour)
	var provider string
	if sub.PaymentProvider != nil {
		provider = *sub.PaymentProvider
	}
	order := &billing.PaymentOrder{
		OrganizationID:  sub.OrganizationID,
		OrderNo:         orderNo,
		OrderType:       billing.OrderTypeRenewal,
		PaymentProvider: provider,
		PaymentMethod:   sub.PaymentMethod,
		Currency:        "CNY",
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
		return fmt.Errorf("failed to create payment order: %w", err)
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

	return nil
}

// executeAlipayAgreementPay executes Alipay agreement payment (代扣)
func (j *SubscriptionRenewJob) executeAlipayAgreementPay(ctx context.Context, sub *billing.Subscription, order *billing.PaymentOrder) error {
	if sub.AlipayAgreementNo == nil || *sub.AlipayAgreementNo == "" {
		return fmt.Errorf("no alipay agreement found")
	}

	provider, err := j.paymentFactory.GetProvider(billing.PaymentProviderAlipay)
	if err != nil {
		return fmt.Errorf("failed to get alipay provider: %w", err)
	}

	// Type assert to AgreementProvider
	agreementProvider, ok := provider.(payment.AgreementProvider)
	if !ok {
		return fmt.Errorf("alipay provider does not support agreements")
	}

	// Execute agreement payment
	resp, err := agreementProvider.ExecuteAgreementPay(ctx, &types.AgreementPayRequest{
		AgreementNo:    *sub.AlipayAgreementNo,
		OrderNo:        order.OrderNo,
		Amount:         order.ActualAmount,
		Currency:       order.Currency,
		Description:    fmt.Sprintf("AgentsMesh Subscription Renewal - %s", sub.BillingCycle),
		IdempotencyKey: order.OrderNo,
	})
	if err != nil {
		return fmt.Errorf("alipay agreement pay failed: %w", err)
	}

	// Update order with transaction info
	updates := map[string]interface{}{
		"external_order_no": resp.TransactionID,
	}

	if resp.Status == "success" {
		updates["status"] = billing.OrderStatusSucceeded
		updates["paid_at"] = resp.PaidAt
	}

	if err := j.db.WithContext(ctx).Model(order).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	// If payment succeeded, extend subscription period
	if resp.Status == "success" {
		return j.extendSubscription(ctx, sub)
	}

	return nil
}

// executeWeChatAgreementPay executes WeChat agreement payment (委托代扣)
func (j *SubscriptionRenewJob) executeWeChatAgreementPay(ctx context.Context, sub *billing.Subscription, order *billing.PaymentOrder) error {
	if sub.WeChatContractID == nil || *sub.WeChatContractID == "" {
		return fmt.Errorf("no wechat contract found")
	}

	provider, err := j.paymentFactory.GetProvider(billing.PaymentProviderWeChat)
	if err != nil {
		return fmt.Errorf("failed to get wechat provider: %w", err)
	}

	// Type assert to AgreementProvider
	agreementProvider, ok := provider.(payment.AgreementProvider)
	if !ok {
		return fmt.Errorf("wechat provider does not support agreements")
	}

	// Execute agreement payment
	resp, err := agreementProvider.ExecuteAgreementPay(ctx, &types.AgreementPayRequest{
		AgreementNo:    *sub.WeChatContractID,
		OrderNo:        order.OrderNo,
		Amount:         order.ActualAmount,
		Currency:       order.Currency,
		Description:    fmt.Sprintf("AgentsMesh Subscription Renewal - %s", sub.BillingCycle),
		IdempotencyKey: order.OrderNo,
	})
	if err != nil {
		return fmt.Errorf("wechat agreement pay failed: %w", err)
	}

	// Update order with transaction info
	updates := map[string]interface{}{
		"external_order_no": resp.TransactionID,
	}

	if resp.Status == "success" {
		updates["status"] = billing.OrderStatusSucceeded
		updates["paid_at"] = resp.PaidAt
	}

	if err := j.db.WithContext(ctx).Model(order).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	// If payment succeeded, extend subscription period
	if resp.Status == "success" {
		return j.extendSubscription(ctx, sub)
	}

	return nil
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

// FreezeExpiredSubscriptions freezes subscriptions that have expired without renewal
// This should be called periodically (e.g., every hour)
func (j *SubscriptionRenewJob) FreezeExpiredSubscriptions(ctx context.Context) error {
	j.logger.Info("checking for expired subscriptions to freeze")

	// Find subscriptions that:
	// - status is active
	// - current_period_end has passed
	// - are not set to cancel (they would have been canceled already)
	result := j.db.WithContext(ctx).
		Model(&billing.Subscription{}).
		Where("status = ?", billing.SubscriptionStatusActive).
		Where("current_period_end < ?", time.Now()).
		Where("cancel_at_period_end = ?", false).
		Updates(map[string]interface{}{
			"status":    billing.SubscriptionStatusFrozen,
			"frozen_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to freeze expired subscriptions: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		j.logger.Info("froze expired subscriptions", "count", result.RowsAffected)
	}

	return nil
}

// SendRenewalReminders sends reminder emails for upcoming renewals
// This should be called daily
func (j *SubscriptionRenewJob) SendRenewalReminders(ctx context.Context) error {
	j.logger.Info("sending renewal reminder emails")

	// Find subscriptions expiring in 7 days, 3 days, or 1 day
	reminderDays := []int{7, 3, 1}

	for _, days := range reminderDays {
		targetDate := time.Now().AddDate(0, 0, days)
		startOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, time.UTC)
		endOfDay := startOfDay.AddDate(0, 0, 1)

		var subscriptions []billing.Subscription
		err := j.db.WithContext(ctx).
			Where("status = ?", billing.SubscriptionStatusActive).
			Where("auto_renew = ?", false). // Only remind manual renewal users
			Where("current_period_end >= ?", startOfDay).
			Where("current_period_end < ?", endOfDay).
			Find(&subscriptions).Error
		if err != nil {
			j.logger.Error("failed to find subscriptions for reminder",
				"days", days,
				"error", err,
			)
			continue
		}

		for _, sub := range subscriptions {
			// TODO: Send email reminder
			// This would integrate with the email service
			j.logger.Info("would send renewal reminder",
				"subscription_id", sub.ID,
				"organization_id", sub.OrganizationID,
				"days_until_expiry", days,
			)
		}
	}

	return nil
}
