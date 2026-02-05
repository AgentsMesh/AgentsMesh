package monitor

// SetCallback sets the status change callback.
// Deprecated: Use Subscribe/Unsubscribe for multi-subscriber support.
func (m *Monitor) SetCallback(callback func(PodStatus)) {
	m.Subscribe("default", callback)
}

// Subscribe registers a callback for status change notifications.
// Multiple subscribers can be registered with unique IDs.
// If the same ID is used, the previous callback will be replaced.
func (m *Monitor) Subscribe(id string, callback func(PodStatus)) {
	m.subMu.Lock()
	defer m.subMu.Unlock()
	m.subscribers[id] = callback
	log.Info("Registered status subscriber", "id", id)
}

// Unsubscribe removes a previously registered callback.
func (m *Monitor) Unsubscribe(id string) {
	m.subMu.Lock()
	defer m.subMu.Unlock()
	delete(m.subscribers, id)
	log.Info("Unregistered status subscriber", "id", id)
}

// notifySubscribers notifies all registered subscribers of a status change.
// Callbacks are invoked in separate goroutines to prevent blocking.
func (m *Monitor) notifySubscribers(status PodStatus) {
	m.subMu.RLock()
	defer m.subMu.RUnlock()

	for id, cb := range m.subscribers {
		// Invoke callback in a goroutine to prevent blocking
		go func(subscriberID string, callback func(PodStatus)) {
			defer func() {
				if r := recover(); r != nil {
					log.Error("Subscriber callback panic recovered",
						"subscriber_id", subscriberID,
						"panic", r)
				}
			}()
			callback(status)
		}(id, cb)
	}
}
