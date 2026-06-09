package channel

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type MessageMentions struct {
	Pods    []string `json:"pods,omitempty"`
	Users   []int64  `json:"users,omitempty"`
	Channel bool     `json:"channel,omitempty"`
}

func (mm *MessageMentions) Scan(value interface{}) error {
	if value == nil {
		*mm = MessageMentions{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("unsupported type for MessageMentions.Scan")
	}
	return json.Unmarshal(bytes, mm)
}

func (mm MessageMentions) Value() (driver.Value, error) {
	return json.Marshal(mm)
}
