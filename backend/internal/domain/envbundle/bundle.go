package envbundle

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// BundleData is a JSONB-backed string KV map. Whether values are stored
// plaintext or encrypted depends on Kind — the service layer applies the
// appropriate transform on write and reverses it on read (see IsEncryptedKind).
// Scan/Value only marshal JSON; encryption is not the storage layer's concern.
type BundleData map[string]string

// Scan implements sql.Scanner for JSONB columns.
func (d *BundleData) Scan(value interface{}) error {
	if value == nil {
		*d = nil
		return nil
	}
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return errors.New("envbundle.BundleData: cannot scan non-bytes/string value")
	}
	return json.Unmarshal(raw, d)
}

// Value implements driver.Valuer for JSONB columns.
func (d BundleData) Value() (driver.Value, error) {
	if d == nil {
		return nil, nil
	}
	return json.Marshal(d)
}

// EnvBundle is a named, owner-scoped set of environment variables an AgentFile
// can reference via `USE_ENV_BUNDLE "name"`. credential becomes one kind among
// others (runtime, shared, …) sharing the same storage + lookup machinery.
type EnvBundle struct {
	ID          int64      `gorm:"primaryKey" json:"id"`
	OwnerScope  string     `gorm:"column:owner_scope;size:16;not null;index" json:"owner_scope"`
	OwnerID     int64      `gorm:"column:owner_id;not null;index" json:"owner_id"`
	AgentSlug   *string    `gorm:"column:agent_slug;size:100;index" json:"agent_slug,omitempty"`
	Name        string     `gorm:"size:100;not null" json:"name"`
	Description *string    `gorm:"type:text" json:"description,omitempty"`
	Kind        string     `gorm:"size:32;not null;index" json:"kind"`
	KindPrimary bool       `gorm:"column:kind_primary;not null;default:false" json:"kind_primary"`
	Data        BundleData `gorm:"type:jsonb" json:"-"`
	IsActive    bool       `gorm:"not null;default:true" json:"is_active"`
	CreatedAt   time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"not null;default:now()" json:"updated_at"`
}

func (EnvBundle) TableName() string {
	return "env_bundles"
}

// Response is the safe-to-serialize API shape. ConfiguredFields holds the
// names of secret keys (never their values); ConfiguredValues holds the
// plaintext of non-secret keys. The split is per-key (see IsNonSecretKey), so
// an encrypted credential bundle can surface a non-secret field's value while
// keeping API keys/tokens hidden. The two slots never share a key.
type Response struct {
	ID          int64   `json:"id"`
	OwnerScope  string  `json:"owner_scope"`
	OwnerID     int64   `json:"owner_id"`
	AgentSlug   *string `json:"agent_slug,omitempty"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Kind        string  `json:"kind"`
	KindPrimary bool    `json:"kind_primary"`
	IsActive    bool    `json:"is_active"`

	ConfiguredFields []string          `json:"configured_fields,omitempty"`
	ConfiguredValues map[string]string `json:"configured_values,omitempty"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ToResponse builds the API response shell. The service layer fills exactly
// one of ConfiguredFields / ConfiguredValues based on whether the kind is
// encrypted — see Service.ResponseWithValues.
func (b *EnvBundle) ToResponse() *Response {
	return &Response{
		ID:          b.ID,
		OwnerScope:  b.OwnerScope,
		OwnerID:     b.OwnerID,
		AgentSlug:   b.AgentSlug,
		Name:        b.Name,
		Description: b.Description,
		Kind:        b.Kind,
		KindPrimary: b.KindPrimary,
		IsActive:    b.IsActive,
		CreatedAt:   b.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   b.UpdatedAt.Format(time.RFC3339),
	}
}

// GroupedByAgent groups bundles by agent_slug for list-by-agent responses.
type GroupedByAgent struct {
	AgentSlug string      `json:"agent_slug"`
	AgentName string      `json:"agent_name,omitempty"`
	Bundles   []*Response `json:"bundles"`
}

// ListResponse is the top-level list response shape used by the personal
// "Env Bundles" settings page.
type ListResponse struct {
	Items []*Response `json:"items"`
}
