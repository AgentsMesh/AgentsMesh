package channel

import (
	"time"

	"github.com/gorilla/websocket"
)

// ChannelManagerConfig holds configuration for the channel manager
type ChannelManagerConfig struct {
	KeepAliveDuration          time.Duration // How long to keep channel alive after all subscribers disconnect
	MaxSubscribersPerPod       int           // Maximum subscribers per pod
	PublisherReconnectTimeout  time.Duration // How long to wait for publisher to reconnect
	SubscriberReconnectTimeout time.Duration // How long to wait for subscriber to reconnect
	PendingConnectionTimeout   time.Duration // How long to wait for counterpart connection
	OutputBufferSize           int           // Max bytes for output buffer
	OutputBufferCount          int           // Max messages for output buffer
}

// DefaultChannelManagerConfig returns default manager configuration
func DefaultChannelManagerConfig() ChannelManagerConfig {
	return ChannelManagerConfig{
		KeepAliveDuration:          30 * time.Second,
		MaxSubscribersPerPod:       10,
		PublisherReconnectTimeout:  30 * time.Second,
		SubscriberReconnectTimeout: 30 * time.Second,
		PendingConnectionTimeout:   60 * time.Second,
		OutputBufferSize:           256 * 1024, // 256KB
		OutputBufferCount:          200,
	}
}

// ChannelStats holds channel statistics
type ChannelStats struct {
	ActiveChannels     int `json:"active_channels"`
	TotalSubscribers   int `json:"total_subscribers"`
	PendingPublishers  int `json:"pending_publishers"`
	PendingSubscribers int `json:"pending_subscribers"`
}

// MaxSubscribersError indicates max subscribers limit reached
type MaxSubscribersError struct {
	Max int
}

func (e *MaxSubscribersError) Error() string {
	return "maximum subscribers per pod reached"
}

type pendingPublisher struct {
	conn      *websocket.Conn
	podKey    string
	createdAt time.Time
}

type pendingSubscriber struct {
	conn         *websocket.Conn
	subscriberID string
	podKey       string
	createdAt    time.Time
}

// closeWithReason sends a WebSocket Close frame with a reason before closing the connection.
func closeWithReason(conn *websocket.Conn, reason string) {
	msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, reason)
	_ = conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(writeWait))
	_ = conn.Close()
}
