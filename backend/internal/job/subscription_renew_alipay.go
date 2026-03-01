package job

import (
	"context"
	"fmt"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/payment/types"
)

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
