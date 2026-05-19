package blockstore

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type JSONMap map[string]any

func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return errors.New("unsupported type for blockstore.JSONMap Scan")
	}
	return json.Unmarshal(raw, m)
}

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}

func (m JSONMap) Clone() JSONMap {
	if m == nil {
		return nil
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	var out JSONMap
	_ = json.Unmarshal(raw, &out)
	return out
}
