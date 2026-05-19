package runner

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type HostInfo map[string]interface{}

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

func (hi HostInfo) Value() (driver.Value, error) {
	if hi == nil {
		return nil, nil
	}
	return json.Marshal(hi)
}

type StringSlice []string

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

func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

type AgentVersion struct {
	Slug    string `json:"slug"`
	Version string `json:"version"`
	Path    string `json:"path,omitempty"`
}

type AgentVersionSlice []AgentVersion

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

func (s AgentVersionSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

func (s AgentVersionSlice) GetAgentVersion(slug string) *AgentVersion {
	for _, v := range s {
		if v.Slug == slug {
			return &v
		}
	}
	return nil
}

const (
	RunnerStatusOnline  = "online"
	RunnerStatusOffline = "offline"
	RunnerStatusBusy    = "busy"
)

const (
	VisibilityOrganization = "organization"
	VisibilityPrivate      = "private"
)

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

	AvailableAgents StringSlice `gorm:"type:jsonb" json:"available_agents,omitempty"`

	AgentVersions AgentVersionSlice `gorm:"type:jsonb" json:"agent_versions,omitempty"`

	HostInfo HostInfo `gorm:"type:jsonb" json:"host_info,omitempty"`

	Tags StringSlice `gorm:"type:jsonb;default:'[]'" json:"tags,omitempty"`

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

func (r *Runner) IsOnline() bool {
	return r.Status == RunnerStatusOnline
}

func (r *Runner) CanAcceptPod() bool {
	return r.IsEnabled && r.IsOnline() && r.CurrentPods < r.MaxConcurrentPods
}

func (r *Runner) SupportsAgent(agentSlug string) bool {
	for _, slug := range r.AvailableAgents {
		if slug == agentSlug {
			return true
		}
	}
	return false
}

func (r *Runner) CanAcceptPodForAgent(agentSlug string) bool {
	return r.CanAcceptPod() && r.SupportsAgent(agentSlug)
}
