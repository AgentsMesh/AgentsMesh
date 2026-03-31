package billing

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// BillingAddress represents billing address as JSONB
type BillingAddress map[string]interface{}

// Scan implements sql.Scanner for BillingAddress
func (ba *BillingAddress) Scan(value interface{}) error {
	if value == nil {
		*ba = nil
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
	return json.Unmarshal(bytes, ba)
}

// Value implements driver.Valuer for BillingAddress
func (ba BillingAddress) Value() (driver.Value, error) {
	if ba == nil {
		return nil, nil
	}
	return json.Marshal(ba)
}

// LineItems represents invoice line items as JSONB
type LineItems []LineItem

// LineItem represents a single line item
type LineItem struct {
	Description string  `json:"description"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Amount      float64 `json:"amount"`
}

// Scan implements sql.Scanner for LineItems
func (li *LineItems) Scan(value interface{}) error {
	if value == nil {
		*li = nil
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
	return json.Unmarshal(bytes, li)
}

// Value implements driver.Valuer for LineItems
func (li LineItems) Value() (driver.Value, error) {
	if li == nil {
		return nil, nil
	}
	return json.Marshal(li)
}

// Invoice represents an invoice record
type Invoice struct {
	ID             int64  `gorm:"primaryKey" json:"id"`
	OrganizationID int64  `gorm:"not null;index" json:"organization_id"`
	PaymentOrderID *int64 `json:"payment_order_id,omitempty"`

	// Invoice information
	InvoiceNo string `gorm:"size:64;not null;uniqueIndex" json:"invoice_no"`
	Status    string `gorm:"size:50;not null;default:'draft'" json:"status"`

	// Amount
	Currency  string  `gorm:"size:10;not null;default:'USD'" json:"currency"`
	Subtotal  float64 `gorm:"type:decimal(10,2);not null" json:"subtotal"`
	TaxAmount float64 `gorm:"type:decimal(10,2);default:0" json:"tax_amount"`
	Total     float64 `gorm:"type:decimal(10,2);not null" json:"total"`

	// Billing information
	BillingName    *string        `gorm:"size:255" json:"billing_name,omitempty"`
	BillingEmail   *string        `gorm:"size:255" json:"billing_email,omitempty"`
	BillingAddress BillingAddress `gorm:"type:jsonb" json:"billing_address,omitempty"`

	// Invoice period
	PeriodStart time.Time `gorm:"not null" json:"period_start"`
	PeriodEnd   time.Time `gorm:"not null" json:"period_end"`

	// Line items
	LineItems LineItems `gorm:"type:jsonb;not null;default:'[]'" json:"line_items"`

	// PDF
	PDFURL *string `gorm:"type:text" json:"pdf_url,omitempty"`

	// Timestamps
	IssuedAt  *time.Time `json:"issued_at,omitempty"`
	DueAt     *time.Time `json:"due_at,omitempty"`
	PaidAt    *time.Time `json:"paid_at,omitempty"`
	CreatedAt time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time  `gorm:"not null;default:now()" json:"updated_at"`

	// Associations
	PaymentOrder *PaymentOrder `gorm:"foreignKey:PaymentOrderID" json:"payment_order,omitempty"`
}

func (Invoice) TableName() string {
	return "invoices"
}

// IsPaid returns true if the invoice is paid
func (i *Invoice) IsPaid() bool {
	return i.Status == InvoiceStatusPaid
}
