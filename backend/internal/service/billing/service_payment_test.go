package billing

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// ===========================================
// Payment Order Tests
// ===========================================

func TestCreatePaymentOrder(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-001",
		OrderType:       billing.OrderTypeSubscription,
		Amount:          19.99,
		ActualAmount:    19.99,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}

	err := service.CreatePaymentOrder(ctx, order)
	if err != nil {
		t.Fatalf("failed to create payment order: %v", err)
	}
	if order.ID == 0 {
		t.Error("expected order ID to be set")
	}
}

func TestGetPaymentOrderByNo(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-002",
		OrderType:       billing.OrderTypeSubscription,
		Amount:          19.99,
		ActualAmount:    19.99,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	result, err := service.GetPaymentOrderByNo(ctx, "ORD-002")
	if err != nil {
		t.Fatalf("failed to get payment order: %v", err)
	}
	if result.OrderNo != "ORD-002" {
		t.Errorf("expected order no 'ORD-002', got %s", result.OrderNo)
	}
}

func TestGetPaymentOrderByNoNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	_, err := service.GetPaymentOrderByNo(ctx, "nonexistent")
	if err != ErrOrderNotFound {
		t.Errorf("expected ErrOrderNotFound, got %v", err)
	}
}

func TestGetPaymentOrderByExternalNo(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	extNo := "ext_123"
	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-003",
		ExternalOrderNo: &extNo,
		OrderType:       billing.OrderTypeSubscription,
		Amount:          19.99,
		ActualAmount:    19.99,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	result, err := service.GetPaymentOrderByExternalNo(ctx, "ext_123")
	if err != nil {
		t.Fatalf("failed to get payment order: %v", err)
	}
	if result.OrderNo != "ORD-003" {
		t.Errorf("expected order no 'ORD-003', got %s", result.OrderNo)
	}
}

func TestUpdatePaymentOrderStatus(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-004",
		OrderType:       billing.OrderTypeSubscription,
		Amount:          19.99,
		ActualAmount:    19.99,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	err := service.UpdatePaymentOrderStatus(ctx, "ORD-004", billing.OrderStatusSucceeded, nil)
	if err != nil {
		t.Fatalf("failed to update order status: %v", err)
	}

	result, _ := service.GetPaymentOrderByNo(ctx, "ORD-004")
	if result.Status != billing.OrderStatusSucceeded {
		t.Errorf("expected status succeeded, got %s", result.Status)
	}
	if result.PaidAt == nil {
		t.Error("expected PaidAt to be set")
	}
}

func TestUpdatePaymentOrderStatusFailed(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-005",
		OrderType:       billing.OrderTypeSubscription,
		Amount:          19.99,
		ActualAmount:    19.99,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	reason := "Card declined"
	err := service.UpdatePaymentOrderStatus(ctx, "ORD-005", billing.OrderStatusFailed, &reason)
	if err != nil {
		t.Fatalf("failed to update order status: %v", err)
	}

	result, _ := service.GetPaymentOrderByNo(ctx, "ORD-005")
	if result.FailureReason == nil || *result.FailureReason != "Card declined" {
		t.Error("expected failure reason to be set")
	}
}

// ===========================================
// Invoice Tests
// ===========================================

func TestCreateInvoice(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	now := time.Now()
	invoice := &billing.Invoice{
		OrganizationID: 1,
		InvoiceNo:      "INV-001",
		Status:         billing.InvoiceStatusDraft,
		Subtotal:       19.99,
		Total:          19.99,
		PeriodStart:    now,
		PeriodEnd:      now.AddDate(0, 1, 0),
		LineItems:      billing.LineItems{}, // Initialize empty slice for SQLite compatibility
	}

	err := service.CreateInvoice(ctx, invoice)
	if err != nil {
		t.Fatalf("failed to create invoice: %v", err)
	}
	if invoice.ID == 0 {
		t.Error("expected invoice ID to be set")
	}
}

func TestGetInvoicesByOrg(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	now := time.Now()
	for i := 1; i <= 3; i++ {
		invoice := &billing.Invoice{
			OrganizationID: 1,
			InvoiceNo:      "INV-00" + string(rune('0'+i)),
			Status:         billing.InvoiceStatusPaid,
			Subtotal:       19.99,
			Total:          19.99,
			PeriodStart:    now,
			PeriodEnd:      now.AddDate(0, 1, 0),
			LineItems:      billing.LineItems{}, // Initialize for SQLite
		}
		service.CreateInvoice(ctx, invoice)
	}

	invoices, err := service.GetInvoicesByOrg(ctx, 1, 0, 0)
	if err != nil {
		t.Fatalf("failed to get invoices: %v", err)
	}
	if len(invoices) != 3 {
		t.Errorf("expected 3 invoices, got %d", len(invoices))
	}
}

func TestGetInvoicesByOrgWithPagination(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	now := time.Now()
	for i := 1; i <= 5; i++ {
		invoice := &billing.Invoice{
			OrganizationID: 1,
			InvoiceNo:      "INV-00" + string(rune('0'+i)),
			Status:         billing.InvoiceStatusPaid,
			Subtotal:       19.99,
			Total:          19.99,
			PeriodStart:    now,
			PeriodEnd:      now.AddDate(0, 1, 0),
			LineItems:      billing.LineItems{}, // Initialize for SQLite
		}
		service.CreateInvoice(ctx, invoice)
	}

	invoices, err := service.GetInvoicesByOrg(ctx, 1, 2, 0)
	if err != nil {
		t.Fatalf("failed to get invoices: %v", err)
	}
	if len(invoices) != 2 {
		t.Errorf("expected 2 invoices, got %d", len(invoices))
	}
}

// ===========================================
// Transaction Tests
// ===========================================

func TestCreatePaymentTransaction(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	// Create order first
	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-TX-001",
		OrderType:       billing.OrderTypeSubscription,
		Amount:          19.99,
		ActualAmount:    19.99,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	tx := &billing.PaymentTransaction{
		PaymentOrderID:  order.ID,
		TransactionType: billing.TransactionTypePayment,
		Amount:          19.99,
		Currency:        "USD",
		Status:          billing.TransactionStatusSucceeded,
	}

	err := service.CreatePaymentTransaction(ctx, tx)
	if err != nil {
		t.Fatalf("failed to create payment transaction: %v", err)
	}
	if tx.ID == 0 {
		t.Error("expected transaction ID to be set")
	}
}
