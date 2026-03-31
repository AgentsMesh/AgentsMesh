package billing

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// Features represents plan features as JSONB
type Features map[string]interface{}

// Scan implements sql.Scanner for Features
func (f *Features) Scan(value interface{}) error {
	if value == nil {
		*f = nil
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
	return json.Unmarshal(bytes, f)
}

// Value implements driver.Valuer for Features
func (f Features) Value() (driver.Value, error) {
	if f == nil {
		return nil, nil
	}
	return json.Marshal(f)
}

// SubscriptionPlan represents a subscription plan
type SubscriptionPlan struct {
	ID          int64  `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"size:50;not null;uniqueIndex" json:"name"`
	DisplayName string `gorm:"size:100;not null" json:"display_name"`

	PricePerSeatMonthly float64 `gorm:"type:decimal(10,2);not null;default:0" json:"price_per_seat_monthly"`
	PricePerSeatYearly  float64 `gorm:"type:decimal(10,2);not null;default:0" json:"price_per_seat_yearly"`
	IncludedPodMinutes  int     `gorm:"not null;default:0" json:"included_pod_minutes"`
	PricePerExtraMinute float64 `gorm:"type:decimal(10,4);not null;default:0" json:"price_per_extra_minute"`

	MaxUsers          int `gorm:"not null" json:"max_users"`
	MaxRunners        int `gorm:"not null" json:"max_runners"`
	MaxConcurrentPods int `gorm:"not null;default:0" json:"max_concurrent_pods"`
	MaxRepositories   int `gorm:"not null" json:"max_repositories"`

	Features Features `gorm:"type:jsonb;not null;default:'{}'" json:"features"`

	// Stripe Price IDs
	StripePriceIDMonthly *string `gorm:"size:255" json:"stripe_price_id_monthly,omitempty"`
	StripePriceIDYearly  *string `gorm:"size:255" json:"stripe_price_id_yearly,omitempty"`

	IsActive bool `gorm:"not null;default:true" json:"is_active"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
}

// GetPrice returns the price per seat for the given billing cycle
func (p *SubscriptionPlan) GetPrice(billingCycle string) float64 {
	if billingCycle == BillingCycleYearly {
		return p.PricePerSeatYearly
	}
	return p.PricePerSeatMonthly
}

func (SubscriptionPlan) TableName() string {
	return "subscription_plans"
}
