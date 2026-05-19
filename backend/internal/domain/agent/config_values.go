package agent

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

const (
	ConfigKeyModel          = "model"
	ConfigKeyPermissionMode = "permission_mode"
)

type ConfigValues map[string]interface{}

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

func (cv ConfigValues) Value() (driver.Value, error) {
	if cv == nil {
		return json.Marshal(make(map[string]interface{}))
	}
	return json.Marshal(cv)
}

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
