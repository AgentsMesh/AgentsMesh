package agent

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// Message type constants
const (
	// Task related
	MessageTypeTaskAssignment = "task_assignment"
	MessageTypeTaskAccepted   = "task_accepted"
	MessageTypeTaskCompleted  = "task_completed"
	MessageTypeTaskFailed     = "task_failed"

	// Progress related
	MessageTypeProgressUpdate  = "progress_update"
	MessageTypeStatusRequest   = "status_request"
	MessageTypeStatusResponse  = "status_response"

	// Requirement related
	MessageTypeRequirement           = "requirement"
	MessageTypeClarificationRequest  = "clarification_request"
	MessageTypeClarificationResponse = "clarification_response"

	// Assistance/Reporting
	MessageTypeHelpRequest    = "help_request"
	MessageTypeHelpResponse   = "help_response"
	MessageTypeReport         = "report"
	MessageTypeSummaryRequest = "summary_request"
	MessageTypeSummaryResponse = "summary_response"

	// Binding related (system messages)
	MessageTypeBindRequest  = "bind_request"
	MessageTypeBindAccepted = "bind_accepted"
	MessageTypeBindRejected = "bind_rejected"
	MessageTypeBindRevoked  = "bind_revoked"
)

// Message status constants
const (
	MessageStatusPending    = "pending"     // Waiting to be delivered
	MessageStatusDelivered  = "delivered"   // Successfully delivered
	MessageStatusRead       = "read"        // Marked as read by receiver
	MessageStatusFailed     = "failed"      // Delivery failed (will retry)
	MessageStatusDeadLetter = "dead_letter" // Max retries exceeded
)

// MessageContent represents message content as JSON
type MessageContent map[string]interface{}

// Scan implements sql.Scanner for MessageContent
func (mc *MessageContent) Scan(value interface{}) error {
	if value == nil {
		*mc = nil
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return errors.New("type assertion to []byte or string failed")
	}
	return json.Unmarshal(data, mc)
}

// Value implements driver.Valuer for MessageContent
func (mc MessageContent) Value() (driver.Value, error) {
	if mc == nil {
		return nil, nil
	}
	return json.Marshal(mc)
}

// AgentMessage represents a message between agent pods
type AgentMessage struct {
	ID int64 `gorm:"primaryKey" json:"id"`

	// Sender and receiver pod keys
	SenderPod   string `gorm:"size:100;not null;index;column:sender_pod" json:"sender_pod"`
	ReceiverPod string `gorm:"size:100;not null;index;column:receiver_pod" json:"receiver_pod"`

	// Message type and content
	MessageType string         `gorm:"size:50;not null" json:"message_type"`
	Content     MessageContent `gorm:"type:jsonb;not null;default:'{}'" json:"content"`

	// Delivery status tracking
	Status string `gorm:"size:50;not null;default:'pending'" json:"status"`

	// Delivery attempts and retry info
	DeliveryAttempts    int        `gorm:"not null;default:0" json:"delivery_attempts"`
	MaxRetries          int        `gorm:"not null;default:3" json:"max_retries"`
	LastDeliveryAttempt *time.Time `json:"last_delivery_attempt,omitempty"`
	NextRetryAt         *time.Time `json:"next_retry_at,omitempty"`
	DeliveryError       *string    `gorm:"size:500" json:"delivery_error,omitempty"`

	// Timestamps
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	ReadAt      *time.Time `json:"read_at,omitempty"`

	// Message threading
	ParentMessageID *int64  `json:"parent_message_id,omitempty"`
	CorrelationID   *string `gorm:"size:100;index" json:"correlation_id,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

func (AgentMessage) TableName() string {
	return "agent_messages"
}

// IsRead checks if the message has been read
func (m *AgentMessage) IsRead() bool {
	return m.Status == MessageStatusRead
}

// IsDelivered checks if the message has been delivered
func (m *AgentMessage) IsDelivered() bool {
	return m.Status == MessageStatusDelivered || m.Status == MessageStatusRead
}

// IsPending checks if the message is pending delivery
func (m *AgentMessage) IsPending() bool {
	return m.Status == MessageStatusPending
}

// IsFailed checks if the message delivery failed
func (m *AgentMessage) IsFailed() bool {
	return m.Status == MessageStatusFailed || m.Status == MessageStatusDeadLetter
}

// CanRetry checks if the message can be retried
func (m *AgentMessage) CanRetry() bool {
	return m.Status == MessageStatusFailed && m.DeliveryAttempts < m.MaxRetries
}

// DeadLetterEntry represents a failed message moved to dead letter queue
type DeadLetterEntry struct {
	ID int64 `gorm:"primaryKey" json:"id"`

	// Reference to original message
	OriginalMessageID int64 `gorm:"not null;uniqueIndex" json:"original_message_id"`

	// Failure information
	Reason       string `gorm:"size:500;not null" json:"reason"`
	FinalAttempt int    `gorm:"not null" json:"final_attempt"`
	MovedAt      time.Time `gorm:"not null;default:now()" json:"moved_at"`

	// Replay information
	ReplayedAt   *time.Time `json:"replayed_at,omitempty"`
	ReplayResult *string    `gorm:"size:500" json:"replay_result,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`

	// Association
	OriginalMessage *AgentMessage `gorm:"foreignKey:OriginalMessageID" json:"original_message,omitempty"`
}

func (DeadLetterEntry) TableName() string {
	return "agent_message_dead_letters"
}
