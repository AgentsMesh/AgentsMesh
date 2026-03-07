package infra

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// --- PaymentOrder ---

func (r *billingRepository) CreatePaymentOrder(ctx context.Context, order *billing.PaymentOrder) error {
	return r.db.WithContext(ctx).Create(order).Error
}

func (r *billingRepository) GetPaymentOrderByNo(ctx context.Context, orderNo string) (*billing.PaymentOrder, error) {
	var order billing.PaymentOrder
	if err := r.db.WithContext(ctx).Where("order_no = ?", orderNo).First(&order).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (r *billingRepository) GetPaymentOrderByExternalNo(ctx context.Context, externalNo string) (*billing.PaymentOrder, error) {
	var order billing.PaymentOrder
	if err := r.db.WithContext(ctx).Where("external_order_no = ?", externalNo).First(&order).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (r *billingRepository) UpdatePaymentOrderStatus(ctx context.Context, orderNo string, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&billing.PaymentOrder{}).Where("order_no = ?", orderNo).Updates(updates).Error
}

// --- PaymentTransaction ---

func (r *billingRepository) CreatePaymentTransaction(ctx context.Context, tx *billing.PaymentTransaction) error {
	return r.db.WithContext(ctx).Create(tx).Error
}

// --- Invoice ---

func (r *billingRepository) CreateInvoice(ctx context.Context, invoice *billing.Invoice) error {
	return r.db.WithContext(ctx).Create(invoice).Error
}

func (r *billingRepository) ListInvoicesByOrg(ctx context.Context, orgID int64, limit, offset int) ([]*billing.Invoice, error) {
	var invoices []*billing.Invoice
	query := r.db.WithContext(ctx).Where("organization_id = ?", orgID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Find(&invoices).Error; err != nil {
		return nil, err
	}
	return invoices, nil
}

// --- UsageRecord ---

func (r *billingRepository) CreateUsageRecord(ctx context.Context, record *billing.UsageRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *billingRepository) SumUsageByPeriod(ctx context.Context, orgID int64, usageType string, periodStart, periodEnd time.Time) (float64, error) {
	var total float64
	if err := r.db.WithContext(ctx).Model(&billing.UsageRecord{}).
		Where("organization_id = ? AND usage_type = ? AND period_start >= ? AND period_end <= ?",
			orgID, usageType, periodStart, periodEnd).
		Select("COALESCE(SUM(quantity), 0)").
		Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (r *billingRepository) ListUsageHistory(ctx context.Context, orgID int64, usageType string, since time.Time) ([]*billing.UsageRecord, error) {
	var records []*billing.UsageRecord
	query := r.db.WithContext(ctx).Where("organization_id = ? AND period_start >= ?", orgID, since)
	if usageType != "" {
		query = query.Where("usage_type = ?", usageType)
	}
	if err := query.Order("period_start DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

// --- WebhookEvent ---

func (r *billingRepository) CreateWebhookEvent(ctx context.Context, event *billing.WebhookEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *billingRepository) DeleteWebhookEvent(ctx context.Context, eventID, provider string) error {
	return r.db.WithContext(ctx).
		Where("event_id = ? AND provider = ?", eventID, provider).
		Delete(&billing.WebhookEvent{}).Error
}
