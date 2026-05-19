package runner

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Certificate represents a certificate issued to a Runner for mTLS authentication.
type Certificate struct {
	ID               int64      `gorm:"primaryKey" json:"id"`
	RunnerID         int64      `gorm:"not null;index" json:"runner_id"`
	SerialNumber     string     `gorm:"size:64;uniqueIndex;not null" json:"serial_number"`
	Fingerprint      string     `gorm:"size:128;not null" json:"fingerprint"`
	IssuedAt         time.Time  `gorm:"not null" json:"issued_at"`
	ExpiresAt        time.Time  `gorm:"not null" json:"expires_at"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	RevocationReason *string    `gorm:"size:255" json:"revocation_reason,omitempty"`
	CreatedAt        time.Time  `gorm:"not null;default:now()" json:"created_at"`
}

func (Certificate) TableName() string {
	return "runner_certificates"
}

func (c *Certificate) IsRevoked() bool {
	return c.RevokedAt != nil
}

func (c *Certificate) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

func (c *Certificate) IsValid() bool {
	return !c.IsRevoked() && !c.IsExpired()
}

type PendingAuth struct {
	ID             int64      `gorm:"primaryKey" json:"id"`
	AuthKey        string     `gorm:"size:64;uniqueIndex;not null" json:"auth_key"`
	MachineKey     string     `gorm:"size:128;not null" json:"machine_key"`
	NodeID         *string    `gorm:"size:255" json:"node_id,omitempty"`
	Labels         Labels     `gorm:"type:jsonb" json:"labels,omitempty"`
	Authorized     bool       `gorm:"not null;default:false" json:"authorized"`
	OrganizationID *int64     `json:"organization_id,omitempty"`
	RunnerID       *int64     `json:"runner_id,omitempty"`
	ExpiresAt      time.Time  `gorm:"not null" json:"expires_at"`
	CreatedAt      time.Time  `gorm:"not null;default:now()" json:"created_at"`
}

func (PendingAuth) TableName() string {
	return "runner_pending_auths"
}

func (p *PendingAuth) IsExpired() bool {
	return time.Now().After(p.ExpiresAt)
}

type Labels map[string]string

func (l *Labels) Scan(value interface{}) error {
	if value == nil {
		*l = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("unsupported type for Scan")
	}
	return json.Unmarshal(bytes, l)
}

func (l Labels) Value() (driver.Value, error) {
	if l == nil {
		return nil, nil
	}
	return json.Marshal(l)
}

type GRPCRegistrationToken struct {
	ID             int64      `gorm:"primaryKey" json:"id"`
	TokenHash      string     `gorm:"size:128;uniqueIndex;not null" json:"-"` // Never expose hash
	OrganizationID int64      `gorm:"not null;index" json:"organization_id"`
	Name           *string    `gorm:"size:255" json:"name,omitempty"`
	Labels         Labels     `gorm:"type:jsonb" json:"labels,omitempty"`
	SingleUse      bool       `gorm:"not null;default:true" json:"single_use"`
	MaxUses        int        `gorm:"not null;default:1" json:"max_uses"`
	UsedCount      int        `gorm:"not null;default:0" json:"used_count"`
	ExpiresAt      time.Time  `gorm:"not null" json:"expires_at"`
	CreatedBy      *int64     `json:"created_by,omitempty"`
	CreatedAt      time.Time  `gorm:"not null;default:now()" json:"created_at"`
}

func (GRPCRegistrationToken) TableName() string {
	return "runner_grpc_registration_tokens"
}

func (t *GRPCRegistrationToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

func (t *GRPCRegistrationToken) IsExhausted() bool {
	return t.UsedCount >= t.MaxUses
}

func (t *GRPCRegistrationToken) IsValid() bool {
	return !t.IsExpired() && !t.IsExhausted()
}

// ReactivationToken represents a one-time token for reactivating Runners with expired certificates.
// Generated via Web UI, valid for a short time (e.g., 10 minutes).
type ReactivationToken struct {
	ID        int64      `gorm:"primaryKey" json:"id"`
	TokenHash string     `gorm:"size:128;uniqueIndex;not null" json:"-"` // Never expose hash
	RunnerID  int64      `gorm:"not null;index" json:"runner_id"`
	ExpiresAt time.Time  `gorm:"not null" json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedBy *int64     `json:"created_by,omitempty"`
	CreatedAt time.Time  `gorm:"not null;default:now()" json:"created_at"`
}

func (ReactivationToken) TableName() string {
	return "runner_reactivation_tokens"
}

func (t *ReactivationToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

func (t *ReactivationToken) IsUsed() bool {
	return t.UsedAt != nil
}

func (t *ReactivationToken) IsValid() bool {
	return !t.IsExpired() && !t.IsUsed()
}
