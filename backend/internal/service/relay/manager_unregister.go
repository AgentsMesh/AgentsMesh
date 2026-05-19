package relay

import (
	"context"
)

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
