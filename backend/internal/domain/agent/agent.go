package agent

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// CredentialSchema represents the JSON schema for agent credentials
type CredentialSchema []CredentialField

type CredentialField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`     // secret, text
	EnvVar   string `json:"env_var"`  // Environment variable name
	Required bool   `json:"required"`
}

// Scan implements sql.Scanner for CredentialSchema
func (cs *CredentialSchema) Scan(value interface{}) error {
	if value == nil {
		*cs = nil
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
	return json.Unmarshal(data, cs)
}

// Value implements driver.Valuer for CredentialSchema
func (cs CredentialSchema) Value() (driver.Value, error) {
	if cs == nil {
		return nil, nil
	}
	return json.Marshal(cs)
}

// StatusDetection represents the configuration for agent status detection
type StatusDetection map[string]interface{}

// Scan implements sql.Scanner for StatusDetection
func (sd *StatusDetection) Scan(value interface{}) error {
	if value == nil {
		*sd = nil
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
	return json.Unmarshal(data, sd)
}

// Value implements driver.Valuer for StatusDetection
func (sd StatusDetection) Value() (driver.Value, error) {
	if sd == nil {
		return nil, nil
	}
	return json.Marshal(sd)
}

// Agent represents a code agent definition (builtin or custom)
type Agent struct {
	Slug string `gorm:"size:50;primaryKey" json:"slug"`
	Name string `gorm:"size:100;not null" json:"name"`

	Description *string `gorm:"type:text" json:"description,omitempty"`

	LaunchCommand string  `gorm:"size:500;not null" json:"launch_command"`
	Executable    string  `gorm:"size:100" json:"executable,omitempty"`
	DefaultArgs   *string `gorm:"type:text" json:"default_args,omitempty"`

	// Legacy config fields (replaced by PodFile)
	ConfigSchema    ConfigSchema    `gorm:"type:jsonb;not null;default:'{}'" json:"config_schema"`
	CommandTemplate CommandTemplate `gorm:"type:jsonb;not null;default:'{}'" json:"command_template"`
	FilesTemplate   FilesTemplate   `gorm:"type:jsonb" json:"files_template,omitempty"`

	CredentialSchema CredentialSchema `gorm:"type:jsonb;not null;default:'[]'" json:"credential_schema"`
	StatusDetection  StatusDetection  `gorm:"type:jsonb" json:"status_detection,omitempty"`

	PodfileSource *string `gorm:"type:text" json:"podfile_source,omitempty"`

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

	CredentialSchema CredentialSchema `gorm:"type:jsonb;not null;default:'[]'" json:"credential_schema"`
	StatusDetection  StatusDetection  `gorm:"type:jsonb" json:"status_detection,omitempty"`

	PodfileSource *string `gorm:"type:text" json:"podfile_source,omitempty"`

	IsActive bool `gorm:"not null;default:true" json:"is_active"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

func (CustomAgent) TableName() string {
	return "custom_agents"
}
