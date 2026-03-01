package v1

import (
	agentSvc "github.com/AgentsMesh/AgentsMesh/backend/internal/service/agent"
)

// MessageHandler handles agent message API endpoints
type MessageHandler struct {
	msgSvc *agentSvc.MessageService
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(msgSvc *agentSvc.MessageService) *MessageHandler {
	return &MessageHandler{
		msgSvc: msgSvc,
	}
}

// AgentSendMessageRequest represents a request to send an agent message
type AgentSendMessageRequest struct {
	ReceiverPod   string                 `json:"receiver_pod" binding:"required"`
	MessageType   string                 `json:"message_type" binding:"required"`
	Content       map[string]interface{} `json:"content" binding:"required"`
	CorrelationID *string                `json:"correlation_id,omitempty"`
	ReplyToID     *int64                 `json:"reply_to_id,omitempty"`
}

// MarkReadRequest represents a request to mark messages as read
type MarkReadRequest struct {
	MessageIDs []int64 `json:"message_ids" binding:"required"`
}
