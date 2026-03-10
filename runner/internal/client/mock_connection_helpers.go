package client

// GetPods returns pods from handler (if available).
func (m *MockConnection) GetPods() []PodInfo {
	m.mu.Lock()
	handler := m.handler
	m.mu.Unlock()
	if handler != nil {
		return handler.OnListPods()
	}
	return nil
}

// GetEvents returns captured events (thread-safe).
func (m *MockConnection) GetEvents() []EventCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]EventCall, len(m.Events))
	copy(result, m.Events)
	return result
}

// IsStarted returns whether Start was called.
func (m *MockConnection) IsStarted() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.started
}

// IsStopped returns whether Stop was called.
func (m *MockConnection) IsStopped() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopped
}

// Reset clears all captured calls.
func (m *MockConnection) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events = nil
	m.started = false
	m.stopped = false
}

// QueueUsage returns the mock queue usage (always 0 for testing).
func (m *MockConnection) QueueUsage() float64 {
	return 0.0
}
