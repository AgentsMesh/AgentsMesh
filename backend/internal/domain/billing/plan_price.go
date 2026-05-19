package billing

import "time"

type PlanPrice struct {
	ID       int64  `gorm:"primaryKey" json:"id"`
	PlanID   int64  `gorm:"not null" json:"plan_id"`
	Currency string `gorm:"size:3;not null" json:"currency"`

	PriceMonthly float64 `gorm:"type:decimal(10,2);not null" json:"price_monthly"`
	PriceYearly  float64 `gorm:"type:decimal(10,2);not null" json:"price_yearly"`

	StripePriceIDMonthly *string `gorm:"size:255" json:"stripe_price_id_monthly,omitempty"`
	StripePriceIDYearly  *string `gorm:"size:255" json:"stripe_price_id_yearly,omitempty"`

	LemonSqueezyVariantIDMonthly *string `gorm:"column:lemonsqueezy_variant_id_monthly;size:255" json:"lemonsqueezy_variant_id_monthly,omitempty"`
	LemonSqueezyVariantIDYearly  *string `gorm:"column:lemonsqueezy_variant_id_yearly;size:255" json:"lemonsqueezy_variant_id_yearly,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	Plan *SubscriptionPlan `gorm:"foreignKey:PlanID" json:"plan,omitempty"`
}

func (PlanPrice) TableName() string {
	return "plan_prices"
}

func (p *PlanPrice) GetPrice(billingCycle string) float64 {
	if billingCycle == BillingCycleYearly {
		return p.PriceYearly
	}
	return p.PriceMonthly
}
