package relay

import "context"

// Flush implements RelayClient. Mock Send calls are synchronous, so there is
// no downstream queue left to drain.
func (m *MockClient) Flush(context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FlushCalls++
	return nil
}

// Reset clears all tracking state.
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ConnectCalled = false
	m.StartCalled = false
	m.StopCalled = false
	m.FlushCalls = 0
	m.UpdateTokenCalls = nil
	m.SentMessages = nil
}

// CountSentByType returns the number of sent messages of the given type.
func (m *MockClient) CountSentByType(msgType byte) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, msg := range m.SentMessages {
		if msg.Type == msgType {
			count++
		}
	}
	return count
}
