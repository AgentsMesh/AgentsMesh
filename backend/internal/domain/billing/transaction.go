package billing

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type RawPayload map[string]interface{}

func (rp *RawPayload) Scan(value interface{}) error {
	if value == nil {
		*rp = nil
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
	return json.Unmarshal(bytes, rp)
}

func (rp RawPayload) Value() (driver.Value, error) {
	if rp == nil {
		return nil, nil
	}
	return json.Marshal(rp)
}

type PaymentTransaction struct {
	ID             int64 `gorm:"primaryKey" json:"id"`
	PaymentOrderID int64 `gorm:"not null;index" json:"payment_order_id"`

	TransactionType       string  `gorm:"size:50;not null" json:"transaction_type"`
	ExternalTransactionID *string `gorm:"size:255" json:"external_transaction_id,omitempty"`

	Amount   float64 `gorm:"type:decimal(10,2);not null" json:"amount"`
	Currency string  `gorm:"size:10;not null;default:'USD'" json:"currency"`

	Status string `gorm:"size:50;not null" json:"status"`

	WebhookEventID   *string    `gorm:"size:255" json:"webhook_event_id,omitempty"`
	WebhookEventType *string    `gorm:"size:100" json:"webhook_event_type,omitempty"`
	RawPayload       RawPayload `gorm:"type:jsonb" json:"raw_payload,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`

	PaymentOrder *PaymentOrder `gorm:"foreignKey:PaymentOrderID" json:"payment_order,omitempty"`
}

func (PaymentTransaction) TableName() string {
	return "payment_transactions"
}
