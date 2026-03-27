package channel

import (
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
)

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
		channel := NewTerminalChannelWithConfig(podKey, m.buildChannelConfig(), m.onAllSubscribersGone, m.onChannelClosed)
		m.channels[podKey] = channel
		m.mu.Unlock()

		channel.SetPublisher(pending.conn)
		channel.AddSubscriber(subscriberID, conn)

		m.logger.Info("Channel created (subscriber connected to waiting publisher)", "pod_key", podKey)
		return nil
	}

	// No publisher waiting, add to pending subscribers
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

// newLogger creates the default logger for the channel manager.
func newLogger() *slog.Logger {
	return slog.With("component", "channel_manager")
}
