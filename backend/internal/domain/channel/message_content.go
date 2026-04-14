package channel

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// InlineElement types
const (
	InlineText      = "text"
	InlineMention   = "mention"
	InlineLink      = "link"
	InlineLinebreak = "linebreak"
)

// Mention entity types
const (
	EntityPod     = "pod"
	EntityUser    = "user"
	EntityTicket  = "ticket"
	EntityChannel = "channel"
)

type InlineStyle struct {
	Bold   bool `json:"bold,omitempty"`
	Italic bool `json:"italic,omitempty"`
	Strike bool `json:"strike,omitempty"`
	Code   bool `json:"code,omitempty"`
}

func (s *InlineStyle) IsEmpty() bool {
	return s == nil || (!s.Bold && !s.Italic && !s.Strike && !s.Code)
}

type InlineElement struct {
	Type       string       `json:"type"`
	Text       string       `json:"text,omitempty"`
	Style      *InlineStyle `json:"style,omitempty"`
	EntityType string       `json:"entity_type,omitempty"`
	EntityKey  string       `json:"entity_key,omitempty"`
	Display    string       `json:"display,omitempty"`
	URL        string       `json:"url,omitempty"`
}

// inlineElementRaw is used for backward-compatible deserialization.
// Old messages have flat boolean fields (bold, italic, strike, code).
// New messages use the consolidated Style object.
type inlineElementRaw struct {
	Type       string       `json:"type"`
	Text       string       `json:"text,omitempty"`
	Style      *InlineStyle `json:"style,omitempty"`
	Bold       bool         `json:"bold,omitempty"`
	Italic     bool         `json:"italic,omitempty"`
	Strike     bool         `json:"strike,omitempty"`
	Code       bool         `json:"code,omitempty"`
	EntityType string       `json:"entity_type,omitempty"`
	EntityKey  string       `json:"entity_key,omitempty"`
	Display    string       `json:"display,omitempty"`
	URL        string       `json:"url,omitempty"`
}

func (el *InlineElement) UnmarshalJSON(data []byte) error {
	var raw inlineElementRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	el.Type = raw.Type
	el.Text = raw.Text
	el.EntityType = raw.EntityType
	el.EntityKey = raw.EntityKey
	el.Display = raw.Display
	el.URL = raw.URL

	// Prefer new Style object; fall back to old flat booleans
	if raw.Style != nil {
		el.Style = raw.Style
	} else if raw.Bold || raw.Italic || raw.Strike || raw.Code {
		el.Style = &InlineStyle{Bold: raw.Bold, Italic: raw.Italic, Strike: raw.Strike, Code: raw.Code}
	}
	return nil
}

type Block struct {
	Type     string            `json:"type"`
	Elements []InlineElement   `json:"elements,omitempty"`
	Children []Block           `json:"children,omitempty"`
	Level    int               `json:"level,omitempty"`
	Language string            `json:"language,omitempty"`
	Text     string            `json:"text,omitempty"`
	Ordered  bool              `json:"ordered,omitempty"`
	Items    [][]InlineElement `json:"items,omitempty"`
}

type MessageContent struct {
	SchemaVersion int     `json:"schema_version"`
	Kind          string  `json:"kind"`
	Blocks        []Block `json:"blocks,omitempty"`
	AttachmentKey string  `json:"attachment_key,omitempty"`
}

func (mc *MessageContent) Scan(value interface{}) error {
	if value == nil {
		*mc = MessageContent{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("unsupported type for MessageContent.Scan")
	}
	return json.Unmarshal(bytes, mc)
}

func (mc MessageContent) Value() (driver.Value, error) {
	return json.Marshal(mc)
}

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
