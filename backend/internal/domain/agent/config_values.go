package agent

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// Canonical AgentFile CONFIG keys consumed by backend code. Bare-string usage
// in service/orchestrator code should reference these constants to avoid typos
// at the boundary between AgentFile DSL and Go.
const (
	ConfigKeyModel          = "model"
	ConfigKeyPermissionMode = "permission_mode"
)

// ConfigValues represents dynamic configuration values (JSONB)
type ConfigValues map[string]interface{}

// GetString returns the string value at key, or "" when missing / wrong type / nil.
func (cv ConfigValues) GetString(key string) string {
	if cv == nil {
		return ""
	}
	v, ok := cv[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// Scan implements sql.Scanner for ConfigValues
func (cv *ConfigValues) Scan(value interface{}) error {
	if value == nil {
		*cv = make(ConfigValues)
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("type assertion to []byte or string failed")
	}
	return json.Unmarshal(bytes, cv)
}

// Value implements driver.Valuer for ConfigValues
func (cv ConfigValues) Value() (driver.Value, error) {
	if cv == nil {
		return json.Marshal(make(map[string]interface{}))
	}
	return json.Marshal(cv)
}

// MergeConfigs merges multiple config maps with priority (later maps override earlier).
// Used for: AgentFile CONFIG defaults -> user personal config -> pod overrides
func MergeConfigs(configs ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for _, config := range configs {
		if config == nil {
			continue
		}
		for k, v := range config {
			result[k] = v
		}
	}

	return result
}
