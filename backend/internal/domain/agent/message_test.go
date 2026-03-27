package agent

import (
	"testing"
	"time"
)

// --- Test Message Type Constants ---

func TestMessageTypeConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{MessageTypeTaskAssignment, "task_assignment"},
		{MessageTypeTaskAccepted, "task_accepted"},
		{MessageTypeTaskCompleted, "task_completed"},
		{MessageTypeTaskFailed, "task_failed"},
		{MessageTypeProgressUpdate, "progress_update"},
		{MessageTypeStatusRequest, "status_request"},
		{MessageTypeStatusResponse, "status_response"},
		{MessageTypeRequirement, "requirement"},
		{MessageTypeClarificationRequest, "clarification_request"},
		{MessageTypeClarificationResponse, "clarification_response"},
		{MessageTypeHelpRequest, "help_request"},
		{MessageTypeHelpResponse, "help_response"},
		{MessageTypeReport, "report"},
		{MessageTypeSummaryRequest, "summary_request"},
		{MessageTypeSummaryResponse, "summary_response"},
		{MessageTypeBindRequest, "bind_request"},
		{MessageTypeBindAccepted, "bind_accepted"},
		{MessageTypeBindRejected, "bind_rejected"},
		{MessageTypeBindRevoked, "bind_revoked"},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("expected '%s', got '%s'", tt.expected, tt.constant)
		}
	}
}

// --- Test Message Status Constants ---

func TestMessageStatusConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{MessageStatusPending, "pending"},
		{MessageStatusDelivered, "delivered"},
		{MessageStatusRead, "read"},
		{MessageStatusFailed, "failed"},
		{MessageStatusDeadLetter, "dead_letter"},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("expected '%s', got '%s'", tt.expected, tt.constant)
		}
	}
}

// --- Test MessageContent ---

