package notification

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type ChannelsJSON map[string]bool

func (c *ChannelsJSON) Scan(src interface{}) error {
	if src == nil {
		*c = nil
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("ChannelsJSON.Scan: unsupported type %T", src)
	}
	return json.Unmarshal(data, c)
}

func (c ChannelsJSON) Value() (driver.Value, error) {
	if c == nil {
		return nil, nil
	}
	return json.Marshal(c)
}

type PreferenceRecord struct {
	UserID   int64        `gorm:"primaryKey"`
	Source   string       `gorm:"primaryKey;size:50"`
	EntityID string       `gorm:"size:200;default:''"` // empty string = global preference
	IsMuted  bool         `gorm:"default:false"`
	Channels ChannelsJSON `gorm:"type:jsonb;default:'{\"toast\":true,\"browser\":true}'"`
}

func (PreferenceRecord) TableName() string { return "notification_preferences" }

type PreferenceRepository interface {
	// GetPreference returns the preference for a specific (user, source, entityID).
	// Returns nil if not found.
	GetPreference(ctx context.Context, userID int64, source string, entityID string) (*PreferenceRecord, error)

	SetPreference(ctx context.Context, record *PreferenceRecord) error

	ListPreferences(ctx context.Context, userID int64) ([]PreferenceRecord, error)

	DeletePreference(ctx context.Context, userID int64, source string, entityID string) error
}
