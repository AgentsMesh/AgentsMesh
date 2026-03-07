package billing

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// ===========================================
// Payment Order Operations
// ===========================================

// CreatePaymentOrder creates a new payment order
func (s *Service) CreatePaymentOrder(ctx context.Context, order *billing.PaymentOrder) error {
	return s.db.WithContext(ctx).Create(order).Error
}

// GetPaymentOrderByNo returns a payment order by order number
func (s *Service) GetPaymentOrderByNo(ctx context.Context, orderNo string) (*billing.PaymentOrder, error) {
	var order billing.PaymentOrder
	if err := s.db.WithContext(ctx).Where("order_no = ?", orderNo).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return &order, nil
}

// GetPaymentOrderByExternalNo returns a payment order by external order number
func (s *Service) GetPaymentOrderByExternalNo(ctx context.Context, externalNo string) (*billing.PaymentOrder, error) {
	var order billing.PaymentOrder
	if err := s.db.WithContext(ctx).Where("external_order_no = ?", externalNo).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return &order, nil
}

// UpdatePaymentOrderStatus updates the status of a payment order
func (s *Service) UpdatePaymentOrderStatus(ctx context.Context, orderNo string, status string, failureReason *string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if failureReason != nil {
		updates["failure_reason"] = *failureReason
	}
	if status == billing.OrderStatusSucceeded {
		now := time.Now()
		updates["paid_at"] = &now
	}

	return s.db.WithContext(ctx).Model(&billing.PaymentOrder{}).
		Where("order_no = ?", orderNo).
		Updates(updates).Error
}

// ===========================================
// Payment Transaction Operations
// ===========================================

// CreatePaymentTransaction creates a new payment transaction
func (s *Service) CreatePaymentTransaction(ctx context.Context, tx *billing.PaymentTransaction) error {
	return s.db.WithContext(ctx).Create(tx).Error
}

// ===========================================
// Invoice Operations
// ===========================================

// CreateInvoice creates a new invoice
func (s *Service) CreateInvoice(ctx context.Context, invoice *billing.Invoice) error {
	return s.db.WithContext(ctx).Create(invoice).Error
}

// GetInvoicesByOrg returns all invoices for an organization
func (s *Service) GetInvoicesByOrg(ctx context.Context, orgID int64, limit, offset int) ([]*billing.Invoice, error) {
	var invoices []*billing.Invoice
	query := s.db.WithContext(ctx).Where("organization_id = ?", orgID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	if err := query.Find(&invoices).Error; err != nil {
		return nil, err
	}
	return invoices, nil
}
