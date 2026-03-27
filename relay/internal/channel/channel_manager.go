package channel

import (
	"sync"
	"time"
)

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

	logger interface{ Info(string, ...any) }
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
		logger:               newLogger(),
	}

	// Start cleanup goroutine for pending connections
	go m.cleanupPendingConnections()

	return m
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
