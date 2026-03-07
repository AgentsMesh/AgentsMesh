package relay

import (
	"context"
)

// ForceUnregister removes a relay immediately.
// Skips store deletion if the relay is not in memory (avoids unnecessary I/O).
func (m *Manager) ForceUnregister(relayID string) {
	m.mu.Lock()
	_, existed := m.relays[relayID]
	delete(m.relays, relayID)
	m.mu.Unlock()

	if existed {
		m.deleteFromStore(relayID)
	}
	m.logger.Info("Relay force unregistered", "relay_id", relayID)
}

// GracefulUnregister marks a relay as offline (graceful shutdown from relay itself)
func (m *Manager) GracefulUnregister(relayID string, reason string) {
	m.mu.Lock()

	_, ok := m.relays[relayID]
	if !ok {
		m.mu.Unlock()
		return
	}

	delete(m.relays, relayID)
	m.mu.Unlock()

	m.deleteFromStore(relayID)
	m.logger.Info("Relay gracefully unregistered",
		"relay_id", relayID,
		"reason", reason)
}

// deleteFromStore removes a relay from the persistent store if configured.
func (m *Manager) deleteFromStore(relayID string) {
	if m.store == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), storeOpTimeout)
	defer cancel()
	if err := m.store.DeleteRelay(ctx, relayID); err != nil {
		m.logger.Warn("Failed to delete relay from store", "relay_id", relayID, "error", err)
	}
}
