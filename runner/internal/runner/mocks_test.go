package runner

import (
	"sync"
)

// mockEventSender is a mock for testing EventSender interface
type mockEventSender struct {
	statuses []struct {
		podKey string
		status     string
		data       map[string]interface{}
	}
	outputs []struct {
		podKey string
		data       []byte
	}
	mu sync.Mutex
}

func newMockEventSender() *mockEventSender {
	return &mockEventSender{}
}

func (m *mockEventSender) SendPodStatus(podKey, status string, data map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statuses = append(m.statuses, struct {
		podKey string
		status     string
		data       map[string]interface{}
	}{podKey, status, data})
}

// NOTE: SendTerminalOutput removed - terminal output is exclusively streamed via Relay

// mockOutputHandler is a mock for testing OutputHandler interface
type mockOutputHandler struct {
	outputs []struct {
		podKey string
		data       []byte
	}
	shouldBackpressure bool
	mu                 sync.Mutex
}

func newMockOutputHandler() *mockOutputHandler {
	return &mockOutputHandler{}
}

func (m *mockOutputHandler) SendOutput(podKey string, data []byte) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outputs = append(m.outputs, struct {
		podKey string
		data       []byte
	}{podKey, data})
	return !m.shouldBackpressure
}

func (m *mockOutputHandler) GetOutputs() []struct {
	podKey string
	data       []byte
} {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]struct {
		podKey string
		data       []byte
	}, len(m.outputs))
	copy(result, m.outputs)
	return result
}

func (m *mockOutputHandler) SetBackpressure(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldBackpressure = enabled
}

// contains checks if s contains substr (helper for tests)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
