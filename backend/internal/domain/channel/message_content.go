package channel

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

const (
	InlineText      = "text"
	InlineMention   = "mention"
	InlineLink      = "link"
	InlineLinebreak = "linebreak"
)

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

	if raw.Style != nil {
		el.Style = raw.Style
	} else if raw.Bold || raw.Italic || raw.Strike || raw.Code {
		el.Style = &InlineStyle{Bold: raw.Bold, Italic: raw.Italic, Strike: raw.Strike, Code: raw.Code}
	}
	return nil
}

type Block struct {
	Type     string          `json:"type"`
	Elements []InlineElement `json:"elements,omitempty"`
	Children []Block         `json:"children,omitempty"`
	Level    int             `json:"level,omitempty"`
	Language string          `json:"language,omitempty"`
	Text     string          `json:"text,omitempty"`
	Ordered  bool            `json:"ordered,omitempty"`
	Items [][]Block `json:"items,omitempty"`
}

// blockRaw is the on-wire shape used during decoding so we can intercept the
// `items` field and accept both the new `[][]Block` shape and the legacy
// `[][]InlineElement` shape (rows written before migration 000130). Legacy
// items are wrapped into a single paragraph block so reads always produce
// schema-valid blocks regardless of when the row was written.
type blockRaw struct {
	Type     string            `json:"type"`
	Elements []InlineElement   `json:"elements,omitempty"`
	Children []Block           `json:"children,omitempty"`
	Level    int               `json:"level,omitempty"`
	Language string            `json:"language,omitempty"`
	Text     string            `json:"text,omitempty"`
	Ordered  bool              `json:"ordered,omitempty"`
	Items    []json.RawMessage `json:"items,omitempty"`
}

var legacyBlockTypes = map[string]bool{
	"paragraph": true, "heading": true, "code_block": true, "quote": true, "list": true,
}

func (b *Block) UnmarshalJSON(data []byte) error {
	var raw blockRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	b.Type = raw.Type
	b.Elements = raw.Elements
	b.Children = raw.Children
	b.Level = raw.Level
	b.Language = raw.Language
	b.Text = raw.Text
	b.Ordered = raw.Ordered
	if len(raw.Items) == 0 {
		b.Items = nil
		return nil
	}
	b.Items = make([][]Block, 0, len(raw.Items))
	for _, item := range raw.Items {
		blocks, err := decodeItem(item)
		if err != nil {
			return err
		}
		b.Items = append(b.Items, blocks)
	}
	return nil
}

func decodeItem(raw json.RawMessage) ([]Block, error) {
	var probe []json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, err
	}
	if len(probe) == 0 {
		return nil, nil
	}
	var first struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(probe[0], &first); err != nil {
		return nil, err
	}
	if legacyBlockTypes[first.Type] {
		var blocks []Block
		if err := json.Unmarshal(raw, &blocks); err != nil {
			return nil, err
		}
		return blocks, nil
	}
	var inline []InlineElement
	if err := json.Unmarshal(raw, &inline); err != nil {
		return nil, err
	}
	return []Block{{Type: "paragraph", Elements: inline}}, nil
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
