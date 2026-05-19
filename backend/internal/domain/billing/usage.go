package billing

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type UsageMetadata map[string]interface{}

func (um *UsageMetadata) Scan(value interface{}) error {
	if value == nil {
		*um = nil
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
	return json.Unmarshal(bytes, um)
}

func (um UsageMetadata) Value() (driver.Value, error) {
	if um == nil {
		return nil, nil
	}
	return json.Marshal(um)
}

type UsageRecord struct {
	ID             int64 `gorm:"primaryKey" json:"id"`
	OrganizationID int64 `gorm:"not null;index" json:"organization_id"`

	UsageType string  `gorm:"size:50;not null;index" json:"usage_type"`
	Quantity  float64 `gorm:"type:decimal(10,2);not null" json:"quantity"`

	PeriodStart time.Time `gorm:"not null" json:"period_start"`
	PeriodEnd   time.Time `gorm:"not null" json:"period_end"`

	Metadata UsageMetadata `gorm:"type:jsonb" json:"metadata,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
}

func (UsageRecord) TableName() string {
	return "usage_records"
}
