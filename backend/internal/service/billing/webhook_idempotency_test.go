package billing

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
	"github.com/anthropics/agentsmesh/backend/internal/service/payment"
)

// ===========================================
// Idempotency and Helper Functions Tests
// ===========================================

func TestWebhookIdempotency(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(newTestRepo(db), "")
	ctx := context.Background()

	seedTestPlan(t, db)
	plan, _ := service.GetPlan(ctx, "based")

	order := &billing.PaymentOrder{
		OrganizationID:  1,
		OrderNo:         "ORD-IDEMPOTENT",
		OrderType:       billing.OrderTypeSubscription,
		PlanID:          &plan.ID,
		BillingCycle:    billing.BillingCycleMonthly,
		Seats:           1,
		Amount:          0,
		ActualAmount:    0,
		PaymentProvider: billing.PaymentProviderStripe,
		Status:          billing.OrderStatusPending,
		CreatedByID:     1,
	}
	service.CreatePaymentOrder(ctx, order)

	c, _ := createTestGinContext()
	event := &payment.WebhookEvent{
		EventID:   "evt_idempotent_test",
		EventType: "checkout.session.completed",
		OrderNo:   "ORD-IDEMPOTENT",
		Amount:    0,
		Currency:  "USD",
		Status:    billing.OrderStatusSucceeded,
	}

	err := service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	// Second call should be idempotent
	err = service.HandlePaymentSucceeded(c, event)
	if err != nil {
		t.Errorf("expected idempotent return (nil), got error: %v", err)
	}
}

func TestIsDuplicateKeyError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected bool
	}{
		{"PostgreSQL duplicate", "duplicate key value violates", true},
		{"SQLite duplicate", "UNIQUE constraint failed: table.column", true},
		{"MySQL duplicate", "Duplicate entry for key PRIMARY", true},
		{"Other error", "connection refused", false},
		{"Nil error", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.errMsg != "" {
				err = fmt.Errorf("%s", tt.errMsg)
			}
			result := isDuplicateKeyError(err)
			if result != tt.expected {
				t.Errorf("isDuplicateKeyError(%q) = %v, want %v", tt.errMsg, result, tt.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello world", "foo", false},
		{"hello", "hello", true},
		{"hello", "hello world", false},
		{"", "", true},
		{"hello", "", true},
		{"", "hello", false},
	}

	for _, tt := range tests {
		name := fmt.Sprintf("%s_in_%s", tt.substr, tt.s)
		if name == "_in_" {
			name = "empty_strings"
		}
		t.Run(name, func(t *testing.T) {
			result := strings.Contains(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("strings.Contains(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}
