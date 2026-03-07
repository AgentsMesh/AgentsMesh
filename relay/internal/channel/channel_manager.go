package channel

import (
	"log/slog"
	"sync"
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

// closeWithReason sends a WebSocket Close frame with a reason before closing the connection.
func closeWithReason(conn *websocket.Conn, reason string) {
	msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, reason)
	_ = conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(writeWait))
	_ = conn.Close()
}

// ChannelManager manages terminal channels
// Channels are keyed by PodKey (not session ID)
type ChannelManager struct {
	mu       sync.RWMutex
	channels map[string]*TerminalChannel // podKey -> channel

	// Pending connections waiting for counterpart
	pendingPublishers  map[string]*pendingPublisher  // podKey -> pending publisher (runner)
	pendingSubscribers map[string]*pendingSubscriber // podKey -> pending subscriber (browser)

	// Configuration
	config ChannelManagerConfig

	// Callbacks
	onAllSubscribersGone func(podKey string)

	// Shutdown signal for cleanupPendingConnections goroutine
	closeOnce sync.Once
	done      chan struct{}

	logger *slog.Logger
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

// NewChannelManager creates a new channel manager with default configuration
func NewChannelManager(keepAliveDuration time.Duration, maxSubscribersPerPod int, onAllSubscribersGone func(string)) *ChannelManager {
	cfg := DefaultChannelManagerConfig()
	cfg.KeepAliveDuration = keepAliveDuration
	cfg.MaxSubscribersPerPod = maxSubscribersPerPod
	return NewChannelManagerWithConfig(cfg, onAllSubscribersGone)
}

// NewChannelManagerWithConfig creates a new channel manager with custom configuration
func NewChannelManagerWithConfig(cfg ChannelManagerConfig, onAllSubscribersGone func(string)) *ChannelManager {
	m := &ChannelManager{
		channels:             make(map[string]*TerminalChannel),
		pendingPublishers:    make(map[string]*pendingPublisher),
		pendingSubscribers:   make(map[string]*pendingSubscriber),
		config:               cfg,
		onAllSubscribersGone: onAllSubscribersGone,
		done:                 make(chan struct{}),
		logger:               slog.With("component", "channel_manager"),
	}

	// Start cleanup goroutine for pending connections
	go m.cleanupPendingConnections()

	return m
}

// HandlePublisherConnect handles a publisher (runner) WebSocket connection
// The channel is identified by podKey, not session ID
func (m *ChannelManager) HandlePublisherConnect(podKey string, conn *websocket.Conn) error {
	m.mu.Lock()

	// Check if channel already exists for this pod
	if channel, ok := m.channels[podKey]; ok {
		m.mu.Unlock()
		// Channel exists, update publisher connection (reconnection scenario)
		channel.SetPublisher(conn)
		m.logger.Info("Publisher reconnected to existing channel", "pod_key", podKey)
		return nil
	}

	// Check if there's a pending subscriber waiting for this pod
	if pending, ok := m.pendingSubscribers[podKey]; ok {
		delete(m.pendingSubscribers, podKey)

		// Create new channel and insert into map while still holding lock.
		// This prevents TOCTOU race where concurrent requests for the same podKey
		// don't see the channel being created.
		channel := NewTerminalChannelWithConfig(podKey, m.buildChannelConfig(), m.onAllSubscribersGone, m.onChannelClosed)
		m.channels[podKey] = channel
		m.mu.Unlock()

		// SetPublisher/AddSubscriber have their own internal locks
		channel.SetPublisher(conn)
		channel.AddSubscriber(pending.subscriberID, pending.conn)

		m.logger.Info("Channel created (publisher connected to waiting subscriber)", "pod_key", podKey)
		return nil
	}

	// No subscriber waiting, add to pending publishers
	// Close any existing pending publisher for this pod to prevent connection leak
	if old, exists := m.pendingPublishers[podKey]; exists {
		closeWithReason(old.conn, "replaced by new publisher connection")
	}
	m.pendingPublishers[podKey] = &pendingPublisher{
		conn:      conn,
		podKey:    podKey,
		createdAt: time.Now(),
	}
	m.mu.Unlock()

	m.logger.Info("Publisher waiting for subscriber", "pod_key", podKey)
	return nil
}

// HandleSubscriberConnect handles a subscriber (browser) WebSocket connection
// The channel is identified by podKey, not session ID
func (m *ChannelManager) HandleSubscriberConnect(podKey, subscriberID string, conn *websocket.Conn) error {
	m.mu.Lock()

	// Check if channel already exists for this pod
	if channel, ok := m.channels[podKey]; ok {
		m.mu.Unlock()

		// Atomically check subscriber limit and add (prevents over-admission race)
		if err := channel.AddSubscriberWithLimit(subscriberID, conn, m.config.MaxSubscribersPerPod); err != nil {
			return err
		}
		m.logger.Info("Subscriber joined existing channel", "pod_key", podKey, "subscriber_id", subscriberID)
		return nil
	}

	// Check if there's a pending publisher waiting for this pod
	if pending, ok := m.pendingPublishers[podKey]; ok {
		delete(m.pendingPublishers, podKey)

		// Create new channel and insert into map while still holding lock.
		// This prevents TOCTOU race where concurrent requests for the same podKey
		// don't see the channel being created.
		channel := NewTerminalChannelWithConfig(podKey, m.buildChannelConfig(), m.onAllSubscribersGone, m.onChannelClosed)
		m.channels[podKey] = channel
		m.mu.Unlock()

		// SetPublisher/AddSubscriber have their own internal locks
		channel.SetPublisher(pending.conn)
		channel.AddSubscriber(subscriberID, conn)

		m.logger.Info("Channel created (subscriber connected to waiting publisher)", "pod_key", podKey)
		return nil
	}

	// No publisher waiting, add to pending subscribers
	// Close any existing pending subscriber for this pod to prevent connection leak
	if old, exists := m.pendingSubscribers[podKey]; exists {
		closeWithReason(old.conn, "replaced by new subscriber connection")
	}
	m.pendingSubscribers[podKey] = &pendingSubscriber{
		conn:         conn,
		subscriberID: subscriberID,
		podKey:       podKey,
		createdAt:    time.Now(),
	}
	m.mu.Unlock()

	m.logger.Info("Subscriber waiting for publisher", "pod_key", podKey, "subscriber_id", subscriberID)
	return nil
}

// GetChannel returns a channel by pod key
func (m *ChannelManager) GetChannel(podKey string) *TerminalChannel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.channels[podKey]
}

