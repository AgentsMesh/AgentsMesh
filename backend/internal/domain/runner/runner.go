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
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("unsupported type for HostInfo Scan")
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
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("unsupported type for StringSlice Scan")
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

// AgentVersion represents the version info of an installed agent on a runner.
type AgentVersion struct {
	Slug    string `json:"slug"`
	Version string `json:"version"`
	Path    string `json:"path,omitempty"`
}

// AgentVersionSlice is a custom type for []AgentVersion that implements sql.Scanner and driver.Valuer
type AgentVersionSlice []AgentVersion

// Scan implements sql.Scanner for AgentVersionSlice
func (s *AgentVersionSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("unsupported type for AgentVersionSlice Scan")
	}
	return json.Unmarshal(bytes, s)
}

// Value implements driver.Valuer for AgentVersionSlice
func (s AgentVersionSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// GetAgentVersion returns the version info for a specific agent slug, or nil if not found.
func (s AgentVersionSlice) GetAgentVersion(slug string) *AgentVersion {
	for _, v := range s {
		if v.Slug == slug {
			return &v
		}
	}
	return nil
}

// Runner status constants
const (
	RunnerStatusOnline  = "online"
	RunnerStatusOffline = "offline"
	RunnerStatusBusy    = "busy"
)

// Runner visibility constants
const (
	VisibilityOrganization = "organization"
	VisibilityPrivate      = "private"
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

	// AvailableAgents is the list of agent slugs available on this runner
	// Populated during initialization handshake
	AvailableAgents StringSlice `gorm:"type:jsonb" json:"available_agents,omitempty"`

	// AgentVersions stores detected version info for each available agent
	// Populated during initialization handshake (requires Runner >= 0.4.7)
	AgentVersions AgentVersionSlice `gorm:"type:jsonb" json:"agent_versions,omitempty"`

	HostInfo HostInfo `gorm:"type:jsonb" json:"host_info,omitempty"`

	// Visibility controls who can see/use this runner
	Visibility         string `gorm:"size:20;not null;default:'organization'" json:"visibility"`
	RegisteredByUserID *int64 `json:"registered_by_user_id,omitempty"`

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

// SupportsAgent returns true if runner supports the given agent slug
func (r *Runner) SupportsAgent(agentSlug string) bool {
	for _, slug := range r.AvailableAgents {
		if slug == agentSlug {
			return true
		}
	}
	return false
}

// CanAcceptPodForAgent returns true if runner can accept a pod for the given agent
func (r *Runner) CanAcceptPodForAgent(agentSlug string) bool {
	return r.CanAcceptPod() && r.SupportsAgent(agentSlug)
}
