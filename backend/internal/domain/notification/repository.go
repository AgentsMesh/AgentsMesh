package notification

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// ChannelsJSON is a map[string]bool stored as JSONB in PostgreSQL.
type ChannelsJSON map[string]bool

// Scan implements sql.Scanner for reading JSONB from the database.
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

// Value implements driver.Valuer for writing JSONB to the database.
func (c ChannelsJSON) Value() (driver.Value, error) {
	if c == nil {
		return nil, nil
	}
	return json.Marshal(c)
}

// PreferenceRecord is the GORM model for notification_preferences table
type PreferenceRecord struct {
	UserID   int64        `gorm:"primaryKey"`
	Source   string       `gorm:"primaryKey;size:50"`
	EntityID string       `gorm:"size:200;default:''"` // empty string = global preference
	IsMuted  bool         `gorm:"default:false"`
	Channels ChannelsJSON `gorm:"type:jsonb;default:'{\"toast\":true,\"browser\":true}'"`
}

// TableName specifies the database table name
func (PreferenceRecord) TableName() string { return "notification_preferences" }

// PreferenceRepository defines data access for notification preferences
type PreferenceRepository interface {
	// GetPreference returns the preference for a specific (user, source, entityID).
	// Returns nil if not found.
	GetPreference(ctx context.Context, userID int64, source string, entityID string) (*PreferenceRecord, error)

	// SetPreference creates or updates a preference record
	SetPreference(ctx context.Context, record *PreferenceRecord) error

	// ListPreferences returns all preferences for a user
	ListPreferences(ctx context.Context, userID int64) ([]PreferenceRecord, error)

	// DeletePreference removes a preference record
	DeletePreference(ctx context.Context, userID int64, source string, entityID string) error
}
