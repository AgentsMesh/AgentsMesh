package runner

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// ==================== Certificate Management ====================

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

// IsRevoked returns true if the certificate has been revoked.
func (c *Certificate) IsRevoked() bool {
	return c.RevokedAt != nil
}

// IsExpired returns true if the certificate has expired.
func (c *Certificate) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// IsValid returns true if the certificate is valid (not revoked and not expired).
func (c *Certificate) IsValid() bool {
	return !c.IsRevoked() && !c.IsExpired()
}

// ==================== Pending Auth (Interactive Registration) ====================

// PendingAuth represents a pending interactive authorization request.
// Used for Tailscale-style registration where Runner generates a machine_key,
// gets an auth URL, and user authorizes in browser.
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

// IsExpired returns true if the auth request has expired.
func (p *PendingAuth) IsExpired() bool {
	return time.Now().After(p.ExpiresAt)
}

// Labels is a custom type for map[string]string that implements sql.Scanner and driver.Valuer.
type Labels map[string]string

// Scan implements sql.Scanner for Labels.
func (l *Labels) Scan(value interface{}) error {
	if value == nil {
		*l = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, l)
}

// Value implements driver.Valuer for Labels.
func (l Labels) Value() (driver.Value, error) {
	if l == nil {
		return nil, nil
	}
	return json.Marshal(l)
}

// ==================== Registration Token (Pre-generated Token) ====================

// GRPCRegistrationToken represents a pre-generated registration token.
// Created by org admins for automated/scripted Runner registration.
// Named with GRPC prefix to avoid conflict with existing RegistrationToken.
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

// IsExpired returns true if the token has expired.
func (t *GRPCRegistrationToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsExhausted returns true if the token has been used the maximum number of times.
func (t *GRPCRegistrationToken) IsExhausted() bool {
	return t.UsedCount >= t.MaxUses
}

// IsValid returns true if the token can still be used.
func (t *GRPCRegistrationToken) IsValid() bool {
	return !t.IsExpired() && !t.IsExhausted()
}

// ==================== Reactivation Token ====================

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

// IsExpired returns true if the token has expired.
func (t *ReactivationToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsUsed returns true if the token has been used.
func (t *ReactivationToken) IsUsed() bool {
	return t.UsedAt != nil
}

// IsValid returns true if the token can still be used.
func (t *ReactivationToken) IsValid() bool {
	return !t.IsExpired() && !t.IsUsed()
}