// CloseChannel closes and removes a channel by pod key
func (m *ChannelManager) CloseChannel(podKey string) {
	m.mu.Lock()
	channel, ok := m.channels[podKey]
	if ok {
		delete(m.channels, podKey)
	}
	m.mu.Unlock()

	if channel != nil {
		channel.Close()
	}
}

// onChannelClosed is called when a channel closes
func (m *ChannelManager) onChannelClosed(podKey string) {
	m.mu.Lock()
	delete(m.channels, podKey)
	m.mu.Unlock()
	m.logger.Info("Channel removed", "pod_key", podKey)
}

// cleanupPendingConnections periodically cleans up stale pending connections
func (m *ChannelManager) cleanupPendingConnections() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
		case <-m.done:
			return
		}

		m.mu.Lock()

		now := time.Now()
		timeout := m.config.PendingConnectionTimeout

		// Clean up stale pending publishers
		for podKey, pending := range m.pendingPublishers {
			if now.Sub(pending.createdAt) > timeout {
				_ = pending.conn.Close()
				delete(m.pendingPublishers, podKey)
				m.logger.Info("Cleaned up stale pending publisher", "pod_key", podKey)
			}
		}

		// Clean up stale pending subscribers
		for podKey, pending := range m.pendingSubscribers {
			if now.Sub(pending.createdAt) > timeout {
				_ = pending.conn.Close()
				delete(m.pendingSubscribers, podKey)
				m.logger.Info("Cleaned up stale pending subscriber", "pod_key", podKey)
			}
		}

		m.mu.Unlock()
	}
}

// Close stops the cleanup goroutine and cleans up all connections.
// Closes active channels, pending publishers, and pending subscribers.
// Safe to call multiple times.
func (m *ChannelManager) Close() {
	m.closeOnce.Do(func() {
		close(m.done)
	})

	// Close all active channels to release WebSocket connections
	m.mu.Lock()
	channels := make([]*TerminalChannel, 0, len(m.channels))
	for podKey, ch := range m.channels {
		channels = append(channels, ch)
		delete(m.channels, podKey)
	}
	// Clean up pending connections
	for podKey, pending := range m.pendingPublishers {
		_ = pending.conn.Close()
		delete(m.pendingPublishers, podKey)
	}
	for podKey, pending := range m.pendingSubscribers {
		_ = pending.conn.Close()
		delete(m.pendingSubscribers, podKey)
	}
	m.mu.Unlock()

	// Close channels outside the lock to avoid deadlock
	for _, ch := range channels {
		ch.Close()
	}
}

// buildChannelConfig creates a ChannelConfig from ChannelManagerConfig
func (m *ChannelManager) buildChannelConfig() ChannelConfig {
	return ChannelConfig{
		KeepAliveDuration:          m.config.KeepAliveDuration,
		PublisherReconnectTimeout:  m.config.PublisherReconnectTimeout,
		SubscriberReconnectTimeout: m.config.SubscriberReconnectTimeout,
		OutputBufferSize:           m.config.OutputBufferSize,
		OutputBufferCount:          m.config.OutputBufferCount,
	}
}

// Stats returns channel statistics
func (m *ChannelManager) Stats() ChannelStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalSubscribers := 0
	for _, channel := range m.channels {
		totalSubscribers += channel.SubscriberCount()
	}

	return ChannelStats{
		ActiveChannels:     len(m.channels),
		TotalSubscribers:   totalSubscribers,
		PendingPublishers:  len(m.pendingPublishers),
		PendingSubscribers: len(m.pendingSubscribers),
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
