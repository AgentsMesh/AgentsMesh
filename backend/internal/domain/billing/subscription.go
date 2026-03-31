package billing

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// CustomQuotas represents custom quota overrides
type CustomQuotas map[string]interface{}

// Scan implements sql.Scanner for CustomQuotas
func (cq *CustomQuotas) Scan(value interface{}) error {
	if value == nil {
		*cq = nil
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
	return json.Unmarshal(bytes, cq)
}

// Value implements driver.Valuer for CustomQuotas
func (cq CustomQuotas) Value() (driver.Value, error) {
	if cq == nil {
		return nil, nil
	}
	return json.Marshal(cq)
}

// Subscription represents an organization's subscription
type Subscription struct {
	ID             int64 `gorm:"primaryKey" json:"id"`
	OrganizationID int64 `gorm:"not null;uniqueIndex" json:"organization_id"`
	PlanID         int64 `gorm:"not null" json:"plan_id"`

	Status       string `gorm:"size:50;not null;default:'active'" json:"status"`
	BillingCycle string `gorm:"size:20;not null;default:'monthly'" json:"billing_cycle"`

	CurrentPeriodStart time.Time `gorm:"not null" json:"current_period_start"`
	CurrentPeriodEnd   time.Time `gorm:"not null" json:"current_period_end"`

	// Payment information
	PaymentProvider *string `gorm:"size:50" json:"payment_provider,omitempty"`
	PaymentMethod   *string `gorm:"size:50" json:"payment_method,omitempty"`
	AutoRenew       bool    `gorm:"not null;default:false" json:"auto_renew"`
	SeatCount       int     `gorm:"not null;default:1" json:"seat_count"`

	// Stripe integration
	StripeCustomerID     *string `gorm:"size:255" json:"stripe_customer_id,omitempty"`
	StripeSubscriptionID *string `gorm:"size:255" json:"stripe_subscription_id,omitempty"`

	// Alipay integration
	AlipayAgreementNo *string `gorm:"size:255" json:"alipay_agreement_no,omitempty"`

	// WeChat integration
	WeChatContractID *string `gorm:"column:wechat_contract_id;size:255" json:"wechat_contract_id,omitempty"`

	// LemonSqueezy integration
	LemonSqueezyCustomerID     *string `gorm:"column:lemonsqueezy_customer_id;size:255" json:"lemonsqueezy_customer_id,omitempty"`
	LemonSqueezySubscriptionID *string `gorm:"column:lemonsqueezy_subscription_id;size:255" json:"lemonsqueezy_subscription_id,omitempty"`

	// Cancellation
	CanceledAt        *time.Time `json:"canceled_at,omitempty"`
	CancelAtPeriodEnd bool       `gorm:"not null;default:false" json:"cancel_at_period_end"`

	// Frozen state (non-payment)
	FrozenAt *time.Time `json:"frozen_at,omitempty"`

	// Pending changes (take effect at period end)
	DowngradeToPlan  *string `gorm:"size:50" json:"downgrade_to_plan,omitempty"`
	NextBillingCycle *string `gorm:"size:20" json:"next_billing_cycle,omitempty"`

	CustomQuotas CustomQuotas `gorm:"type:jsonb" json:"custom_quotas,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Associations
	Plan *SubscriptionPlan `gorm:"foreignKey:PlanID" json:"plan,omitempty"`
}

func (Subscription) TableName() string {
	return "subscriptions"
}

// IsFrozen returns true if the subscription is frozen
func (s *Subscription) IsFrozen() bool {
	return s.Status == SubscriptionStatusFrozen || s.FrozenAt != nil
}

// IsActive returns true if the subscription is active and not frozen
func (s *Subscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive && s.FrozenAt == nil
}

// CanAddSeats returns true if seats can be added (not Based plan)
func (s *Subscription) CanAddSeats(plan *SubscriptionPlan) bool {
	if plan == nil {
		plan = s.Plan
	}
	// Based plan has fixed 1 seat, cannot add more
	return plan != nil && plan.Name != PlanBased
}

// IsTrialing returns true if the subscription is in trial period
func (s *Subscription) IsTrialing() bool {
	return s.Status == SubscriptionStatusTrialing
}

// GetRemainingTrialDays returns the number of days remaining in the trial period
func (s *Subscription) GetRemainingTrialDays() int {
	if s.Status != SubscriptionStatusTrialing {
		return 0
	}
	remaining := time.Until(s.CurrentPeriodEnd).Hours() / 24
	if remaining < 0 {
		return 0
	}
	return int(remaining)
}

// GetAvailableSeats returns the number of available (unused) seats
func (s *Subscription) GetAvailableSeats(usedSeats int) int {
	return s.SeatCount - usedSeats
}
