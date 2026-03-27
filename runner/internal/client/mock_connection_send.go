package client

import (
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// Mock Send methods — event recording implementations for testing.

// SendObservePodResult records a pod observation result.
func (m *MockConnection) SendObservePodResult(requestID, podKey, output, screen string, cursorX, cursorY, totalLines int, hasMore bool, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SendErr != nil {
		return m.SendErr
	}
	m.Events = append(m.Events, EventCall{Type: MessageType("pod_snapshot_result"), Data: map[string]interface{}{
		"request_id":  requestID,
		"pod_key":     podKey,
		"output":      output,
		"screen":      screen,
		"cursor_x":    cursorX,
		"cursor_y":    cursorY,
		"total_lines": totalLines,
		"has_more":    hasMore,
		"error":       errMsg,
	}})
	return nil
}

// SendOSCNotification records an OSC notification event.
func (m *MockConnection) SendOSCNotification(podKey, title, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events = append(m.Events, EventCall{
		Type: "osc_notification",
		Data: map[string]string{"pod_key": podKey, "title": title, "body": body},
	})
	return nil
}

// SendOSCTitle records an OSC title change event.
func (m *MockConnection) SendOSCTitle(podKey, title string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events = append(m.Events, EventCall{
		Type: "osc_title",
		Data: map[string]string{"pod_key": podKey, "title": title},
	})
	return nil
}

// SendAgentStatus records an agent status change event.
func (m *MockConnection) SendAgentStatus(podKey string, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SendErr != nil {
		return m.SendErr
	}
	m.Events = append(m.Events, EventCall{
		Type: MessageType("agent_status"),
		Data: map[string]string{"pod_key": podKey, "status": status},
	})
	return nil
}

// SendUpgradeStatus records an upgrade status event.
func (m *MockConnection) SendUpgradeStatus(event *runnerv1.UpgradeStatusEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SendErr != nil {
		return m.SendErr
	}
	m.Events = append(m.Events, EventCall{Type: "upgrade_status", Data: event})
	return nil
}

// SendLogUploadStatus records a log upload status event.
func (m *MockConnection) SendLogUploadStatus(event *runnerv1.LogUploadStatusEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SendErr != nil {
		return m.SendErr
	}
	m.Events = append(m.Events, EventCall{Type: "log_upload_status", Data: event})
	return nil
}

// SendTokenUsage records a token usage report.
func (m *MockConnection) SendTokenUsage(podKey string, models []*runnerv1.TokenModelUsage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SendErr != nil {
		return m.SendErr
	}
	m.Events = append(m.Events, EventCall{
		Type: "token_usage",
		Data: map[string]any{"pod_key": podKey, "models": models},
	})
	return nil
}

// SendMessage records a raw RunnerMessage.
func (m *MockConnection) SendMessage(msg *runnerv1.RunnerMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SendErr != nil {
		return m.SendErr
	}
	m.Events = append(m.Events, EventCall{Type: "raw_message", Data: msg})
	return nil
}
