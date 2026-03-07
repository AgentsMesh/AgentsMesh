package billing

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// ===========================================
// Payment Order Operations
// ===========================================

// CreatePaymentOrder creates a new payment order
func (s *Service) CreatePaymentOrder(ctx context.Context, order *billing.PaymentOrder) error {
	return s.repo.CreatePaymentOrder(ctx, order)
}

// GetPaymentOrderByNo returns a payment order by order number
func (s *Service) GetPaymentOrderByNo(ctx context.Context, orderNo string) (*billing.PaymentOrder, error) {
	order, err := s.repo.GetPaymentOrderByNo(ctx, orderNo)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	return order, nil
}

// GetPaymentOrderByExternalNo returns a payment order by external order number
func (s *Service) GetPaymentOrderByExternalNo(ctx context.Context, externalNo string) (*billing.PaymentOrder, error) {
	order, err := s.repo.GetPaymentOrderByExternalNo(ctx, externalNo)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	return order, nil
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

	return s.repo.UpdatePaymentOrderStatus(ctx, orderNo, updates)
}

// ===========================================
// Payment Transaction Operations
// ===========================================

// CreatePaymentTransaction creates a new payment transaction
func (s *Service) CreatePaymentTransaction(ctx context.Context, tx *billing.PaymentTransaction) error {
	return s.repo.CreatePaymentTransaction(ctx, tx)
}

// ===========================================
// Invoice Operations
// ===========================================

// CreateInvoice creates a new invoice
func (s *Service) CreateInvoice(ctx context.Context, invoice *billing.Invoice) error {
	return s.repo.CreateInvoice(ctx, invoice)
}

// GetInvoicesByOrg returns all invoices for an organization
func (s *Service) GetInvoicesByOrg(ctx context.Context, orgID int64, limit, offset int) ([]*billing.Invoice, error) {
	return s.repo.ListInvoicesByOrg(ctx, orgID, limit, offset)
}
