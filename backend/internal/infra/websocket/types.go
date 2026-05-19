package websocket

import (
	"encoding/json"
	"time"
)

const (
	writeWait = 10 * time.Second

	pongWait = 60 * time.Second

	pingPeriod = (pongWait * 9) / 10

	maxMessageSize = 64 * 1024
)

type MessageType string

const (
	MessageTypePodStatus      MessageType = "pod:status"
	MessageTypeAgentStatus    MessageType = "agent:status"
	MessageTypeChannelMessage MessageType = "channel:message"
	MessageTypePing           MessageType = "ping"
	MessageTypePong           MessageType = "pong"
	MessageTypeError          MessageType = "error"
)

type Message struct {
	Type      MessageType     `json:"type"`
	PodKey    string          `json:"pod_key,omitempty"`
	ChannelID int64           `json:"channel_id,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

type PodStatusData struct {
	Status      string `json:"status"`
	AgentStatus string `json:"agent_status,omitempty"`
}

type PodMessage struct {
	PodKey  string
	Message []byte
}

type ChannelMessage struct {
	ChannelID int64
	Message   []byte
}

type OrgMessage struct {
	OrgID   int64
	Message []byte
}

type UserMessage struct {
	UserID  int64
	Message []byte
}
