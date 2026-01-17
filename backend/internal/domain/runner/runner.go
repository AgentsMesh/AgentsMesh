package runner

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// HostInfo represents runner host information
type HostInfo map[string]interface{}

// Scan implements sql.Scanner for HostInfo
func (hi *HostInfo) Scan(value interface{}) error {
	if value == nil {
		*hi = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, hi)
}

// Value implements driver.Valuer for HostInfo
func (hi HostInfo) Value() (driver.Value, error) {
	if hi == nil {
		return nil, nil
	}
	return json.Marshal(hi)
}

// StringSlice is a custom type for []string that implements sql.Scanner and driver.Valuer
type StringSlice []string

// Scan implements sql.Scanner for StringSlice
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, s)
}

// Value implements driver.Valuer for StringSlice
func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// RegistrationToken represents a token used to register runners
type RegistrationToken struct {
	ID             int64  `gorm:"primaryKey" json:"id"`
	OrganizationID int64  `gorm:"not null;index" json:"organization_id"`
	TokenHash      string `gorm:"size:255;not null;uniqueIndex" json:"-"`
	Description    string `gorm:"type:text" json:"description,omitempty"`
	CreatedByID    int64  `gorm:"not null" json:"created_by_id"`

	IsActive  bool       `gorm:"not null;default:true" json:"is_active"`
	MaxUses   *int       `json:"max_uses,omitempty"`
	UsedCount int        `gorm:"not null;default:0" json:"used_count"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
}

func (RegistrationToken) TableName() string {
	return "runner_registration_tokens"
}

// Runner status constants
const (
	RunnerStatusOnline  = "online"
	RunnerStatusOffline = "offline"
	RunnerStatusBusy    = "busy"
)

// Runner represents a self-hosted runner
type Runner struct {
	ID             int64  `gorm:"primaryKey" json:"id"`
	OrganizationID int64  `gorm:"not null;index" json:"organization_id"`
	NodeID         string `gorm:"size:100;not null" json:"node_id"`
	Description    string `gorm:"type:text" json:"description,omitempty"`

	Status            string     `gorm:"size:50;not null;default:'offline';index" json:"status"`
	LastHeartbeat     *time.Time `json:"last_heartbeat,omitempty"`
	CurrentPods       int        `gorm:"not null;default:0" json:"current_pods"`
	MaxConcurrentPods int        `gorm:"not null;default:5" json:"max_concurrent_pods"`
	RunnerVersion     *string    `gorm:"size:50" json:"runner_version,omitempty"`
	IsEnabled         bool       `gorm:"not null;default:true" json:"is_enabled"`

	// AvailableAgents is the list of agent type slugs available on this runner
	// Populated during initialization handshake
	AvailableAgents StringSlice `gorm:"type:jsonb" json:"available_agents,omitempty"`

	HostInfo HostInfo `gorm:"type:jsonb" json:"host_info,omitempty"`

	// mTLS certificate fields (added for gRPC migration)
	CertSerialNumber *string    `gorm:"size:64" json:"cert_serial_number,omitempty"`
	CertExpiresAt    *time.Time `json:"cert_expires_at,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

func (Runner) TableName() string {
	return "runners"
}

// IsOnline returns true if runner is online
func (r *Runner) IsOnline() bool {
	return r.Status == RunnerStatusOnline
}

// CanAcceptPod returns true if runner can accept new pods
func (r *Runner) CanAcceptPod() bool {
	return r.IsEnabled && r.IsOnline() && r.CurrentPods < r.MaxConcurrentPods
}

// SupportsAgent returns true if runner supports the given agent type slug
func (r *Runner) SupportsAgent(agentSlug string) bool {
	for _, slug := range r.AvailableAgents {
		if slug == agentSlug {
			return true
		}
	}
	return false
}

// CanAcceptPodForAgent returns true if runner can accept a pod for the given agent type
func (r *Runner) CanAcceptPodForAgent(agentSlug string) bool {
	return r.CanAcceptPod() && r.SupportsAgent(agentSlug)
}