func TestMessageContentScanNil(t *testing.T) {
	var mc MessageContent
	err := mc.Scan(nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mc != nil {
		t.Error("expected nil MessageContent")
	}
}

func TestMessageContentScanValid(t *testing.T) {
	var mc MessageContent
	err := mc.Scan([]byte(`{"key":"value","number":123}`))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mc["key"] != "value" {
		t.Errorf("expected 'value', got %v", mc["key"])
	}
}

func TestMessageContentScanInvalidType(t *testing.T) {
	var mc MessageContent
	err := mc.Scan("not bytes")
	if err == nil {
		t.Error("expected error for invalid type")
	}
}

func TestMessageContentScanInvalidJSON(t *testing.T) {
	var mc MessageContent
	err := mc.Scan([]byte(`invalid json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestMessageContentValueNil(t *testing.T) {
	var mc MessageContent
	val, err := mc.Value()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != nil {
		t.Error("expected nil value")
	}
}

func TestMessageContentValueValid(t *testing.T) {
	mc := MessageContent{"key": "value"}
	val, err := mc.Value()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val == nil {
		t.Error("expected non-nil value")
	}
}

// --- Test AgentMessage ---

func TestAgentMessageTableName(t *testing.T) {
	m := AgentMessage{}
	if m.TableName() != "agent_messages" {
		t.Errorf("expected 'agent_messages', got %s", m.TableName())
	}
}

func TestAgentMessageIsRead(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{MessageStatusRead, true},
		{MessageStatusPending, false},
		{MessageStatusDelivered, false},
		{MessageStatusFailed, false},
	}

	for _, tt := range tests {
		m := &AgentMessage{Status: tt.status}
		if m.IsRead() != tt.expected {
			t.Errorf("status %s: expected IsRead() = %v", tt.status, tt.expected)
		}
	}
}

func TestAgentMessageIsDelivered(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{MessageStatusDelivered, true},
		{MessageStatusRead, true}, // Read implies delivered
		{MessageStatusPending, false},
		{MessageStatusFailed, false},
	}

	for _, tt := range tests {
		m := &AgentMessage{Status: tt.status}
		if m.IsDelivered() != tt.expected {
			t.Errorf("status %s: expected IsDelivered() = %v", tt.status, tt.expected)
		}
	}
}

func TestAgentMessageIsPending(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{MessageStatusPending, true},
		{MessageStatusDelivered, false},
		{MessageStatusRead, false},
		{MessageStatusFailed, false},
	}

	for _, tt := range tests {
		m := &AgentMessage{Status: tt.status}
		if m.IsPending() != tt.expected {
			t.Errorf("status %s: expected IsPending() = %v", tt.status, tt.expected)
		}
	}
}

func TestAgentMessageIsFailed(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{MessageStatusFailed, true},
		{MessageStatusDeadLetter, true},
		{MessageStatusPending, false},
		{MessageStatusDelivered, false},
	}

	for _, tt := range tests {
		m := &AgentMessage{Status: tt.status}
		if m.IsFailed() != tt.expected {
			t.Errorf("status %s: expected IsFailed() = %v", tt.status, tt.expected)
		}
	}
}

func TestAgentMessageCanRetry(t *testing.T) {
	tests := []struct {
		name             string
		status           string
		deliveryAttempts int
		maxRetries       int
		expected         bool
	}{
		{"failed with retries left", MessageStatusFailed, 1, 3, true},
		{"failed at max retries", MessageStatusFailed, 3, 3, false},
		{"failed over max retries", MessageStatusFailed, 5, 3, false},
		{"dead letter cannot retry", MessageStatusDeadLetter, 1, 3, false},
		{"pending cannot retry", MessageStatusPending, 0, 3, false},
		{"delivered cannot retry", MessageStatusDelivered, 0, 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &AgentMessage{
				Status:           tt.status,
				DeliveryAttempts: tt.deliveryAttempts,
				MaxRetries:       tt.maxRetries,
			}
			if m.CanRetry() != tt.expected {
				t.Errorf("expected CanRetry() = %v, got %v", tt.expected, m.CanRetry())
			}
		})
	}
}

func TestAgentMessageStruct(t *testing.T) {
	now := time.Now()
	correlationID := "corr-123"
	parentID := int64(99)

	m := AgentMessage{
		ID: 1,
		SenderPod:        "pod-sender",
		ReceiverPod:      "pod-receiver",
		MessageType:      MessageTypeTaskAssignment,
		Content:          MessageContent{"task": "build"},
		Status:           MessageStatusPending,
		DeliveryAttempts: 0,
		MaxRetries:       3,
		ParentMessageID:  &parentID,
		CorrelationID:    &correlationID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if m.ID != 1 {
		t.Errorf("expected ID 1, got %d", m.ID)
	}
	if m.SenderPod != "pod-sender" {
		t.Errorf("expected SenderPod 'pod-sender', got %s", m.SenderPod)
	}
	if m.MessageType != MessageTypeTaskAssignment {
		t.Errorf("expected MessageType 'task_assignment', got %s", m.MessageType)
	}
}

// --- Test DeadLetterEntry ---

func TestDeadLetterEntryTableName(t *testing.T) {
	d := DeadLetterEntry{}
	if d.TableName() != "agent_message_dead_letters" {
		t.Errorf("expected 'agent_message_dead_letters', got %s", d.TableName())
	}
}

func TestDeadLetterEntryStruct(t *testing.T) {
	now := time.Now()
	result := "success"

	d := DeadLetterEntry{
		ID: 1,
		OriginalMessageID: 100,
		Reason:            "Max retries exceeded",
		FinalAttempt:      3,
		MovedAt:           now,
		ReplayedAt:        &now,
		ReplayResult:      &result,
		CreatedAt:         now,
	}

	if d.ID != 1 {
		t.Errorf("expected ID 1, got %d", d.ID)
	}
	if d.OriginalMessageID != 100 {
		t.Errorf("expected OriginalMessageID 100, got %d", d.OriginalMessageID)
	}
	if d.Reason != "Max retries exceeded" {
		t.Errorf("expected Reason 'Max retries exceeded', got %s", d.Reason)
	}
	if d.FinalAttempt != 3 {
		t.Errorf("expected FinalAttempt 3, got %d", d.FinalAttempt)
	}
}

// --- Benchmark Tests ---

func BenchmarkAgentMessageIsRead(b *testing.B) {
	m := &AgentMessage{Status: MessageStatusRead}
	for i := 0; i < b.N; i++ {
		m.IsRead()
	}
}

func BenchmarkAgentMessageIsDelivered(b *testing.B) {
	m := &AgentMessage{Status: MessageStatusDelivered}
	for i := 0; i < b.N; i++ {
		m.IsDelivered()
	}
}

func BenchmarkAgentMessageCanRetry(b *testing.B) {
	m := &AgentMessage{Status: MessageStatusFailed, DeliveryAttempts: 1, MaxRetries: 3}
	for i := 0; i < b.N; i++ {
		m.CanRetry()
	}
}

func BenchmarkMessageContentScan(b *testing.B) {
	data := []byte(`{"key":"value","number":123}`)
	for i := 0; i < b.N; i++ {
		var mc MessageContent
		mc.Scan(data)
	}
}
