package websocket

import (
	"encoding/json"
	"time"
)

// ========== Constants ==========

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 64 * 1024 // 64KB
)

// ========== Message Types ==========

// MessageType defines the type of WebSocket message
type MessageType string

const (
	MessageTypePodStatus      MessageType = "pod:status"
	MessageTypeAgentStatus    MessageType = "agent:status"
	MessageTypeChannelMessage MessageType = "channel:message"
	MessageTypePing           MessageType = "ping"
	MessageTypePong           MessageType = "pong"
	MessageTypeError          MessageType = "error"
	// NOTE: pod:input/pod:output/pod:resize are NOT here - pod I/O is streamed via Relay, not WebSocket
)

// ========== Message Structures ==========

// Message represents a WebSocket message
type Message struct {
	Type      MessageType     `json:"type"`
	PodKey    string          `json:"pod_key,omitempty"`
	ChannelID int64           `json:"channel_id,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

// PodStatusData represents pod status update
type PodStatusData struct {
	Status      string `json:"status"`
	AgentStatus string `json:"agent_status,omitempty"`
}

// ========== Broadcast Message Structures ==========

// PodMessage represents a message to broadcast to a pod
type PodMessage struct {
	PodKey  string
	Message []byte
}

// ChannelMessage represents a message to broadcast to a channel
type ChannelMessage struct {
	ChannelID int64
	Message   []byte
}

// OrgMessage represents a message to broadcast to an organization
type OrgMessage struct {
	OrgID   int64
	Message []byte
}

// UserMessage represents a message to send to a specific user
type UserMessage struct {
	UserID  int64
	Message []byte
}
