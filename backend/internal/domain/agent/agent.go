package agent

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
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

// AgentType represents a type of code agent (builtin or custom)
type AgentType struct {
	ID   int64  `gorm:"primaryKey" json:"id"`
	Slug string `gorm:"size:50;not null;uniqueIndex" json:"slug"`
	Name string `gorm:"size:100;not null" json:"name"`

	Description *string `gorm:"type:text" json:"description,omitempty"`

	LaunchCommand string  `gorm:"size:500;not null" json:"launch_command"`
	Executable    string  `gorm:"size:100" json:"executable,omitempty"` // Executable name for availability check
	DefaultArgs   *string `gorm:"type:text" json:"default_args,omitempty"`

	// New fields for config-driven agent setup
	ConfigSchema    ConfigSchema    `gorm:"type:jsonb;not null;default:'{}'" json:"config_schema"`
	CommandTemplate CommandTemplate `gorm:"type:jsonb;not null;default:'{}'" json:"command_template"`
	FilesTemplate   FilesTemplate   `gorm:"type:jsonb" json:"files_template,omitempty"`

	CredentialSchema CredentialSchema `gorm:"type:jsonb;not null;default:'[]'" json:"credential_schema"`
	StatusDetection  StatusDetection  `gorm:"type:jsonb" json:"status_detection,omitempty"`

	IsBuiltin bool `gorm:"not null;default:false" json:"is_builtin"`
	IsActive  bool `gorm:"not null;default:true" json:"is_active"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

func (AgentType) TableName() string {
	return "agent_types"
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


// UserAgentCredential represents user-level agent credentials (overrides org config)
type UserAgentCredential struct {
	ID          int64 `gorm:"primaryKey" json:"id"`
	UserID      int64 `gorm:"not null;index" json:"user_id"`
	AgentTypeID int64 `gorm:"not null" json:"agent_type_id"`

	CredentialsEncrypted EncryptedCredentials `gorm:"type:jsonb;not null" json:"-"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// Associations
	AgentType *AgentType `gorm:"foreignKey:AgentTypeID" json:"agent_type,omitempty"`
}

func (UserAgentCredential) TableName() string {
	return "user_agent_credentials"
}

// CustomAgentType represents organization-specific custom agent types
type CustomAgentType struct {
	ID             int64  `gorm:"primaryKey" json:"id"`
	OrganizationID int64  `gorm:"not null;index" json:"organization_id"`
	Slug           string `gorm:"size:50;not null" json:"slug"`
	Name           string `gorm:"size:100;not null" json:"name"`

	Description *string `gorm:"type:text" json:"description,omitempty"`

	LaunchCommand string  `gorm:"size:500;not null" json:"launch_command"`
	DefaultArgs   *string `gorm:"type:text" json:"default_args,omitempty"`

	CredentialSchema CredentialSchema `gorm:"type:jsonb;not null;default:'[]'" json:"credential_schema"`
	StatusDetection  StatusDetection  `gorm:"type:jsonb" json:"status_detection,omitempty"`

	IsActive bool `gorm:"not null;default:true" json:"is_active"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

func (CustomAgentType) TableName() string {
	return "custom_agent_types"
}
