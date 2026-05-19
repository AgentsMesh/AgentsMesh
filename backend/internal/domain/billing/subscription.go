package billing

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type CustomQuotas map[string]interface{}

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

func (cq CustomQuotas) Value() (driver.Value, error) {
	if cq == nil {
		return nil, nil
	}
	return json.Marshal(cq)
}

type Subscription struct {
	ID             int64 `gorm:"primaryKey" json:"id"`
	OrganizationID int64 `gorm:"not null;uniqueIndex" json:"organization_id"`
	PlanID         int64 `gorm:"not null" json:"plan_id"`

	Status       string `gorm:"size:50;not null;default:'active'" json:"status"`
	BillingCycle string `gorm:"size:20;not null;default:'monthly'" json:"billing_cycle"`

	CurrentPeriodStart time.Time `gorm:"not null" json:"current_period_start"`
	CurrentPeriodEnd   time.Time `gorm:"not null" json:"current_period_end"`

	PaymentProvider *string `gorm:"size:50" json:"payment_provider,omitempty"`
	PaymentMethod   *string `gorm:"size:50" json:"payment_method,omitempty"`
	AutoRenew       bool    `gorm:"not null;default:false" json:"auto_renew"`
	SeatCount       int     `gorm:"not null;default:1" json:"seat_count"`

	StripeCustomerID     *string `gorm:"size:255" json:"stripe_customer_id,omitempty"`
	StripeSubscriptionID *string `gorm:"size:255" json:"stripe_subscription_id,omitempty"`

	AlipayAgreementNo *string `gorm:"size:255" json:"alipay_agreement_no,omitempty"`

	WeChatContractID *string `gorm:"column:wechat_contract_id;size:255" json:"wechat_contract_id,omitempty"`

	LemonSqueezyCustomerID     *string `gorm:"column:lemonsqueezy_customer_id;size:255" json:"lemonsqueezy_customer_id,omitempty"`
	LemonSqueezySubscriptionID *string `gorm:"column:lemonsqueezy_subscription_id;size:255" json:"lemonsqueezy_subscription_id,omitempty"`

	CanceledAt        *time.Time `json:"canceled_at,omitempty"`
	CancelAtPeriodEnd bool       `gorm:"not null;default:false" json:"cancel_at_period_end"`

	FrozenAt *time.Time `json:"frozen_at,omitempty"`

	DowngradeToPlan  *string `gorm:"size:50" json:"downgrade_to_plan,omitempty"`
	NextBillingCycle *string `gorm:"size:20" json:"next_billing_cycle,omitempty"`

	CustomQuotas CustomQuotas `gorm:"type:jsonb" json:"custom_quotas,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	Plan *SubscriptionPlan `gorm:"foreignKey:PlanID" json:"plan,omitempty"`
}

func (Subscription) TableName() string {
	return "subscriptions"
}

func (s *Subscription) IsFrozen() bool {
	return s.Status == SubscriptionStatusFrozen || s.FrozenAt != nil
}

func (s *Subscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive && s.FrozenAt == nil
}

func (s *Subscription) CanAddSeats(plan *SubscriptionPlan) bool {
	if plan == nil {
		plan = s.Plan
	}
	return plan != nil && plan.Name != PlanBased
}

func (s *Subscription) IsTrialing() bool {
	return s.Status == SubscriptionStatusTrialing
}

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

func (s *Subscription) GetAvailableSeats(usedSeats int) int {
	return s.SeatCount - usedSeats
}
