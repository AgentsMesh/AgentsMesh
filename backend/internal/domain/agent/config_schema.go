package agent

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// ConfigSchema defines the configuration fields for an agent type
type ConfigSchema struct {
	Fields []ConfigField `json:"fields"`
}

// Scan implements sql.Scanner for ConfigSchema
func (cs *ConfigSchema) Scan(value interface{}) error {
	if value == nil {
		*cs = ConfigSchema{}
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

// Value implements driver.Valuer for ConfigSchema
func (cs ConfigSchema) Value() (driver.Value, error) {
	return json.Marshal(cs)
}

// ConfigField defines a single configuration field
// Frontend is responsible for i18n using: agent.{slug}.fields.{name}.label
type ConfigField struct {
	Name       string        `json:"name"`                 // Field name (e.g., "model")
	Type       string        `json:"type"`                 // boolean, string, select, number, secret
	Default    interface{}   `json:"default,omitempty"`    // Default value
	Required   bool          `json:"required,omitempty"`   // Whether the field is required
	Options    []FieldOption `json:"options,omitempty"`    // Options for select type
	Validation *Validation   `json:"validation,omitempty"` // Validation rules
	ShowWhen   *Condition    `json:"show_when,omitempty"`  // Conditional display
}

// FieldOption defines an option for select type fields
// Frontend is responsible for i18n using: agent.{slug}.fields.{fieldName}.options.{value}
type FieldOption struct {
	Value string `json:"value"`
}

// Validation defines validation rules for a field
type Validation struct {
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`   // Regex pattern
	MinLength *int     `json:"min_length,omitempty"`
	MaxLength *int     `json:"max_length,omitempty"`
}

// Condition defines a condition for conditional display or argument inclusion
type Condition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, neq, in, not_in, empty, not_empty
	Value    interface{} `json:"value,omitempty"`
}

// CommandTemplate defines how to build launch arguments from config
type CommandTemplate struct {
	Args []ArgRule `json:"args"`
}

// Scan implements sql.Scanner for CommandTemplate
func (ct *CommandTemplate) Scan(value interface{}) error {
	if value == nil {
		*ct = CommandTemplate{}
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
	return json.Unmarshal(data, ct)
}

// Value implements driver.Valuer for CommandTemplate
func (ct CommandTemplate) Value() (driver.Value, error) {
	return json.Marshal(ct)
}

// ArgRule defines a rule for adding arguments
type ArgRule struct {
	Condition *Condition `json:"condition,omitempty"` // When to add these args
	Args      []string   `json:"args"`                // Argument templates, supports {{.config.xxx}}
}

// FilesTemplate defines files to create in the sandbox
type FilesTemplate []FileTemplate

// Scan implements sql.Scanner for FilesTemplate
func (ft *FilesTemplate) Scan(value interface{}) error {
	if value == nil {
		*ft = nil
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
	return json.Unmarshal(data, ft)
}

// Value implements driver.Valuer for FilesTemplate
func (ft FilesTemplate) Value() (driver.Value, error) {
	if ft == nil {
		return nil, nil
	}
	return json.Marshal(ft)
}

// FileTemplate defines a file to create in the sandbox
type FileTemplate struct {
	Condition       *Condition `json:"condition,omitempty"`    // When to create this file
	PathTemplate    string     `json:"path_template"`          // Path template, supports {{.sandbox.root_path}}, {{.sandbox.work_dir}}
	ContentTemplate string     `json:"content_template"`       // Content template, supports Go template syntax
	Mode            int        `json:"mode,omitempty"`         // File permission, default 0644
	IsDirectory     bool       `json:"is_directory,omitempty"` // Whether this is a directory
}

// Evaluate checks if the condition is met given the config values
func (c *Condition) Evaluate(config map[string]interface{}) bool {
	if c == nil {
		return true
	}

	fieldValue, exists := config[c.Field]

	switch c.Operator {
	case "eq":
		return exists && fieldValue == c.Value
	case "neq":
		return !exists || fieldValue != c.Value
	case "empty":
		return !exists || fieldValue == nil || fieldValue == ""
	case "not_empty":
		return exists && fieldValue != nil && fieldValue != ""
	case "in":
		if values, ok := c.Value.([]interface{}); ok {
			for _, v := range values {
				if fieldValue == v {
					return true
				}
			}
		}
		return false
	case "not_in":
		if values, ok := c.Value.([]interface{}); ok {
			for _, v := range values {
				if fieldValue == v {
					return false
				}
			}
		}
		return true
	default:
		return true
	}
}
