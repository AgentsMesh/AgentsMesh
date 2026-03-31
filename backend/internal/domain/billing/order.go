package billing

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// OrderMetadata represents optional order metadata
type OrderMetadata map[string]interface{}

// Scan implements sql.Scanner for OrderMetadata
func (om *OrderMetadata) Scan(value interface{}) error {
	if value == nil {
		*om = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("unsupported type for Scan")
	}
	return json.Unmarshal(bytes, om)
}

// Value implements driver.Valuer for OrderMetadata
func (om OrderMetadata) Value() (driver.Value, error) {
	if om == nil {
		return nil, nil
	}
	return json.Marshal(om)
}

// PaymentOrder represents a payment order
type PaymentOrder struct {
	ID             int64 `gorm:"primaryKey" json:"id"`
	OrganizationID int64 `gorm:"not null;index" json:"organization_id"`

	// Order identification
	OrderNo         string  `gorm:"size:64;not null;uniqueIndex" json:"order_no"`
	ExternalOrderNo *string `gorm:"size:255" json:"external_order_no,omitempty"`

	// Order type and relation
	OrderType    string `gorm:"size:50;not null" json:"order_type"`
	PlanID       *int64 `json:"plan_id,omitempty"`
	BillingCycle string `gorm:"size:20" json:"billing_cycle,omitempty"`
	Seats        int    `gorm:"default:1" json:"seats"`

	// Amount information
	Currency       string  `gorm:"size:10;not null;default:'USD'" json:"currency"`
	Amount         float64 `gorm:"type:decimal(10,2);not null" json:"amount"`
	DiscountAmount float64 `gorm:"type:decimal(10,2);default:0" json:"discount_amount"`
	ActualAmount   float64 `gorm:"type:decimal(10,2);not null" json:"actual_amount"`

	// Payment information
	PaymentProvider string  `gorm:"size:50;not null" json:"payment_provider"`
	PaymentMethod   *string `gorm:"size:50" json:"payment_method,omitempty"`

	// Status
	Status string `gorm:"size:50;not null;default:'pending'" json:"status"`

	// Metadata
	Metadata      OrderMetadata `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`
	FailureReason *string       `gorm:"type:text" json:"failure_reason,omitempty"`

	// Idempotency
	IdempotencyKey *string `gorm:"size:64;uniqueIndex" json:"idempotency_key,omitempty"`

	// Timestamps
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
	CreatedAt   time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"not null;default:now()" json:"updated_at"`
	CreatedByID int64      `gorm:"not null" json:"created_by_id"`

	// Associations
	Plan *SubscriptionPlan `gorm:"foreignKey:PlanID" json:"plan,omitempty"`
}

func (PaymentOrder) TableName() string {
	return "payment_orders"
}

// IsPending returns true if the order is pending payment
func (o *PaymentOrder) IsPending() bool {
	return o.Status == OrderStatusPending
}

// IsSucceeded returns true if the order is paid
func (o *PaymentOrder) IsSucceeded() bool {
	return o.Status == OrderStatusSucceeded
}

// IsExpired returns true if the order has expired
func (o *PaymentOrder) IsExpired() bool {
	return o.ExpiresAt != nil && time.Now().After(*o.ExpiresAt)
}
