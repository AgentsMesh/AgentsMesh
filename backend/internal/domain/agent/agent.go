package agent

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Agent represents a code agent definition (builtin or custom)
type Agent struct {
	Slug string `gorm:"size:50;primaryKey" json:"slug"`
	Name string `gorm:"size:100;not null" json:"name"`

	Description *string `gorm:"type:text" json:"description,omitempty"`

	LaunchCommand string  `gorm:"size:500;not null" json:"launch_command"`
	Executable    string  `gorm:"size:100" json:"executable,omitempty"`
	DefaultArgs   *string `gorm:"type:text" json:"default_args,omitempty"`

	AgentfileSource *string `gorm:"type:text;column:agentfile_source" json:"agentfile_source,omitempty"`

	IsBuiltin bool `gorm:"not null;default:false" json:"is_builtin"`
	IsActive  bool `gorm:"not null;default:true" json:"is_active"`

	SupportedModes string `gorm:"column:supported_modes;type:varchar(50);default:pty;not null" json:"supported_modes"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

func (Agent) TableName() string {
	return "agents"
}

// SupportsMode returns true if this agent supports the given interaction mode.
func (a *Agent) SupportsMode(mode string) bool {
	for _, m := range strings.Split(a.SupportedModes, ",") {
		if strings.TrimSpace(m) == mode {
			return true
		}
	}
	return false
}

// EncryptedCredentials represents encrypted credential storage
type EncryptedCredentials map[string]string

// Scan implements sql.Scanner for EncryptedCredentials
func (ec *EncryptedCredentials) Scan(value interface{}) error {
	if value == nil {
		*ec = nil
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return errors.New("type assertion to []byte or string failed")
	}
	return json.Unmarshal(data, ec)
}

// Value implements driver.Valuer for EncryptedCredentials
func (ec EncryptedCredentials) Value() (driver.Value, error) {
	if ec == nil {
		return nil, nil
	}
	return json.Marshal(ec)
}

// CustomAgent represents an organization-specific custom agent
type CustomAgent struct {
	OrganizationID int64  `gorm:"primaryKey;autoIncrement:false" json:"organization_id"`
	Slug           string `gorm:"primaryKey;size:50" json:"slug"`
	Name           string `gorm:"size:100;not null" json:"name"`

	Description *string `gorm:"type:text" json:"description,omitempty"`

	LaunchCommand string  `gorm:"size:500;not null" json:"launch_command"`
	DefaultArgs   *string `gorm:"type:text" json:"default_args,omitempty"`

	AgentfileSource *string `gorm:"type:text;column:agentfile_source" json:"agentfile_source,omitempty"`

	IsActive bool `gorm:"not null;default:true" json:"is_active"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

func (CustomAgent) TableName() string {
	return "custom_agents"
}
